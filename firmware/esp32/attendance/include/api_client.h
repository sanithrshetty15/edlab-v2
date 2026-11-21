#pragma once

#include <Arduino.h>
#include <ArduinoJson.h>
#include <freertos/FreeRTOS.h>
#include <freertos/semphr.h>
#include <cstddef>
#include <cstdint>

#include "wifi_manager.h"

namespace attendance {

struct AttendanceResult {
  enum class Kind : uint8_t {
    Confirmed,
    Duplicate,
    Rejected,
    Retryable,
  };

  Kind kind = Kind::Retryable;
  String outcome;
  String message;
};

struct HeartbeatInfo {
  const char* firmwareVersion = "";
  const char* readerModel = "";
  bool readerConnected = false;
  const char* readerReadiness = "not_ready";
  int readerUart = 0;
  size_t queueDepth = 0;
  const char* networkStatus = "offline";
  int64_t lastSyncEpoch = 0;
  int64_t lastCaptureEpoch = 0;
};

struct EnrollmentInfo {
  bool valid = false;
  String action;
  String uid;
  String status;
  String memberName;
  String memberLoginId;
  String errorCode;
  String errorMessage;
  int slot = 0;
  bool active = false;
  bool cleanupRequired = false;
};

struct BatchItemResult {
  char idempotencyKey[80] = {0};
  bool success = false;
  bool retryable = true;
  String message;
};

struct BatchResult {
  uint8_t count = 0;
  BatchItemResult items[20];
};

class ApiClient {
 public:
  void begin(WifiManager* wifi);

  bool ensureAuthenticated();
  void forgetToken();
  bool authenticated() const;

  AttendanceResult submitAttendance(uint16_t slot, const char* idempotencyKey,
                                    const char* capturedAt);
  bool submitNoMatch(const char* idempotencyKey, const char* capturedAt);

  bool syncBatch(const String& requestBody, BatchResult& result, int& httpStatus,
                 String& errorMessage);
  bool sendHeartbeat(const HeartbeatInfo& info);

  bool pollEnrollment(EnrollmentInfo& info, int& httpStatus);
  bool getEnrollment(const char* uid, EnrollmentInfo& info, int& httpStatus);
  bool claimEnrollment(const char* uid, int& httpStatus);
  bool reportProgress(const char* uid, const char* status, int slot, const char* errorCode,
                      const char* errorMessage, int& httpStatus);
  bool completeEnrollment(const char* uid, int slot, const char* deviceUserId, int& httpStatus);
  bool failEnrollment(const char* uid, int slot, const char* errorCode, const char* errorMessage,
                      int& httpStatus);
  bool acknowledgeCleanup(const char* uid, bool success, const char* errorCode,
                          const char* errorMessage, int& httpStatus);

 private:
  bool authenticateLocked();
  bool requestLocked(const char* method, const String& path, const String& body, String& response,
                     int& httpStatus, int timeoutMs, bool retryAuth);
  bool performHttpLocked(const char* method, const String& path, const String& body,
                         const char* token, String& response, int& httpStatus, int timeoutMs);
  bool parseEnrollmentResponse(const String& response, EnrollmentInfo& info) const;
  bool parseEnrollmentObject(JsonVariantConst value, EnrollmentInfo& info) const;

  WifiManager* wifi_ = nullptr;
  String baseUrl_;
  String token_;
  int64_t deviceId_ = 0;
  uint32_t lastAuthAttemptMs_ = 0;
  uint8_t authAttempt_ = 0;
  SemaphoreHandle_t mutex_ = nullptr;
};

}  // namespace attendance
