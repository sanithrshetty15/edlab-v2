#pragma once

#include <atomic>
#include <cstdint>

#include "api_client.h"
#include "enrollment_store.h"
#include "feedback.h"
#include "fingerprint_reader.h"

namespace attendance {

class EnrollmentManager {
 public:
  void begin(ApiClient* api, FingerprintReader* reader, FeedbackController* feedback,
             EnrollmentStore* store);

  bool busy() const { return busy_.load(); }
  bool recoveryPending() const { return recoveryPending_.load(); }
  bool startFromCommand(const EnrollmentInfo& command);
  void update();

 private:
  enum class Phase : uint8_t {
    Idle,
    RecoverQuery,
    Claim,
    CheckSlot,
    BeginProgress,
    WaitFirst,
    ConvertFirst,
    WaitRemoval,
    WaitSecond,
    ConvertSecond,
    RegModel,
    CheckDuplicate,
    Store,
    Complete,
    Fail,
    DeleteTemplate,
    DeleteComplete,
    CleanupLocal,
    CleanupAck,
  };

  void startRecovery();
  void updateRecoveryQuery();
  void updateClaim();
  void updateCheckSlot();
  void updateBeginProgress();
  void updateWaitFirst();
  void updateConvertFirst();
  void updateWaitRemoval();
  void updateWaitSecond();
  void updateConvertSecond();
  void updateRegModel();
  void updateDuplicateCheck();
  void updateStore();
  void updateComplete();
  void updateFail();
  void updateDeleteTemplate();
  void updateDeleteComplete();
  void updateCleanupLocal();
  void updateCleanupAck();

  void abortWithFailure(const char* code, const char* message);
  void finish(bool success, bool audible = true);
  void schedule(uint32_t delayMs);
  bool transientStatus(int status) const;
  bool deleteLocalSlot();
  bool saveRecovery();
  bool loadRecovery();

  ApiClient* api_ = nullptr;
  FingerprintReader* reader_ = nullptr;
  FeedbackController* feedback_ = nullptr;
  EnrollmentStore* store_ = nullptr;

  std::atomic<bool> busy_{false};
  std::atomic<bool> recoveryPending_{false};
  Phase phase_ = Phase::Idle;

  EnrollmentInfo command_;
  EnrollmentRecovery recovery_;
  String failCode_;
  String failMessage_;

  uint32_t nextActionMs_ = 0;
  uint32_t deadlineMs_ = 0;
  uint8_t attempts_ = 0;
  uint8_t readerErrors_ = 0;
  int failSlot_ = 0;
  bool failSent_ = false;
  bool cleanupOnly_ = false;
  bool deleteLocalOnFailure_ = true;
  bool recoveryMode_ = false;
};

}  // namespace attendance
