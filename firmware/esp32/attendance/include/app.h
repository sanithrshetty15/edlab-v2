#pragma once

#include <Arduino.h>
#include <freertos/FreeRTOS.h>
#include <freertos/semphr.h>
#include <freertos/task.h>
#include <atomic>
#include <cstdint>
#include <freertos/semphr.h>

#include "api_client.h"
#include "attendance_logic.h"
#include "enrollment_manager.h"
#include "enrollment_store.h"
#include "event_id.h"
#include "feedback.h"
#include "fingerprint_reader.h"
#include "offline_queue.h"
#include "wifi_manager.h"

namespace attendance {

class App {
 public:
  void setup();
  void loop();

 private:
  static void networkTaskEntry(void* context);
  void networkLoop();
  bool createNetworkTask();

  void transitionTo(TerminalState next);
  void updateReadiness();
  const char* readinessName() const;
  const char* readerReadinessName() const;

  bool validateConfiguration();
  void updateBootState();
  void updateScan();
  void submitMatchedSlot(uint16_t slot);
  void reportUnknownFingerprint();
  void beginRemovalWait();
  void updateRemovalWait();
  void handleReaderFailure();

  void publishEnrollmentCommand(const EnrollmentInfo& command);
  bool takeEnrollmentCommand(EnrollmentInfo& command);
  void handleEnrollmentState();

  void flushOfflineQueue();
  void sendHeartbeat();

  WifiManager wifi_;
  ApiClient api_;
  FingerprintReader reader_;
  FeedbackController feedback_;
  OfflineQueue queue_;
  EnrollmentStore enrollmentStore_;
  EnrollmentManager enrollment_;
  EventIdGenerator eventIds_;

  TerminalState state_ = TerminalState::Boot;
  bool networkEnabled_ = false;
  bool networkTaskStarted_ = false;
  TaskHandle_t networkTask_ = nullptr;
  SemaphoreHandle_t commandMutex_ = nullptr;
  EnrollmentInfo pendingCommand_;
  std::atomic<bool> commandReady_{false};
  std::atomic<bool> commandInFlight_{false};

  std::atomic<bool> networkOnline_{false};
  std::atomic<bool> authenticated_{false};
  std::atomic<bool> readerReady_{false};
  std::atomic<uint8_t> readiness_{0};
  std::atomic<uint32_t> lastSyncEpoch_{0};
  std::atomic<uint32_t> lastCaptureEpoch_{0};

  uint32_t bootAtMs_ = 0;
  uint32_t lastReaderRecoveryMs_ = 0;
  uint32_t lastScanPollMs_ = 0;
  uint32_t lastScanResultMs_ = 0;
  uint32_t removalDeadlineMs_ = 0;
  bool awaitingRemoval_ = false;
  bool lastOnline_ = false;
  bool readyCuePlayed_ = false;

  uint32_t lastHeartbeatMs_ = 0;
  uint32_t lastNetworkStartAttemptMs_ = 0;
  uint32_t nextEnrollmentPollMs_ = 0;
  uint32_t nextQueueFlushMs_ = 0;
  uint8_t queueFailures_ = 0;
  uint8_t queueBatchSize_ = 20;
};

}  // namespace attendance
