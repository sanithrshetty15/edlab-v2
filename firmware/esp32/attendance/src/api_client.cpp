#include "api_client.h"

#include <ArduinoJson.h>
#include <HTTPClient.h>
#include <WiFiClient.h>
#include <WiFiClientSecure.h>
#include <cstring>

#include "attendance_logic.h"
#include "config.h"
#include "gts_root_r4.h"
#include "logger.h"
#include "time_utils.h"

namespace attendance {
namespace {
constexpr const char* kTag = "api";
constexpr const char* kAttendancePath = "/api/v1/attendance-terminal/attendance";
constexpr const char* kHeartbeatPath = "/api/v1/attendance-terminal/heartbeat";
constexpr const char* kPollPath = "/api/v1/attendance-terminal/enrollment-session";

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

bool validSessionUid(const char* uid) {
  if (uid == nullptr) {
    return false;
  }
  const size_t length = strlen(uid);
  if (length < 8 || length > 80) {
    return false;
  }
  for (size_t i = 0; i < length; ++i) {
    const char c = uid[i];
    const bool allowed = (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
                         (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == ':';
    if (!allowed) {
      return false;
    }
  }
  return true;
}

String bounded(const char* value, size_t maxLength) {
  if (value == nullptr) {
    return String();
  }
  String result(value);
  if (result.length() > maxLength) {
    result.remove(maxLength);
  }
  return result;
}

String enrollmentPath(const char* uid, const char* suffix = "") {
  String path = "/api/v1/attendance-terminal/enrollment-sessions/";
  path += uid;
  path += suffix;
  return path;
}
}  // namespace

void ApiClient::begin(WifiManager* wifi) {
  wifi_ = wifi;
  baseUrl_ = config::kApiBaseUrl;
  while (baseUrl_.endsWith("/")) {
    baseUrl_.remove(baseUrl_.length() - 1);
  }
  token_ = "";
  deviceId_ = 0;
  if (mutex_ == nullptr) {
    mutex_ = xSemaphoreCreateMutex();
  }
}

bool ApiClient::authenticated() const {
  lock(mutex_);
  const bool value = token_.length() > 0 && deviceId_ > 0;
  releaseLock(mutex_);
  return value;
}

void ApiClient::forgetToken() {
  lock(mutex_);
  token_ = "";
  deviceId_ = 0;
  releaseLock(mutex_);
}

bool ApiClient::ensureAuthenticated() {
  lock(mutex_);
  const bool ok = authenticateLocked();
  releaseLock(mutex_);
  return ok;
}

bool ApiClient::authenticateLocked() {
  if (wifi_ == nullptr || !wifi_->connected()) {
    return false;
  }
  if (token_.length() > 0 && deviceId_ > 0) {
    return true;
  }
  const uint32_t now = millis();
  if (lastAuthAttemptMs_ != 0) {
    const uint32_t backoff = exponentialBackoffMs(authAttempt_, config::kAuthRetryBaseMs,
                                                  config::kAuthRetryMaxMs);
    if (now - lastAuthAttemptMs_ < backoff) {
      return false;
    }
  }
  lastAuthAttemptMs_ = now;

  JsonDocument request;
  request["api_key"] = config::kDeviceApiKey;
  request["device_name"] = config::kDeviceName;
  String body;
  serializeJson(request, body);

  String response;
  int status = 0;
  if (!performHttpLocked("POST", "/api/v1/biometric/authenticate-device", body, nullptr, response,
                         status, config::kHttpTimeoutMs)) {
    if (authAttempt_ < 20) {
      ++authAttempt_;
    }
    ATT_LOG_W(kTag, "device authentication failed (HTTP %d)", status);
    return false;
  }
  if (status < 200 || status >= 300) {
    if (authAttempt_ < 20) {
      ++authAttempt_;
    }
    ATT_LOG_W(kTag, "device authentication rejected (HTTP %d)", status);
    return false;
  }

  JsonDocument document;
  if (deserializeJson(document, response) || !document["success"].as<bool>()) {
    if (authAttempt_ < 20) {
      ++authAttempt_;
    }
    ATT_LOG_W(kTag, "device authentication returned invalid JSON");
    return false;
  }
  const int64_t id = document["device"]["id"].as<int64_t>();
  const String newToken = document["device"]["token"].as<String>();
  if (id <= 0 || newToken.length() == 0) {
    if (authAttempt_ < 20) {
      ++authAttempt_;
    }
    ATT_LOG_W(kTag, "device authentication response missing identity");
    return false;
  }
  deviceId_ = id;
  token_ = newToken;
  authAttempt_ = 0;
  ATT_LOG_I(kTag, "authenticated as device %ld", static_cast<long>(deviceId_));
  return true;
}

bool ApiClient::requestLocked(const char* method, const String& path, const String& body,
                              String& response, int& httpStatus, int timeoutMs, bool retryAuth) {
  httpStatus = 0;
  response = "";
  if (wifi_ == nullptr || !wifi_->connected()) {
    return false;
  }
  if (!authenticateLocked()) {
    return false;
  }
  if (!performHttpLocked(method, path, body, token_.c_str(), response, httpStatus, timeoutMs)) {
    return false;
  }
  if (retryAuth && (httpStatus == 401 || httpStatus == 403)) {
    ATT_LOG_W(kTag, "device token rejected; re-authenticating once");
    token_ = "";
    deviceId_ = 0;
    authAttempt_ = 0;
    lastAuthAttemptMs_ = 0;
    if (!authenticateLocked()) {
      return false;
    }
    return performHttpLocked(method, path, body, token_.c_str(), response, httpStatus, timeoutMs);
  }
  return httpStatus > 0;
}

bool ApiClient::performHttpLocked(const char* method, const String& path, const String& body,
                                  const char* token, String& response, int& httpStatus,
                                  int timeoutMs) {
  response = "";
  httpStatus = 0;
  if (wifi_ == nullptr || !wifi_->connected()) {
    return false;
  }
  const String url = baseUrl_ + path;
  const bool https = baseUrl_.startsWith("https://");
  if (!https && !config::kInsecureHttpAllowed) {
    ATT_LOG_E(kTag, "refusing plaintext API URL; use HTTPS or explicitly allow insecure HTTP");
    return false;
  }

  WiFiClientSecure secureClient;
  WiFiClient plainClient;
  HTTPClient http;

  http.setReuse(false);
  http.setFollowRedirects(HTTPC_DISABLE_FOLLOW_REDIRECTS);
  http.setTimeout(timeoutMs);
  http.setConnectTimeout(config::kHttpConnectTimeoutMs);

  NetworkClient* client = nullptr;
  if (https) {
    secureClient.setCACert(kGtsRootR4Pem);
    secureClient.setHandshakeTimeout(config::kHttpTimeoutMs);
    client = &secureClient;
  } else {
    client = &plainClient;
  }

  if (!http.begin(*client, url)) {
    httpStatus = -1;
    ATT_LOG_W(kTag, "HTTP begin failed");
    return false;
  }

  http.addHeader("Accept", "application/json");
  if (token != nullptr && token[0] != '\0') {
    http.addHeader("X-Device-Token", token);
  }
  if (strcmp(method, "GET") == 0) {
    httpStatus = http.GET();
  } else {
    http.addHeader("Content-Type", "application/json");
    httpStatus = http.POST(body);
  }
  if (httpStatus > 0) {
    const int announced = http.getSize();
    if (announced > static_cast<int>(config::kMaxHttpBodyBytes)) {
      httpStatus = -2;
      ATT_LOG_E(kTag, "response too large (%d bytes)", announced);
    } else {
      response = http.getString();
      if (response.length() > config::kMaxHttpBodyBytes) {
        httpStatus = -2;
        response = "";
        ATT_LOG_E(kTag, "response exceeded body limit");
      }
    }
  } else {
    ATT_LOG_W(kTag, "HTTP transport failure (%d)", httpStatus);
  }
  http.end();
  return httpStatus > 0;
}

AttendanceResult ApiClient::submitAttendance(uint16_t slot, const char* idempotencyKey,
                                             const char* capturedAt) {
  AttendanceResult result;
  if (!isValidSlot(slot) || !isValidIdempotencyKey(idempotencyKey)) {
    result.kind = AttendanceResult::Kind::Rejected;
    result.message = "Invalid local slot or idempotency key";
    return result;
  }

  JsonDocument request;
  request["idempotency_key"] = idempotencyKey;
  request["fingerprint_slot"] = slot;
  if (capturedAt != nullptr && capturedAt[0] != '\0') {
    request["captured_at"] = capturedAt;
  }
  String body;
  serializeJson(request, body);

  lock(mutex_);
  String response;
  int status = 0;
  const bool transportOk = requestLocked("POST", kAttendancePath, body, response, status,
                                        config::kHttpTimeoutMs, true);
  releaseLock(mutex_);

  if (!transportOk) {
    result.kind = AttendanceResult::Kind::Retryable;
    result.message = "Backend unavailable";
    return result;
  }
  if (status < 200 || status >= 300) {
    JsonDocument error;
    if (!deserializeJson(error, response)) {
      result.message = bounded(error["message"] | "", 180);
    }
    const HttpClass classification = classifyHttpStatus(status);
    result.kind = (classification == HttpClass::Reject)
                      ? AttendanceResult::Kind::Rejected
                      : AttendanceResult::Kind::Retryable;
    if (result.message.length() == 0) {
      result.message = "Backend rejected the scan";
    }
    ATT_LOG_W(kTag, "attendance HTTP %d for slot %u", status, static_cast<unsigned>(slot));
    return result;
  }

  JsonDocument document;
  if (deserializeJson(document, response)) {
    result.kind = AttendanceResult::Kind::Retryable;
    result.message = "Invalid backend response";
    return result;
  }
  if (!document["success"].as<bool>()) {
    result.kind = AttendanceResult::Kind::Rejected;
    result.message = bounded(document["message"] | "", 180);
    return result;
  }

  result.outcome = document["outcome"].as<String>();
  result.message = bounded(document["message"] | "", 180);
  if (document["duplicate"].as<bool>()) {
    result.kind = AttendanceResult::Kind::Duplicate;
  } else if (result.outcome == "entry" || result.outcome == "exit") {
    result.kind = AttendanceResult::Kind::Confirmed;
  } else if (result.outcome == "unknown_slot" || result.outcome == "member_inactive" ||
             result.outcome == "user_disabled" || result.outcome == "no_match") {
    result.kind = AttendanceResult::Kind::Rejected;
  } else {
    // A previously stored event that has not been processed yet is retried,
    // never reported as confirmed.
    result.kind = AttendanceResult::Kind::Retryable;
  }
  return result;
}

bool ApiClient::submitNoMatch(const char* idempotencyKey, const char* capturedAt) {
  if (!isValidIdempotencyKey(idempotencyKey)) {
    return false;
  }
  JsonDocument request;
  request["idempotency_key"] = idempotencyKey;
  request["result"] = "no_match";
  if (capturedAt != nullptr && capturedAt[0] != '\0') {
    request["captured_at"] = capturedAt;
  }
  String body;
  serializeJson(request, body);
  lock(mutex_);
  String response;
  int status = 0;
  const bool ok = requestLocked("POST", kAttendancePath, body, response, status,
                                config::kHttpTimeoutMs, true);
  releaseLock(mutex_);
  return ok && status >= 200 && status < 300;
}

bool ApiClient::syncBatch(const String& requestBody, BatchResult& result, int& httpStatus,
                          String& errorMessage) {
  result = BatchResult();
  errorMessage = "";
  lock(mutex_);
  String response;
  const bool ok = requestLocked("POST", "/api/v1/attendance-terminal/attendance/batch", requestBody,
                                response, httpStatus, config::kHttpTimeoutMs, true);
  releaseLock(mutex_);
  if (!ok) {
    errorMessage = "Backend unavailable";
    return false;
  }
  if (httpStatus < 200 || httpStatus >= 300) {
    JsonDocument error;
    if (!deserializeJson(error, response)) {
      errorMessage = bounded(error["message"] | "", 180);
    }
    return false;
  }

  JsonDocument document;
  if (deserializeJson(document, response) || !document["success"].as<bool>()) {
    errorMessage = "Invalid batch response";
    return false;
  }
  JsonArray items = document["results"].as<JsonArray>();
  for (JsonObject item : items) {
    if (result.count >= 20) {
      break;
    }
    BatchItemResult& target = result.items[result.count];
    const char* key = item["idempotency_key"] | "";
    strncpy(target.idempotencyKey, key, sizeof(target.idempotencyKey) - 1);
    target.success = item["success"] | false;
    target.retryable = item["retryable"] | true;
    target.message = bounded(item["message"] | "", 180);
    ++result.count;
  }
  return true;
}

bool ApiClient::sendHeartbeat(const HeartbeatInfo& info) {
  JsonDocument request;
  request["firmware_version"] = info.firmwareVersion;
  request["reader_model"] = info.readerModel;
  request["reader_connected"] = info.readerConnected;
  request["reader_readiness"] = info.readerReadiness;
  request["reader_key"] = "attendance";
  request["reader_label"] = "Attendance reader";
  request["reader_uart"] = info.readerUart;
  request["queue_depth"] = info.queueDepth;
  request["network_status"] = info.networkStatus;
  char timestamp[24];
  if (timeutil::formatIso8601(info.lastSyncEpoch, timestamp, sizeof(timestamp))) {
    request["last_sync_at"] = timestamp;
  }
  if (timeutil::formatIso8601(info.lastCaptureEpoch, timestamp, sizeof(timestamp))) {
    request["last_capture_at"] = timestamp;
  }
  String body;
  serializeJson(request, body);

  lock(mutex_);
  String response;
  int status = 0;
  const bool ok = requestLocked("POST", kHeartbeatPath, body, response, status,
                                config::kHttpTimeoutMs, true);
  releaseLock(mutex_);
  return ok && status >= 200 && status < 300;
}

bool ApiClient::pollEnrollment(EnrollmentInfo& info, int& httpStatus) {
  info = EnrollmentInfo();
  lock(mutex_);
  String response;
  const bool ok = requestLocked("GET", kPollPath, "", response, httpStatus,
                                config::kEnrollmentHttpTimeoutMs, true);
  releaseLock(mutex_);
  if (!ok || httpStatus < 200 || httpStatus >= 300) {
    return false;
  }
  return parseEnrollmentResponse(response, info);
}

bool ApiClient::getEnrollment(const char* uid, EnrollmentInfo& info, int& httpStatus) {
  info = EnrollmentInfo();
  if (!validSessionUid(uid)) {
    httpStatus = -3;
    return false;
  }
  lock(mutex_);
  String response;
  const bool ok = requestLocked("GET", enrollmentPath(uid), "", response, httpStatus,
                                config::kEnrollmentHttpTimeoutMs, true);
  releaseLock(mutex_);
  if (!ok || httpStatus < 200 || httpStatus >= 300) {
    return false;
  }
  return parseEnrollmentResponse(response, info);
}

bool ApiClient::claimEnrollment(const char* uid, int& httpStatus) {
  if (!validSessionUid(uid)) {
    httpStatus = -3;
    return false;
  }
  JsonDocument request;
  String body;
  serializeJson(request, body);
  lock(mutex_);
  String response;
  const bool ok = requestLocked("POST", enrollmentPath(uid, "/claim"), body, response, httpStatus,
                                config::kEnrollmentHttpTimeoutMs, true);
  releaseLock(mutex_);
  if (!ok || httpStatus < 200 || httpStatus >= 300) {
    return false;
  }
  JsonDocument document;
  return !deserializeJson(document, response) && document["success"].as<bool>();
}

bool ApiClient::reportProgress(const char* uid, const char* status, int slot, const char* errorCode,
                               const char* errorMessage, int& httpStatus) {
  if (!validSessionUid(uid) || status == nullptr) {
    httpStatus = -3;
    return false;
  }
  JsonDocument request;
  request["status"] = status;
  if (slot >= 1) {
    request["fingerprint_slot"] = slot;
  }
  if (errorCode != nullptr && errorCode[0] != '\0') {
    request["error_code"] = bounded(errorCode, 64);
  }
  if (errorMessage != nullptr && errorMessage[0] != '\0') {
    request["error_message"] = bounded(errorMessage, 240);
  }
  String body;
  serializeJson(request, body);
  lock(mutex_);
  String response;
  const bool ok = requestLocked("POST", enrollmentPath(uid, "/progress"), body, response, httpStatus,
                                config::kEnrollmentHttpTimeoutMs, true);
  releaseLock(mutex_);
  if (!ok || httpStatus < 200 || httpStatus >= 300) {
    return false;
  }
  JsonDocument document;
  return !deserializeJson(document, response) && document["success"].as<bool>();
}

bool ApiClient::completeEnrollment(const char* uid, int slot, const char* deviceUserId,
                                   int& httpStatus) {
  if (!validSessionUid(uid) || !isValidSlot(slot)) {
    httpStatus = -3;
    return false;
  }
  JsonDocument request;
  request["fingerprint_slot"] = slot;
  if (deviceUserId != nullptr && deviceUserId[0] != '\0') {
    request["device_user_id"] = bounded(deviceUserId, 80);
  }
  String body;
  serializeJson(request, body);
  lock(mutex_);
  String response;
  const bool ok = requestLocked("POST", enrollmentPath(uid, "/complete"), body, response, httpStatus,
                                config::kEnrollmentHttpTimeoutMs, true);
  releaseLock(mutex_);
  if (!ok || httpStatus < 200 || httpStatus >= 300) {
    return false;
  }
  JsonDocument document;
  return !deserializeJson(document, response) && document["success"].as<bool>();
}

bool ApiClient::failEnrollment(const char* uid, int slot, const char* errorCode,
                               const char* errorMessage, int& httpStatus) {
  if (!validSessionUid(uid)) {
    httpStatus = -3;
    return false;
  }
  JsonDocument request;
  if (slot >= 1) {
    request["fingerprint_slot"] = slot;
  }
  request["error_code"] = bounded(errorCode, 64);
  request["error_message"] = bounded(errorMessage, 240);
  String body;
  serializeJson(request, body);
  lock(mutex_);
  String response;
  const bool ok = requestLocked("POST", enrollmentPath(uid, "/fail"), body, response, httpStatus,
                                config::kEnrollmentHttpTimeoutMs, true);
  releaseLock(mutex_);
  if (!ok || httpStatus < 200 || httpStatus >= 300) {
    return false;
  }
  JsonDocument document;
  return !deserializeJson(document, response) && document["success"].as<bool>();
}

bool ApiClient::acknowledgeCleanup(const char* uid, bool success, const char* errorCode,
                                   const char* errorMessage, int& httpStatus) {
  if (!validSessionUid(uid)) {
    httpStatus = -3;
    return false;
  }
  JsonDocument request;
  request["success"] = success;
  if (errorCode != nullptr && errorCode[0] != '\0') {
    request["error_code"] = bounded(errorCode, 64);
  }
  if (errorMessage != nullptr && errorMessage[0] != '\0') {
    request["error_message"] = bounded(errorMessage, 240);
  }
  String body;
  serializeJson(request, body);
  lock(mutex_);
  String response;
  const bool ok = requestLocked("POST", enrollmentPath(uid, "/cleanup"), body, response, httpStatus,
                                config::kEnrollmentHttpTimeoutMs, true);
  releaseLock(mutex_);
  if (!ok || httpStatus < 200 || httpStatus >= 300) {
    return false;
  }
  JsonDocument document;
  return !deserializeJson(document, response) && document["success"].as<bool>();
}

bool ApiClient::parseEnrollmentResponse(const String& response, EnrollmentInfo& info) const {
  JsonDocument document;
  if (deserializeJson(document, response) || !document["success"].as<bool>()) {
    return false;
  }
  info.action = document["action"] | "";
  if (document["enrollment"].isNull()) {
    info.valid = false;
    return true;
  }
  return parseEnrollmentObject(document["enrollment"], info);
}

bool ApiClient::parseEnrollmentObject(JsonVariantConst value, EnrollmentInfo& info) const {
  if (value.isNull()) {
    return false;
  }
  JsonObjectConst object = value.as<JsonObjectConst>();
  info.uid = object["session_uid"] | "";
  info.status = object["status"] | "";
  info.memberName = object["member_name"] | "";
  info.memberLoginId = object["member_login_id"] | "";
  info.errorCode = object["error_code"] | "";
  info.errorMessage = object["error_message"] | "";
  info.slot = object["fingerprint_slot"] | 0;
  info.active = object["is_active"] | false;
  info.cleanupRequired = object["cleanup_required"] | false;
  info.valid = info.uid.length() > 0;
  return info.valid;
}

}  // namespace attendance
