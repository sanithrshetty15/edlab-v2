#include "enrollment_manager.h"

#include <Arduino.h>
#include <cstring>

#include "attendance_logic.h"
#include "config.h"
#include "logger.h"

namespace attendance {
namespace {
constexpr uint8_t kErrNoFinger = 0x02;
constexpr uint8_t kErrNotFound = 0x09;
constexpr uint8_t kNoConfirmation = 0xFF;
constexpr uint8_t kMaxClaimAttempts = 10;
constexpr uint8_t kMaxFailureAttempts = 5;
constexpr uint8_t kMaxCleanupAttempts = 5;
}  // namespace

void EnrollmentManager::begin(ApiClient* api, FingerprintReader* reader,
                              FeedbackController* feedback, EnrollmentStore* store) {
  api_ = api;
  reader_ = reader;
  feedback_ = feedback;
  store_ = store;
  busy_.store(false);
  phase_ = Phase::Idle;
  if (store_ != nullptr && store_->begin()) {
    recoveryPending_.store(loadRecovery());
  }
}

bool EnrollmentManager::loadRecovery() {
  if (store_ == nullptr) {
    return false;
  }
  const bool found = store_->load(recovery_);
  if (found) {
    ATT_LOG_W("enroll", "pending enrollment recovery for slot %d", recovery_.slot);
  }
  return found;
}

bool EnrollmentManager::saveRecovery() {
  if (store_ == nullptr) {
    return false;
  }
  const bool saved = store_->save(command_.uid.c_str(), command_.slot);
  if (saved) {
    recovery_.present = true;
    recovery_.slot = command_.slot;
    recovery_.templateStored = false;
    strncpy(recovery_.uid, command_.uid.c_str(), sizeof(recovery_.uid) - 1);
  }
  return saved;
}

bool EnrollmentManager::transientStatus(int status) const {
  return status <= 0 || status == 408 || status == 425 || status == 429 || status >= 500;
}

void EnrollmentManager::schedule(uint32_t delayMs) { nextActionMs_ = millis() + delayMs; }

void EnrollmentManager::startRecovery() {
  if (busy_.load() || recovery_.uid[0] == '\0') {
    recoveryPending_.store(false);
    return;
  }
  command_ = EnrollmentInfo();
  command_.uid = recovery_.uid;
  command_.slot = recovery_.slot;
  recoveryMode_ = true;
  cleanupOnly_ = false;
  deleteLocalOnFailure_ = true;
  busy_.store(true);
  if (feedback_ != nullptr) {
    feedback_->setEnrollmentIndicator(true);
    feedback_->play(FeedbackCue::Enrollment);
  }
  phase_ = Phase::RecoverQuery;
  deadlineMs_ = millis() + config::kEnrollmentFingerTimeoutMs;
  ATT_LOG_I("enroll", "checking pending enrollment %s", recovery_.uid);
}

bool EnrollmentManager::startFromCommand(const EnrollmentInfo& command) {
  if (busy_.load() || recoveryPending_.load()) {
    return false;
  }
  if (command.uid.length() == 0 || command.action.length() == 0) {
    return false;
  }
  if (command.action != "enroll" && command.action != "delete" && command.action != "cleanup") {
    ATT_LOG_W("enroll", "ignoring unsupported command action");
    return false;
  }
  if ((command.action == "enroll" || command.action == "delete") &&
      !isValidSlot(command.slot)) {
    ATT_LOG_E("enroll", "command has invalid slot %d", command.slot);
    return false;
  }
  if (command.action == "cleanup" && !isValidSlot(command.slot)) {
    ATT_LOG_E("enroll", "cleanup command has no slot");
    return false;
  }

  command_ = command;
  recoveryMode_ = false;
  failSent_ = false;
  cleanupOnly_ = false;
  deleteLocalOnFailure_ = true;
  failSlot_ = command.slot;
  attempts_ = 0;
  readerErrors_ = 0;
  busy_.store(true);
  if (feedback_ != nullptr) {
    feedback_->setEnrollmentIndicator(true);
    feedback_->play(FeedbackCue::Enrollment);
  }
  if (command.action == "cleanup") {
    phase_ = Phase::CleanupLocal;
    cleanupOnly_ = true;
  } else {
    phase_ = Phase::Claim;
  }
  nextActionMs_ = 0;
  ATT_LOG_I("enroll", "started %s for slot %d", command.action.c_str(), command.slot);
  return true;
}

void EnrollmentManager::update() {
  if (!busy_.load()) {
    if (recoveryPending_.load() && api_ != nullptr && api_->authenticated() &&
        timeReached(millis(), nextActionMs_)) {
      startRecovery();
    }
    return;
  }
  if (!timeReached(millis(), nextActionMs_)) {
    return;
  }

  switch (phase_) {
    case Phase::Idle:
      break;
    case Phase::RecoverQuery:
      updateRecoveryQuery();
      break;
    case Phase::Claim:
      updateClaim();
      break;
    case Phase::CheckSlot:
      updateCheckSlot();
      break;
    case Phase::BeginProgress:
      updateBeginProgress();
      break;
    case Phase::WaitFirst:
      updateWaitFirst();
      break;
    case Phase::ConvertFirst:
      updateConvertFirst();
      break;
    case Phase::WaitRemoval:
      updateWaitRemoval();
      break;
    case Phase::WaitSecond:
      updateWaitSecond();
      break;
    case Phase::ConvertSecond:
      updateConvertSecond();
      break;
    case Phase::RegModel:
      updateRegModel();
      break;
    case Phase::CheckDuplicate:
      updateDuplicateCheck();
      break;
    case Phase::Store:
      updateStore();
      break;
    case Phase::Complete:
      updateComplete();
      break;
    case Phase::Fail:
      updateFail();
      break;
    case Phase::DeleteTemplate:
      updateDeleteTemplate();
      break;
    case Phase::DeleteComplete:
      updateDeleteComplete();
      break;
    case Phase::CleanupLocal:
      updateCleanupLocal();
      break;
    case Phase::CleanupAck:
      updateCleanupAck();
      break;
  }
}

void EnrollmentManager::updateRecoveryQuery() {
  EnrollmentInfo info;
  int status = 0;
  if (!api_->getEnrollment(recovery_.uid, info, status)) {
    if (status == 404) {
      ATT_LOG_W("enroll", "recovery session no longer exists; removing local recovery state");
      if (recovery_.templateStored && !deleteLocalSlot()) {
        ATT_LOG_E("enroll", "could not remove orphan template; keeping recovery state");
        recoveryPending_.store(true);
        finish(false, false);
        return;
      }
      store_->clear();
      recoveryPending_.store(false);
      finish(true);
      return;
    }
    if (transientStatus(status)) {
      recoveryPending_.store(true);
      finish(false, false);
      schedule(5000);
      return;
    }
    finish(false);
    recoveryPending_.store(true);
    return;
  }

  if (info.action == "cleanup") {
    command_.action = "cleanup";
    command_.slot = info.slot > 0 ? info.slot : recovery_.slot;
    cleanupOnly_ = true;
    phase_ = Phase::CleanupLocal;
    return;
  }
  if (info.action == "completed" || info.status == "completed") {
    store_->clear();
    recoveryPending_.store(false);
    finish(true);
    return;
  }
  if (info.action == "enroll") {
    command_.action = "enroll";
    command_.slot = recovery_.slot;
    command_.memberLoginId = info.memberLoginId;
    if (recovery_.templateStored) {
      ATT_LOG_I("enroll", "template already stored; retrying backend completion");
      phase_ = Phase::Complete;
      attempts_ = 0;
      nextActionMs_ = 0;
    } else {
      failSlot_ = command_.slot;
      abortWithFailure("power_loss_during_enrollment",
                       "The terminal restarted before the template was stored");
    }
    return;
  }
  if (info.action == "delete") {
    command_.action = "delete";
    command_.slot = recovery_.slot;
    failSlot_ = command_.slot;
    phase_ = Phase::Claim;
    return;
  }

  // No active command and no cleanup: never clear the recovery record unless
  // an untracked local template has actually been removed.
  if (recovery_.templateStored && !deleteLocalSlot()) {
    ATT_LOG_E("enroll", "could not remove untracked template; keeping recovery state");
    recoveryPending_.store(true);
    finish(false, false);
    return;
  }
  store_->clear();
  recoveryPending_.store(false);
  finish(true);
}

void EnrollmentManager::updateClaim() {
  int status = 0;
  if (api_->claimEnrollment(command_.uid.c_str(), status)) {
    ATT_LOG_I("enroll", "claimed session %s", command_.uid.c_str());
    if (command_.action == "delete") {
      if (!saveRecovery()) {
        // Do not delete the template without a recovery record: a power loss
        // between delete and backend completion would leave a stale mapping.
        deleteLocalOnFailure_ = false;
        failSlot_ = 0;
        abortWithFailure("storage_unavailable", "Could not persist enrollment recovery state");
        return;
      }
      phase_ = Phase::DeleteTemplate;
    } else {
      phase_ = Phase::CheckSlot;
    }
    return;
  }

  if (transientStatus(status)) {
    ++attempts_;
    if (attempts_ <= kMaxClaimAttempts) {
      schedule(2000);
      return;
    }
    ATT_LOG_W("enroll", "claim retries exhausted");
    recoveryPending_.store(recoveryMode_);
    finish(false, false);
    return;
  }

  ATT_LOG_W("enroll", "claim rejected (HTTP %d)", status);
  if (status == 409 || status == 404) {
    if (command_.action == "delete") {
      // A delete session that expired before completion must not leave the
      // local template active. The backend mapping is stale; an admin can
      // re-issue removal, but the fingerprint itself is gone.
      const bool removed = deleteLocalSlot();
      if (!removed) {
        ATT_LOG_E("enroll", "delete session expired; local template removal failed");
      }
      if (recoveryMode_) {
        store_->clear();
        recoveryPending_.store(false);
      }
      finish(false);
      return;
    }
    if (recovery_.present && recovery_.templateStored) {
      cleanupOnly_ = true;
      phase_ = Phase::CleanupLocal;
      return;
    }
    if (recoveryMode_) {
      store_->clear();
      recoveryPending_.store(false);
    }
    finish(false);
    return;
  }
  finish(false);
}

void EnrollmentManager::updateCheckSlot() {
  bool occupied = false;
  if (!reader_->templateExists(static_cast<uint16_t>(command_.slot), occupied)) {
    failSlot_ = 0;
    deleteLocalOnFailure_ = false;
    abortWithFailure("reader_unavailable", "Could not inspect the allocated reader slot");
    return;
  }
  if (occupied) {
    // Never ask the backend to clean a slot that already belonged to someone
    // else. Omitting the slot keeps cleanup_required false, and the local
    // deletion step must also skip this pre-existing template.
    failSlot_ = 0;
    deleteLocalOnFailure_ = false;
    abortWithFailure("slot_occupied", "The allocated slot already contains a template");
    return;
  }
  if (!saveRecovery()) {
    failSlot_ = 0;
    abortWithFailure("storage_unavailable", "Could not persist enrollment recovery state");
    return;
  }
  phase_ = Phase::BeginProgress;
  nextActionMs_ = 0;
}

void EnrollmentManager::updateBeginProgress() {
  int status = 0;
  if (api_->reportProgress(command_.uid.c_str(), "waiting_for_finger", command_.slot, "", "",
                           status)) {
    deadlineMs_ = millis() + config::kEnrollmentFingerTimeoutMs;
    nextActionMs_ = 0;
    phase_ = Phase::WaitFirst;
    return;
  }
  if (status == 409) {
    cleanupOnly_ = true;
    phase_ = Phase::CleanupLocal;
    return;
  }
  ++attempts_;
  if (attempts_ <= 3) {
    schedule(1500);
    return;
  }
  failSlot_ = command_.slot;
  abortWithFailure("progress_failed", "Could not report enrollment progress");
}

void EnrollmentManager::updateWaitFirst() {
  if (timeReached(millis(), deadlineMs_)) {
    failSlot_ = command_.slot;
    abortWithFailure("no_finger", "No finger detected for the first scan");
    return;
  }
  if (!timeReached(millis(), nextActionMs_)) {
    return;
  }
  schedule(100);
  if (reader_->getImage()) {
    readerErrors_ = 0;
    phase_ = Phase::ConvertFirst;
    return;
  }
  if (reader_->lastConfirmation() == kErrNoFinger) {
    return;
  }
  ++readerErrors_;
  if (readerErrors_ >= config::kReaderMaxConsecutiveErrors) {
    failSlot_ = command_.slot;
    abortWithFailure("reader_error", "Fingerprint reader stopped responding");
  }
}

void EnrollmentManager::updateConvertFirst() {
  if (!reader_->image2Tz(1)) {
    failSlot_ = command_.slot;
    abortWithFailure("capture_failed", "First fingerprint image could not be used");
    return;
  }
  int status = 0;
  if (api_->reportProgress(command_.uid.c_str(), "finger_detected", command_.slot, "", "",
                           status)) {
    feedback_->play(FeedbackCue::FingerDetected);
    deadlineMs_ = millis() + config::kEnrollmentRemovalTimeoutMs;
    nextActionMs_ = 0;
    phase_ = Phase::WaitRemoval;
    return;
  }
  if (status == 409) {
    cleanupOnly_ = true;
    phase_ = Phase::CleanupLocal;
    return;
  }
  ++attempts_;
  if (attempts_ <= 3) {
    schedule(1500);
    return;
  }
  failSlot_ = command_.slot;
  abortWithFailure("progress_failed", "Could not report the first scan");
}

void EnrollmentManager::updateWaitRemoval() {
  if (timeReached(millis(), deadlineMs_)) {
    failSlot_ = command_.slot;
    abortWithFailure("finger_not_removed", "Finger was not removed in time");
    return;
  }
  if (!timeReached(millis(), nextActionMs_)) {
    return;
  }
  schedule(120);
  if (!reader_->getImage()) {
    if (reader_->lastConfirmation() == kErrNoFinger) {
      int status = 0;
      api_->reportProgress(command_.uid.c_str(), "waiting_for_finger", command_.slot, "", "",
                           status);
      deadlineMs_ = millis() + config::kEnrollmentFingerTimeoutMs;
      nextActionMs_ = 0;
      phase_ = Phase::WaitSecond;
      return;
    }
    if (reader_->lastConfirmation() != kNoConfirmation) {
      ++readerErrors_;
      if (readerErrors_ >= config::kReaderMaxConsecutiveErrors) {
        failSlot_ = command_.slot;
        abortWithFailure("reader_error", "Fingerprint reader stopped responding");
      }
    }
    return;
  }
  // Finger is still present; the captured frame is intentionally discarded.
}

void EnrollmentManager::updateWaitSecond() {
  if (timeReached(millis(), deadlineMs_)) {
    failSlot_ = command_.slot;
    abortWithFailure("no_finger", "No finger detected for the second scan");
    return;
  }
  if (!timeReached(millis(), nextActionMs_)) {
    return;
  }
  schedule(100);
  if (reader_->getImage()) {
    readerErrors_ = 0;
    phase_ = Phase::ConvertSecond;
    return;
  }
  if (reader_->lastConfirmation() == kErrNoFinger) {
    return;
  }
  ++readerErrors_;
  if (readerErrors_ >= config::kReaderMaxConsecutiveErrors) {
    failSlot_ = command_.slot;
    abortWithFailure("reader_error", "Fingerprint reader stopped responding");
  }
}

void EnrollmentManager::updateConvertSecond() {
  if (!reader_->image2Tz(2)) {
    failSlot_ = command_.slot;
    abortWithFailure("capture_failed", "Second fingerprint image could not be used");
    return;
  }
  int status = 0;
  api_->reportProgress(command_.uid.c_str(), "captured", command_.slot, "", "", status);
  if (status == 409) {
    cleanupOnly_ = true;
    phase_ = Phase::CleanupLocal;
    return;
  }
  phase_ = Phase::RegModel;
  nextActionMs_ = 0;
}

void EnrollmentManager::updateRegModel() {
  if (!reader_->regModel()) {
    failSlot_ = command_.slot;
    abortWithFailure("samples_did_not_match", "The two scans did not produce a usable template");
    return;
  }
  readerErrors_ = 0;
  phase_ = Phase::CheckDuplicate;
  nextActionMs_ = 0;
}

void EnrollmentManager::updateDuplicateCheck() {
  // Buffer 1 still holds the merged template produced by RegModel. Searching
  // the stored library is the same proven sequence used by the door-lock
  // firmware and detects an existing enrollment before anything is stored.
  uint16_t existingSlot = 0;
  if (reader_->searchRange(static_cast<uint16_t>(kFingerprintSlotMin),
                           static_cast<uint16_t>(kFingerprintCapacity), existingSlot)) {
    failSlot_ = command_.slot;
    abortWithFailure("duplicate_fingerprint",
                     "This finger is already enrolled in another slot on this terminal");
    return;
  }
  if (reader_->lastConfirmation() == kErrNotFound) {
    phase_ = Phase::Store;
    nextActionMs_ = 0;
    return;
  }
  ++readerErrors_;
  if (readerErrors_ >= config::kReaderMaxConsecutiveErrors) {
    failSlot_ = command_.slot;
    abortWithFailure("reader_error", "Could not check for duplicate fingerprints");
    return;
  }
  schedule(50);
}

void EnrollmentManager::updateStore() {
  if (!reader_->storeModel(static_cast<uint16_t>(command_.slot))) {
    failSlot_ = command_.slot;
    abortWithFailure("template_store_failed", "The reader could not store the template");
    return;
  }
  if (store_ != nullptr && !store_->markTemplateStored()) {
    ATT_LOG_W("enroll", "template stored but recovery flag could not be persisted");
  }
  recovery_.templateStored = true;
  attempts_ = 0;
  phase_ = Phase::Complete;
  nextActionMs_ = 0;
}

void EnrollmentManager::updateComplete() {
  int status = 0;
  if (api_->completeEnrollment(command_.uid.c_str(), command_.slot, "", status)) {
    if (store_ != nullptr) {
      store_->clear();
    }
    recoveryPending_.store(false);
    ATT_LOG_I("enroll", "enrollment completed in slot %d", command_.slot);
    finish(true);
    return;
  }
  if (status == 409 || status == 422) {
    ATT_LOG_W("enroll", "backend rejected completion (HTTP %d); removing local template", status);
    cleanupOnly_ = true;
    phase_ = Phase::CleanupLocal;
    return;
  }
  ++attempts_;
  if (attempts_ < config::kEnrollmentCompleteAttempts) {
    schedule(1000UL * attempts_);
    return;
  }
  failSlot_ = command_.slot;
  abortWithFailure("backend_finalize_failed",
                   "Template stored but the backend did not confirm the mapping");
}

void EnrollmentManager::updateFail() {
  if (!failSent_) {
    int status = 0;
    if (api_->failEnrollment(command_.uid.c_str(), failSlot_, failCode_.c_str(),
                             failMessage_.c_str(), status)) {
      failSent_ = true;
      attempts_ = 0;
      phase_ = Phase::CleanupLocal;
      nextActionMs_ = 0;
      return;
    }
    if (status == 409 || status == 404) {
      failSent_ = true;
      attempts_ = 0;
      phase_ = Phase::CleanupLocal;
      nextActionMs_ = 0;
      return;
    }
    ++attempts_;
    if (attempts_ <= kMaxFailureAttempts) {
      schedule(1500);
      return;
    }
    ATT_LOG_E("enroll", "failure report could not be delivered; keeping recovery state");
    recoveryPending_.store(recovery_.present);
    finish(false, false);
    return;
  }
  phase_ = Phase::CleanupLocal;
  nextActionMs_ = 0;
}

void EnrollmentManager::updateDeleteTemplate() {
  if (!deleteLocalSlot()) {
    failSlot_ = command_.slot;
    abortWithFailure("template_delete_failed", "The reader could not delete the fingerprint");
    return;
  }
  phase_ = Phase::DeleteComplete;
  nextActionMs_ = 0;
}

void EnrollmentManager::updateDeleteComplete() {
  int status = 0;
  if (api_->completeEnrollment(command_.uid.c_str(), command_.slot, "", status)) {
    store_->clear();
    recoveryPending_.store(false);
    ATT_LOG_I("enroll", "fingerprint removed from slot %d", command_.slot);
    finish(true);
    return;
  }
  ++attempts_;
  if (attempts_ < config::kEnrollmentCompleteAttempts) {
    schedule(1000UL * attempts_);
    return;
  }
  ATT_LOG_W("enroll", "template deleted; backend completion will be retried");
  recoveryPending_.store(true);
  finish(false, false);
}

void EnrollmentManager::updateCleanupLocal() {
  if (!cleanupOnly_ && !deleteLocalOnFailure_) {
    // A failure before any local template operation must never erase a slot
    // that may belong to another member.
    phase_ = Phase::CleanupAck;
    attempts_ = 0;
    nextActionMs_ = 0;
    return;
  }
  if (!deleteLocalSlot()) {
    ++attempts_;
    if (attempts_ < 3) {
      schedule(1000);
      return;
    }
    int status = 0;
    api_->acknowledgeCleanup(command_.uid.c_str(), false, "template_delete_failed",
                             "The reader could not delete the local template", status);
    recoveryPending_.store(recovery_.present);
    finish(false);
    return;
  }
  phase_ = Phase::CleanupAck;
  attempts_ = 0;
  nextActionMs_ = 0;
}

void EnrollmentManager::updateCleanupAck() {
  int status = 0;
  if (api_->acknowledgeCleanup(command_.uid.c_str(), true, "", "", status)) {
    if (store_ != nullptr) {
      store_->clear();
    }
    recoveryPending_.store(false);
    finish(cleanupOnly_);
    return;
  }
  if (status == 409 || status == 404) {
    if (store_ != nullptr) {
      store_->clear();
    }
    recoveryPending_.store(false);
    finish(cleanupOnly_);
    return;
  }
  ++attempts_;
  if (attempts_ < kMaxCleanupAttempts) {
    schedule(1500);
    return;
  }
  recoveryPending_.store(recovery_.present);
  finish(false);
}

bool EnrollmentManager::deleteLocalSlot() {
  if (!isValidSlot(command_.slot)) {
    return false;
  }
  return reader_->deleteModel(static_cast<uint16_t>(command_.slot));
}

void EnrollmentManager::abortWithFailure(const char* code, const char* message) {
  failCode_ = code;
  failMessage_ = message;
  failSent_ = false;
  attempts_ = 0;
  phase_ = Phase::Fail;
  nextActionMs_ = 0;
  ATT_LOG_W("enroll", "%s: %s", code, message);
}

void EnrollmentManager::finish(bool success, bool audible) {
  busy_.store(false);
  phase_ = Phase::Idle;
  command_ = EnrollmentInfo();
  failCode_ = "";
  failMessage_ = "";
  failSent_ = false;
  cleanupOnly_ = false;
  deleteLocalOnFailure_ = true;
  recoveryMode_ = false;
  attempts_ = 0;
  readerErrors_ = 0;
  if (feedback_ != nullptr) {
    feedback_->setEnrollmentIndicator(false);
    if (audible) {
      feedback_->play(success ? FeedbackCue::FingerMatched : FeedbackCue::Error);
    }
  }
  if (!recoveryPending_.load()) {
    nextActionMs_ = 0;
  } else {
    nextActionMs_ = millis() + 5000;
  }
}

}  // namespace attendance
