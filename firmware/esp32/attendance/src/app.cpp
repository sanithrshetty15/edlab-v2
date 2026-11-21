#include "app.h"

#include <ArduinoJson.h>
#include <cstring>

#include "config.h"
#include "hardware_config.h"
#include "logger.h"
#include "time_utils.h"

namespace attendance {
namespace {
constexpr const char* kTag = "app";
constexpr uint8_t kErrNoFinger = 0x02;
constexpr uint8_t kErrNotFound = 0x09;
constexpr uint8_t kNoConfirmation = 0xFF;

struct BatchEntry {
  uint32_t sequence = 0;
  char key[80] = {0};
};

const char* statusName(AttendanceResult::Kind kind) {
  switch (kind) {
    case AttendanceResult::Kind::Confirmed:
      return "confirmed";
    case AttendanceResult::Kind::Duplicate:
      return "duplicate";
    case AttendanceResult::Kind::Rejected:
      return "rejected";
    case AttendanceResult::Kind::Retryable:
      return "retryable";
  }
  return "unknown";
}

void lock(SemaphoreHandle_t mutex) {
  if (mutex != nullptr) {
    xSemaphoreTake(mutex, portMAX_DELAY);
  }
}

void releaseLock(SemaphoreHandle_t mutex) {
  if (mutex != nullptr) {
    xSemaphoreGive(mutex);
  }
}
}  // namespace

void App::setup() {
  Logger::begin(115200);
  ATT_LOG_I(kTag, "Edwins Lab attendance terminal %s", config::kFirmwareVersion);
  ATT_LOG_I(kTag, "hardware: reader UART%d RX%d TX%d @ %lu; buzzer %d (%s); LEDs %s",
            hardware::kReaderUart, hardware::kReaderRxPin, hardware::kReaderTxPin,
            static_cast<unsigned long>(hardware::kReaderBaud), hardware::kBuzzerPin,
            hardware::kBuzzerActiveHigh ? "active-high" : "active-low",
            hardware::kStatusLedsEnabled ? "enabled" : "disabled (unverified)");

  eventIds_.begin();
  feedback_.begin();
  feedback_.play(FeedbackCue::Boot);
  queue_.begin();
  readerReady_.store(reader_.begin());
  api_.begin(&wifi_);
  commandMutex_ = xSemaphoreCreateMutex();
  enrollment_.begin(&api_, &reader_, &feedback_, &enrollmentStore_);

  transitionTo(TerminalState::Boot);
  transitionTo(TerminalState::HardwareInit);
  networkEnabled_ = validateConfiguration();
  if (networkEnabled_) {
    wifi_.begin();
    transitionTo(TerminalState::WifiConnecting);
    createNetworkTask();
  } else {
    transitionTo(TerminalState::ErrorRecovery);
    feedback_.play(FeedbackCue::Error);
  }

  if (!reader_.ready()) {
    transitionTo(TerminalState::ErrorRecovery);
    feedback_.play(FeedbackCue::Error);
  }
  bootAtMs_ = millis();
  lastReaderRecoveryMs_ = millis();
}

bool App::validateConfiguration() {
  bool valid = true;
  if (strlen(config::kWifiSsid) == 0 || containsPlaceholder(config::kWifiSsid)) {
    ATT_LOG_E(kTag, "Wi-Fi SSID is missing or a placeholder");
    valid = false;
  }
  if (strlen(config::kWifiPassword) == 0 || containsPlaceholder(config::kWifiPassword)) {
    ATT_LOG_E(kTag, "Wi-Fi password is missing or a placeholder");
    valid = false;
  }
  if (strlen(config::kDeviceApiKey) < 32 || containsPlaceholder(config::kDeviceApiKey)) {
    ATT_LOG_E(kTag, "device API key is missing, a placeholder, or too short");
    valid = false;
  }
  if (containsPlaceholder(config::kDeviceName) || strlen(config::kDeviceName) == 0) {
    ATT_LOG_E(kTag, "device name is missing or a placeholder");
    valid = false;
  }
  const String url = config::kApiBaseUrl;
  if (containsPlaceholder(url.c_str())) {
    ATT_LOG_E(kTag, "API base URL is a placeholder");
    valid = false;
  } else if (!url.startsWith("https://")) {
    if (config::kInsecureHttpAllowed) {
      ATT_LOG_W(kTag, "plaintext HTTP API URL explicitly allowed for development");
    } else {
      ATT_LOG_E(kTag, "API base URL must use HTTPS");
      valid = false;
    }
  }
  if (!valid) {
    ATT_LOG_E(kTag, "network disabled until the configuration is corrected");
  }
  return valid;
}

bool App::createNetworkTask() {
  if (networkTaskStarted_) {
    return true;
  }
  const BaseType_t created =
      xTaskCreatePinnedToCore(networkTaskEntry, "attendance-net", config::kNetworkTaskStackBytes,
                              this, config::kNetworkTaskPriority, &networkTask_,
                              config::kNetworkTaskCore);
  if (created != pdPASS) {
    ATT_LOG_E(kTag, "network task creation failed");
    return false;
  }
  networkTaskStarted_ = true;
  ATT_LOG_I(kTag, "network task started on core %u", config::kNetworkTaskCore);
  return true;
}

void App::networkTaskEntry(void* context) {
  static_cast<App*>(context)->networkLoop();
  vTaskDelete(nullptr);
}

void App::networkLoop() {
  bool clockWarningLogged = false;
  for (;;) {
    wifi_.update();
    const bool online = wifi_.connected();
    networkOnline_.store(online);
    timeutil::ensureStarted(online);

    if (online && !timeutil::clockValid()) {
      // TLS certificate date validation needs a real clock. SNTP normally
      // synchronizes within seconds; never bypass verification while waiting.
      authenticated_.store(false);
      if (!clockWarningLogged) {
        clockWarningLogged = true;
        ATT_LOG_W(kTag, "waiting for SNTP before HTTPS authentication");
      }
      vTaskDelay(pdMS_TO_TICKS(200));
      continue;
    }
    if (online) {
      clockWarningLogged = false;
      const bool authenticated = api_.ensureAuthenticated();
      authenticated_.store(authenticated);
      if (authenticated) {
        const uint32_t now = millis();
        if (lastHeartbeatMs_ == 0 || now - lastHeartbeatMs_ >= config::kHeartbeatIntervalMs) {
          lastHeartbeatMs_ = now;
          sendHeartbeat();
        }
        if (timeReached(now, nextQueueFlushMs_)) {
          flushOfflineQueue();
        }
        if (!commandReady_.load() && !commandInFlight_.load() && !enrollment_.busy() &&
            !enrollment_.recoveryPending() && timeReached(now, nextEnrollmentPollMs_)) {
          nextEnrollmentPollMs_ = now + 2000;
          EnrollmentInfo command;
          int status = 0;
          if (api_.pollEnrollment(command, status) && command.valid &&
              command.action != "none") {
            publishEnrollmentCommand(command);
          }
        }
      }
    } else {
      authenticated_.store(false);
    }
    vTaskDelay(pdMS_TO_TICKS(25));
  }
}

void App::sendHeartbeat() {
  HeartbeatInfo info;
  info.firmwareVersion = config::kFirmwareVersion;
  info.readerModel = hardware::kReaderModel;
  info.readerConnected = readerReady_.load();
  info.readerReadiness = readerReadinessName();
  info.readerUart = hardware::kReaderUart;
  info.queueDepth = queue_.count();
  info.networkStatus = networkOnline_.load() ? "online" : "offline";
  info.lastSyncEpoch = static_cast<int64_t>(lastSyncEpoch_.load());
  info.lastCaptureEpoch = static_cast<int64_t>(lastCaptureEpoch_.load());
  if (!api_.sendHeartbeat(info)) {
    ATT_LOG_W(kTag, "heartbeat failed; next attempt in %lu ms",
              static_cast<unsigned long>(config::kHeartbeatIntervalMs));
  }
}

void App::flushOfflineQueue() {
  if (queue_.count() == 0) {
    queueFailures_ = 0;
    queueBatchSize_ = config::kQueueBatchLimit;
    nextQueueFlushMs_ = millis() + config::kQueueFlushIntervalMs;
    return;
  }

  QueueRecord records[config::kQueueBatchLimit];
  const size_t available = queue_.snapshot(records, queueBatchSize_);
  if (available == 0) {
    nextQueueFlushMs_ = millis() + config::kQueueFlushIntervalMs;
    return;
  }

  JsonDocument batch;
  JsonArray scans = batch["scans"].to<JsonArray>();
  BatchEntry entries[config::kQueueBatchLimit];
  size_t count = 0;
  for (size_t i = 0; i < available; ++i) {
    JsonDocument event;
    if (deserializeJson(event, records[i].payload)) {
      ATT_LOG_E("queue", "dropping unparseable event sequence %lu",
                static_cast<unsigned long>(records[i].sequence));
      queue_.removeBySequence(records[i].sequence);
      queue_.notePermanentFailure();
      continue;
    }
    const char* key = event["idempotency_key"] | "";
    if (!isValidIdempotencyKey(key)) {
      queue_.removeBySequence(records[i].sequence);
      queue_.notePermanentFailure();
      continue;
    }
    scans.add(event.as<JsonObjectConst>());
    entries[count].sequence = records[i].sequence;
    strncpy(entries[count].key, key, sizeof(entries[count].key) - 1);
    ++count;
  }
  if (count == 0) {
    nextQueueFlushMs_ = millis() + config::kQueueFlushIntervalMs;
    return;
  }

  String body;
  serializeJson(batch, body);
  if (body.length() > 6000) {
    if (queueBatchSize_ > 1) {
      queueBatchSize_ = static_cast<uint8_t>(queueBatchSize_ / 2);
    }
    ATT_LOG_W("queue", "batch document too large; reducing batch size to %u",
              static_cast<unsigned>(queueBatchSize_));
    nextQueueFlushMs_ = millis() + 2000;
    return;
  }

  BatchResult result;
  int status = 0;
  String error;
  if (!api_.syncBatch(body, result, status, error)) {
    if (status == 400 || status == 422) {
      if (queueBatchSize_ > 1) {
        queueBatchSize_ = static_cast<uint8_t>(queueBatchSize_ / 2);
      }
      ATT_LOG_W("queue", "backend rejected batch (HTTP %d); batch size now %u", status,
                static_cast<unsigned>(queueBatchSize_));
    } else {
      ATT_LOG_W("queue", "batch sync failed (HTTP %d); queue depth %u", status,
                static_cast<unsigned>(queue_.count()));
    }
    if (queueFailures_ < 20) {
      ++queueFailures_;
    }
    nextQueueFlushMs_ =
        millis() + exponentialBackoffMs(queueFailures_, config::kQueueFlushIntervalMs,
                                        config::kQueueFlushMaxBackoffMs);
    return;
  }

  bool removed = false;
  bool keptRetryable = false;
  for (size_t i = 0; i < count; ++i) {
    const BatchItemResult* match = nullptr;
    for (uint8_t j = 0; j < result.count; ++j) {
      if (strcmp(result.items[j].idempotencyKey, entries[i].key) == 0) {
        match = &result.items[j];
        break;
      }
    }
    if (match == nullptr) {
      keptRetryable = true;
      continue;
    }
    if (match->success) {
      queue_.removeBySequence(entries[i].sequence);
      removed = true;
    } else if (!match->retryable) {
      ATT_LOG_E("queue", "event %s permanently rejected: %s", entries[i].key,
                match->message.c_str());
      queue_.removeBySequence(entries[i].sequence);
      queue_.notePermanentFailure();
      removed = true;
    } else {
      keptRetryable = true;
    }
  }

  if (result.count == 0) {
    if (queueFailures_ < 20) {
      ++queueFailures_;
    }
    nextQueueFlushMs_ =
        millis() + exponentialBackoffMs(queueFailures_, config::kQueueFlushIntervalMs,
                                        config::kQueueFlushMaxBackoffMs);
    ATT_LOG_W("queue", "batch response contained no results; retrying with backoff");
    return;
  }
  if (removed) {
    lastSyncEpoch_.store(static_cast<uint32_t>(timeutil::nowEpoch()));
  }
  if (queue_.count() == 0 && !keptRetryable) {
    queueFailures_ = 0;
    queueBatchSize_ = config::kQueueBatchLimit;
    nextQueueFlushMs_ = millis() + config::kQueueFlushIntervalMs;
  } else {
    queueFailures_ = 0;
    nextQueueFlushMs_ = millis() + 2000;
  }
  ATT_LOG_I("queue", "batch processed: %u confirmed/removed, depth %u", static_cast<unsigned>(count),
            static_cast<unsigned>(queue_.count()));
}

void App::publishEnrollmentCommand(const EnrollmentInfo& command) {
  lock(commandMutex_);
  if (!commandReady_.load()) {
    pendingCommand_ = command;
    commandReady_.store(true);
  }
  releaseLock(commandMutex_);
}

bool App::takeEnrollmentCommand(EnrollmentInfo& command) {
  bool available = false;
  lock(commandMutex_);
  if (commandReady_.load()) {
    command = pendingCommand_;
    pendingCommand_ = EnrollmentInfo();
    commandReady_.store(false);
    commandInFlight_.store(true);
    available = true;
  }
  releaseLock(commandMutex_);
  return available;
}

void App::updateBootState() {
  if (state_ == TerminalState::WifiConnecting &&
      (networkOnline_.load() || millis() - bootAtMs_ > 8000)) {
    transitionTo(TerminalState::BackendConnecting);
  }
  if (state_ == TerminalState::BackendConnecting &&
      (authenticated_.load() || millis() - bootAtMs_ > 12000)) {
    transitionTo(TerminalState::Syncing);
  }
  if (state_ == TerminalState::Syncing) {
    transitionTo(TerminalState::Idle);
  }
}

void App::loop() {
  feedback_.update();

  if (networkEnabled_ && !networkTaskStarted_) {
    if (millis() - lastNetworkStartAttemptMs_ > 5000) {
      lastNetworkStartAttemptMs_ = millis();
      createNetworkTask();
    }
  }
  if (!networkEnabled_) {
    delay(50);
    return;
  }

  updateBootState();

  if (!reader_.ready()) {
    if (millis() - lastReaderRecoveryMs_ >= config::kReaderReinitIntervalMs) {
      lastReaderRecoveryMs_ = millis();
      reader_.end();
      const bool ok = reader_.begin();
      readerReady_.store(ok);
      updateReadiness();
      if (ok) {
        if (state_ == TerminalState::ErrorRecovery) {
          transitionTo(TerminalState::Idle);
        }
        feedback_.play(FeedbackCue::Ready);
      } else {
        if (state_ != TerminalState::ErrorRecovery) {
          transitionTo(TerminalState::ErrorRecovery);
        }
        feedback_.play(FeedbackCue::Error);
      }
    }
    delay(10);
    return;
  }
  readerReady_.store(true);

  if (enrollment_.busy()) {
    if (state_ != TerminalState::EnrollmentMode) {
      transitionTo(TerminalState::EnrollmentMode);
    }
    enrollment_.update();
    delay(1);
    return;
  }

  if (state_ == TerminalState::EnrollmentMode) {
    transitionTo(TerminalState::Idle);
  }

  if (state_ == TerminalState::ErrorRecovery) {
    // A reader recovery above transitions to Idle; other recovery reasons keep
    // the terminal visibly faulted without a reboot loop.
    delay(10);
    return;
  }

  if (state_ != TerminalState::Idle) {
    delay(10);
    return;
  }

  if (enrollment_.recoveryPending()) {
    enrollment_.update();
    if (enrollment_.busy()) {
      transitionTo(TerminalState::EnrollmentMode);
      enrollment_.update();
      delay(1);
      return;
    }
  }

  EnrollmentInfo command;
  if (takeEnrollmentCommand(command)) {
    const bool started = enrollment_.startFromCommand(command);
    commandInFlight_.store(false);
    if (started) {
      transitionTo(TerminalState::EnrollmentMode);
      enrollment_.update();
      delay(1);
      return;
    }
    ATT_LOG_W("enroll", "dropped unusable enrollment command");
  }

  if (!lastOnline_ && networkOnline_.load()) {
    lastOnline_ = true;
  } else if (lastOnline_ && !networkOnline_.load()) {
    lastOnline_ = false;
    feedback_.play(FeedbackCue::WifiLost);
  }

  if (!readyCuePlayed_) {
    readyCuePlayed_ = true;
    feedback_.play(FeedbackCue::Ready);
  }

  if (awaitingRemoval_) {
    updateRemovalWait();
  } else {
    updateScan();
  }
  delay(1);
}

void App::transitionTo(TerminalState next) {
  if (!isTransitionAllowed(state_, next)) {
    ATT_LOG_W(kTag, "unexpected state transition %s -> %s", terminalStateName(state_),
              terminalStateName(next));
  } else if (state_ != next) {
    ATT_LOG_I(kTag, "state %s -> %s", terminalStateName(state_), terminalStateName(next));
  }
  state_ = next;
  updateReadiness();
}

void App::updateReadiness() {
  uint8_t code = 0;
  if (!readerReady_.load()) {
    code = 1;  // initializing
  } else if (state_ == TerminalState::EnrollmentMode) {
    code = 5;  // enrolling
  } else if (state_ == TerminalState::FingerDetected ||
             state_ == TerminalState::FingerMatched ||
             state_ == TerminalState::EventPending ||
             state_ == TerminalState::EventSubmitting ||
             state_ == TerminalState::EventConfirmed) {
    code = 4;  // finger_detected
  } else if (state_ == TerminalState::ErrorRecovery) {
    code = 6;  // capture_failed
  } else if (state_ == TerminalState::Idle) {
    code = 3;  // waiting_for_finger
  } else {
    code = 1;
  }
  readiness_.store(code);
}

const char* App::readinessName() const {
  switch (readiness_.load()) {
    case 1:
      return "initializing";
    case 2:
      return "ready";
    case 3:
      return "waiting_for_finger";
    case 4:
      return "finger_detected";
    case 5:
      return "enrolling";
    case 6:
      return "capture_failed";
    default:
      return "not_ready";
  }
}

const char* App::readerReadinessName() const { return readinessName(); }

void App::beginRemovalWait() {
  awaitingRemoval_ = true;
  removalDeadlineMs_ = millis() + config::kFingerRemovalTimeoutMs;
  lastScanPollMs_ = millis();
}

void App::updateRemovalWait() {
  if (timeReached(millis(), removalDeadlineMs_)) {
    // Do not re-arm while a finger may still be present. Extend the watch
    // window and keep scanning paused; only a real removal re-arms the reader.
    ATT_LOG_W("scan", "finger still present; waiting for removal before re-arming");
    removalDeadlineMs_ = millis() + config::kFingerRemovalTimeoutMs;
    return;
  }
  if (millis() - lastScanPollMs_ < 120) {
    return;
  }
  lastScanPollMs_ = millis();
  if (reader_.getImage()) {
    return;  // Finger still present; discard the frame and keep the reader locked.
  }
  if (reader_.lastConfirmation() == kErrNoFinger) {
    awaitingRemoval_ = false;
    lastScanResultMs_ = millis();
    return;
  }
  if (reader_.lastConfirmation() != kNoConfirmation ||
      reader_.consecutiveErrors() >= config::kReaderMaxConsecutiveErrors) {
    handleReaderFailure();
  }
}

void App::handleReaderFailure() {
  feedback_.play(FeedbackCue::Error);
  if (state_ != TerminalState::Idle) {
    transitionTo(TerminalState::Idle);
  }
  if (reader_.consecutiveErrors() >= config::kReaderMaxConsecutiveErrors) {
    reader_.end();
    readerReady_.store(false);
    lastReaderRecoveryMs_ = millis();
    transitionTo(TerminalState::ErrorRecovery);
  }
  lastScanResultMs_ = millis();
}

void App::updateScan() {
  if (millis() - lastScanPollMs_ < config::kScanPollIntervalMs) {
    return;
  }
  lastScanPollMs_ = millis();
  if (!scanCooldownElapsed(millis(), lastScanResultMs_, config::kScanCooldownMs)) {
    return;
  }

  if (!reader_.getImage()) {
    if (reader_.lastConfirmation() == kErrNoFinger) {
      return;
    }
    handleReaderFailure();
    return;
  }

  transitionTo(TerminalState::FingerDetected);
  feedback_.play(FeedbackCue::FingerDetected);
  if (!reader_.image2Tz(1)) {
    handleReaderFailure();
    beginRemovalWait();
    return;
  }

  uint16_t slot = 0;
  if (reader_.search(slot)) {
    transitionTo(TerminalState::FingerMatched);
    feedback_.play(FeedbackCue::FingerMatched);
    submitMatchedSlot(slot);
    return;
  }
  if (reader_.lastConfirmation() == kErrNotFound) {
    reportUnknownFingerprint();
    return;
  }
  handleReaderFailure();
  beginRemovalWait();
}

void App::submitMatchedSlot(uint16_t slot) {
  const int64_t captureEpoch = timeutil::nowEpoch();
  if (captureEpoch > 0) {
    lastCaptureEpoch_.store(static_cast<uint32_t>(captureEpoch));
  }
  char capturedAt[24] = {0};
  if (captureEpoch > 0) {
    timeutil::formatIso8601(captureEpoch, capturedAt, sizeof(capturedAt));
  }
  const String key = eventIds_.next();

  transitionTo(TerminalState::EventPending);
  transitionTo(TerminalState::EventSubmitting);
  feedback_.play(FeedbackCue::Sending);
  const AttendanceResult result = api_.submitAttendance(slot, key.c_str(), capturedAt);
  ATT_LOG_I("scan", "slot %u result=%s outcome=%s", static_cast<unsigned>(slot),
            statusName(result.kind), result.outcome.c_str());

  switch (result.kind) {
    case AttendanceResult::Kind::Confirmed:
      transitionTo(TerminalState::EventConfirmed);
      feedback_.play(result.outcome == "exit" ? FeedbackCue::ExitConfirmed
                                               : FeedbackCue::EntryConfirmed);
      break;
    case AttendanceResult::Kind::Duplicate:
      feedback_.play(FeedbackCue::Duplicate);
      transitionTo(TerminalState::Idle);
      break;
    case AttendanceResult::Kind::Rejected: {
      ATT_LOG_W("scan", "attendance rejected: %s", result.message.c_str());
      const bool unknown = result.outcome == "unknown_slot" ||
                           result.outcome == "member_inactive" ||
                           result.outcome == "user_disabled" || result.outcome == "no_match";
      feedback_.play(unknown ? FeedbackCue::UnknownFingerprint : FeedbackCue::Error);
      transitionTo(TerminalState::Idle);
      break;
    }
    case AttendanceResult::Kind::Retryable:
      transitionTo(TerminalState::EventPending);
      if (queue_.push(slot, key.c_str(), capturedAt)) {
        feedback_.play(FeedbackCue::OfflineQueued);
      } else {
        ATT_LOG_E("scan", "offline queue full or unavailable; attendance NOT recorded");
        feedback_.play(FeedbackCue::Error);
      }
      transitionTo(TerminalState::Idle);
      break;
  }
  beginRemovalWait();
}

void App::reportUnknownFingerprint() {
  feedback_.play(FeedbackCue::UnknownFingerprint);
  transitionTo(TerminalState::Idle);
  if (api_.authenticated()) {
    const int64_t captureEpoch = timeutil::nowEpoch();
    char capturedAt[24] = {0};
    if (captureEpoch > 0) {
      timeutil::formatIso8601(captureEpoch, capturedAt, sizeof(capturedAt));
    }
    const String key = eventIds_.next();
    api_.submitNoMatch(key.c_str(), capturedAt);
  }
  beginRemovalWait();
}

}  // namespace attendance
