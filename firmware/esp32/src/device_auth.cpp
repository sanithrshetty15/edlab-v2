#include "device_auth.h"

#include "config.h"
#include "wifi_manager.h"

#include <Arduino.h>
#include <ArduinoJson.h>
#include <HTTPClient.h>
#include <mbedtls/md.h>
#include <string.h>

// Short-lived token issued by POST /api/v1/biometric/authenticate-device.
static String g_deviceToken;
static int64_t g_deviceId = 0;

bool isDeviceAuthenticated() {
  return g_deviceToken.length() > 0 && g_deviceId > 0;
}

void invalidateDeviceToken() {
  g_deviceToken = "";
}

int64_t getDeviceId() {
  return g_deviceId;
}

const String& getDeviceToken() {
  return g_deviceToken;
}

String hmacSha256Hex(const String& data) {
  const size_t secretLen = strlen(DEVICE_HMAC_SECRET);
  if (secretLen == 0) {
    return String("");
  }
  const mbedtls_md_info_t* info = mbedtls_md_info_from_type(MBEDTLS_MD_SHA256);
  if (info == nullptr) {
    return String("");
  }
  uint8_t digest[32];
  int rc = mbedtls_md_hmac(info,
                           reinterpret_cast<const unsigned char*>(DEVICE_HMAC_SECRET),
                           secretLen,
                           reinterpret_cast<const unsigned char*>(data.c_str()),
                           data.length(),
                           digest);
  if (rc != 0) {
    return String("");
  }
  char hex[65];
  for (size_t i = 0; i < sizeof(digest); i++) {
    sprintf(hex + (i * 2), "%02x", digest[i]);
  }
  hex[64] = '\0';
  return String(hex);
}

// ---------------------------------------------------------------------------
// Device authentication
// ---------------------------------------------------------------------------

bool authenticateDevice() {
  if (!isWifiConnected()) {
    return false;
  }

  HTTPClient http;
  http.setTimeout(8000);
  String url = String(API_BASE_URL) + "/api/v1/biometric/authenticate-device";
  if (!http.begin(url)) {
    http.end();
    return false;
  }
  http.addHeader("Content-Type", "application/json");

  JsonDocument request;
  request["api_key"] = DEVICE_API_KEY;
  request["device_name"] = DEVICE_NAME;
  String body;
  serializeJson(request, body);

  int httpCode = http.POST(body);
  String response = httpCode > 0 ? http.getString() : String("");
  http.end();

  if (httpCode != HTTP_CODE_OK) {
    Serial.print("Device auth HTTP ");
    Serial.print(httpCode);
    Serial.print(": ");
    Serial.println(response);
    return false;
  }

  JsonDocument doc;
  if (deserializeJson(doc, response)) {
    return false;
  }
  if (!doc["success"].as<bool>()) {
    Serial.print("Device auth rejected: ");
    Serial.println(doc["message"].as<String>());
    return false;
  }

  g_deviceId = doc["device"]["id"].as<int64_t>();
  g_deviceToken = doc["device"]["token"].as<String>();
  if (g_deviceToken.length() == 0 || g_deviceId <= 0) {
    g_deviceToken = "";
    g_deviceId = 0;
    return false;
  }

  Serial.print("Device authenticated, id=");
  Serial.println(static_cast<long>(g_deviceId));
  return true;
}

// ---------------------------------------------------------------------------
// Biometric events
// ---------------------------------------------------------------------------

bool submitBiometricEvent(const String& eventType,
                          const String& memberIdentifier,
                          const String& idempotencyKey,
                          const String& payloadJson) {
  if (!isWifiConnected()) {
    return false;
  }
  if (!isDeviceAuthenticated() && !authenticateDevice()) {
    return false;
  }
  if (eventType.length() == 0 || idempotencyKey.length() == 0) {
    return false;
  }

  // The backend uses an empty event_timestamp as "server now".
  const String timestamp = "";

  // Canonical string the backend signs:
  // deviceID|eventType|idempotencyKey|memberIdentifier|eventTimestamp
  String canonical = String(static_cast<long>(g_deviceId)) + "|" + eventType + "|" +
                     idempotencyKey + "|" + memberIdentifier + "|" + timestamp;
  String signature = hmacSha256Hex(canonical);

  // Only send payload data that is valid JSON.
  JsonDocument payloadCheck;
  String safePayload = payloadJson.length() > 0 ? payloadJson : String("{}");
  if (deserializeJson(payloadCheck, safePayload)) {
    safePayload = "{}";
  }

  JsonDocument request;
  request["device_id"] = g_deviceId;
  request["event_type"] = eventType;
  request["idempotency_key"] = idempotencyKey;
  if (memberIdentifier.length() > 0) {
    request["member_identifier"] = memberIdentifier;
  }
  request["event_timestamp"] = timestamp;
  request["payload"] = serialized(safePayload);
  if (signature.length() > 0) {
    request["hmac_signature"] = signature;
  }

  String body;
  serializeJson(request, body);

  HTTPClient http;
  http.setTimeout(8000);
  String url = String(API_BASE_URL) + "/api/v1/biometric/submit-event";
  if (!http.begin(url)) {
    http.end();
    return false;
  }
  http.addHeader("Content-Type", "application/json");
  http.addHeader("X-Device-Token", g_deviceToken);

  int httpCode = http.POST(body);
  String response = httpCode > 0 ? http.getString() : String("");
  http.end();

  if (httpCode == HTTP_CODE_UNAUTHORIZED || httpCode == HTTP_CODE_FORBIDDEN) {
    invalidateDeviceToken();
  }
  if (httpCode == HTTP_CODE_OK || httpCode == HTTP_CODE_ACCEPTED) {
    return true;
  }

  Serial.print("submit-event ");
  Serial.print(eventType);
  Serial.print(" failed HTTP ");
  Serial.print(httpCode);
  Serial.print(": ");
  Serial.println(response);
  return false;
}

bool submitFingerprintEnrollment(int64_t memberId,
                                 const String& templateRef,
                                 const String& deviceUserId) {
  if (!isWifiConnected()) {
    return false;
  }
  if (!isDeviceAuthenticated() && !authenticateDevice()) {
    return false;
  }
  if (memberId <= 0 || templateRef.length() == 0) {
    return false;
  }

  JsonDocument request;
  request["member_id"] = memberId;
  request["fingerprint_template"] = templateRef;
  request["device_user_id"] = deviceUserId;
  String body;
  serializeJson(request, body);

  HTTPClient http;
  http.setTimeout(8000);
  String url = String(API_BASE_URL) + "/api/v1/biometric/enroll-fingerprint";
  if (!http.begin(url)) {
    http.end();
    return false;
  }
  http.addHeader("Content-Type", "application/json");
  http.addHeader("X-Device-Token", g_deviceToken);

  int httpCode = http.POST(body);
  String response = httpCode > 0 ? http.getString() : String("");
  http.end();

  if (httpCode == HTTP_CODE_UNAUTHORIZED || httpCode == HTTP_CODE_FORBIDDEN) {
    invalidateDeviceToken();
  }
  if (httpCode == HTTP_CODE_OK) {
    return true;
  }
  Serial.print("Enrollment failed HTTP ");
  Serial.print(httpCode);
  Serial.print(": ");
  Serial.println(response);
  return false;
}

// ---------------------------------------------------------------------------
// Enrollment sessions
// ---------------------------------------------------------------------------

namespace {

bool performDeviceRequest(const char* method, const String& path,
                          const String& body, String& response, int& httpCode) {
  response = "";
  httpCode = 0;
  if (!isWifiConnected()) {
    return false;
  }
  if (!isDeviceAuthenticated() && !authenticateDevice()) {
    return false;
  }

  HTTPClient http;
  http.setTimeout(8000);
  String url = String(API_BASE_URL) + path;
  if (!http.begin(url)) {
    http.end();
    return false;
  }
  http.addHeader("Content-Type", "application/json");
  http.addHeader("X-Device-Token", g_deviceToken);

  httpCode = (strcmp(method, "GET") == 0) ? http.GET() : http.POST(body);
  if (httpCode > 0) {
    response = http.getString();
  }
  http.end();

  if (httpCode == HTTP_CODE_UNAUTHORIZED || httpCode == HTTP_CODE_FORBIDDEN) {
    invalidateDeviceToken();
  }
  return httpCode > 0;
}

void parseEnrollmentSession(JsonVariantConst variant, EnrollmentSessionInfo& out) {
  out = EnrollmentSessionInfo();
  if (variant.isNull()) {
    return;
  }
  JsonObjectConst item = variant.as<JsonObjectConst>();
  out.sessionUid = item["session_uid"].as<String>();
  out.status = item["status"].as<String>();
  out.memberName = item["member_name"].as<String>();
  out.memberLoginId = item["member_login_id"].as<String>();
  out.errorCode = item["error_code"].as<String>();
  out.errorMessage = item["error_message"].as<String>();
  out.memberId = item["member_id"].as<int64_t>();
  out.attempts = item["attempts"].as<int>();
  out.cleanupRequired = item["cleanup_required"].as<bool>();
  out.active = item["is_active"].as<bool>();
  JsonVariantConst slot = item["fingerprint_slot"];
  if (!slot.isNull()) {
    out.slot = slot.as<int>();
  }
  out.valid = out.sessionUid.length() > 0;
}

bool parseEnrollmentResponse(const String& response, int httpCode,
                             JsonDocument& doc, EnrollmentSessionInfo& out,
                             String* action = nullptr) {
  if (httpCode <= 0) {
    return false;
  }
  if (deserializeJson(doc, response)) {
    return true;
  }
  if (action != nullptr) {
    *action = doc["action"].as<String>();
  }
  parseEnrollmentSession(doc["enrollment"], out);
  return true;
}

} // namespace

bool authorizeDoorAccess(uint16_t slot, const String& readerKey,
                         bool& duplicate, String& reason) {
  duplicate = false;
  reason = "";
  if (slot < 1 || slot > 127) {
    reason = "invalid_slot";
    return false;
  }

  JsonDocument request;
  request["fingerprint_slot"] = slot;
  request["reader_key"] = readerKey.length() > 0 ? readerKey : String("login");
  String body;
  serializeJson(request, body);

  String response;
  int httpCode = 0;
  if (!performDeviceRequest("POST", "/api/v1/biometric/door-access", body, response, httpCode)) {
    reason = "backend_unreachable";
    return false;
  }
  if (httpCode != HTTP_CODE_OK) {
    Serial.print("door-access HTTP ");
    Serial.print(httpCode);
    Serial.print(": ");
    Serial.println(response);
    reason = String("http_") + String(httpCode);
    return false;
  }

  JsonDocument doc;
  if (deserializeJson(doc, response)) {
    reason = "invalid_response";
    return false;
  }
  duplicate = doc["duplicate"].as<bool>();
  reason = doc["reason"].as<String>();
  return doc["success"].as<bool>() && doc["authorized"].as<bool>();
}

bool fetchEnrollmentCommand(EnrollmentSessionInfo& out, String& action, int& httpCode) {
  out = EnrollmentSessionInfo();
  action = "";
  String response;
  if (!performDeviceRequest("GET", "/api/v1/biometric/enrollment-session", "", response, httpCode)) {
    return false;
  }
  JsonDocument doc;
  if (!parseEnrollmentResponse(response, httpCode, doc, out, &action)) {
    return false;
  }
  if (httpCode != HTTP_CODE_OK || !doc["success"].as<bool>()) {
    return false;
  }
  return true;
}

bool fetchEnrollmentSession(const String& sessionUid, EnrollmentSessionInfo& out,
                            String& action, int& httpCode) {
  out = EnrollmentSessionInfo();
  action = "";
  if (sessionUid.length() == 0) {
    return false;
  }
  String response;
  String path = String("/api/v1/biometric/enrollment-sessions/") + sessionUid;
  if (!performDeviceRequest("GET", path, "", response, httpCode)) {
    return false;
  }
  JsonDocument doc;
  if (!parseEnrollmentResponse(response, httpCode, doc, out, &action)) {
    return false;
  }
  return httpCode == HTTP_CODE_OK && doc["success"].as<bool>();
}

bool updateEnrollmentProgress(const String& sessionUid, const String& status,
                              int slot, const String& errorCode,
                              const String& errorMessage,
                              EnrollmentSessionInfo& out, int& httpCode) {
  out = EnrollmentSessionInfo();
  if (sessionUid.length() == 0 || status.length() == 0) {
    return false;
  }
  JsonDocument request;
  request["status"] = status;
  if (slot >= 1) {
    request["fingerprint_slot"] = slot;
  }
  if (errorCode.length() > 0) {
    request["error_code"] = errorCode;
  }
  if (errorMessage.length() > 0) {
    request["error_message"] = errorMessage;
  }
  String body;
  serializeJson(request, body);
  String response;
  String path = String("/api/v1/biometric/enrollment-sessions/") + sessionUid + "/progress";
  if (!performDeviceRequest("POST", path, body, response, httpCode)) {
    return false;
  }
  JsonDocument doc;
  return parseEnrollmentResponse(response, httpCode, doc, out);
}

bool completeEnrollment(const String& sessionUid, int slot,
                        EnrollmentSessionInfo& out, int& httpCode) {
  out = EnrollmentSessionInfo();
  if (sessionUid.length() == 0 || slot < 1) {
    return false;
  }
  JsonDocument request;
  request["fingerprint_slot"] = slot;
  request["device_user_id"] = String("slot:") + String(slot);
  String body;
  serializeJson(request, body);
  String response;
  String path = String("/api/v1/biometric/enrollment-sessions/") + sessionUid + "/complete";
  if (!performDeviceRequest("POST", path, body, response, httpCode)) {
    return false;
  }
  JsonDocument doc;
  return parseEnrollmentResponse(response, httpCode, doc, out);
}

bool failEnrollment(const String& sessionUid, int slot, const String& errorCode,
                    const String& errorMessage, EnrollmentSessionInfo& out,
                    int& httpCode) {
  out = EnrollmentSessionInfo();
  if (sessionUid.length() == 0) {
    return false;
  }
  JsonDocument request;
  if (slot >= 1) {
    request["fingerprint_slot"] = slot;
  }
  request["error_code"] = errorCode;
  request["error_message"] = errorMessage;
  String body;
  serializeJson(request, body);
  String response;
  String path = String("/api/v1/biometric/enrollment-sessions/") + sessionUid + "/fail";
  if (!performDeviceRequest("POST", path, body, response, httpCode)) {
    return false;
  }
  JsonDocument doc;
  return parseEnrollmentResponse(response, httpCode, doc, out);
}

bool acknowledgeEnrollmentCleanup(const String& sessionUid, int slot,
                                  EnrollmentSessionInfo& out, int& httpCode) {
  out = EnrollmentSessionInfo();
  if (sessionUid.length() == 0) {
    return false;
  }
  JsonDocument request;
  if (slot >= 1) {
    request["fingerprint_slot"] = slot;
  }
  String body;
  serializeJson(request, body);
  String response;
  String path = String("/api/v1/biometric/enrollment-sessions/") + sessionUid + "/cleanup";
  if (!performDeviceRequest("POST", path, body, response, httpCode)) {
    return false;
  }
  JsonDocument doc;
  return parseEnrollmentResponse(response, httpCode, doc, out);
}

bool fetchDeviceStatus(int& pendingEvents) {
  pendingEvents = 0;
  if (!isWifiConnected()) {
    return false;
  }
  if (!isDeviceAuthenticated() && !authenticateDevice()) {
    return false;
  }

  HTTPClient http;
  http.setTimeout(8000);
  String url = String(API_BASE_URL) + "/api/v1/biometric/status";
  if (!http.begin(url)) {
    http.end();
    return false;
  }
  http.addHeader("X-Device-Token", g_deviceToken);

  int httpCode = http.GET();
  String response = httpCode > 0 ? http.getString() : String("");
  http.end();

  if (httpCode == HTTP_CODE_UNAUTHORIZED || httpCode == HTTP_CODE_FORBIDDEN) {
    invalidateDeviceToken();
    return false;
  }
  if (httpCode != HTTP_CODE_OK) {
    return false;
  }

  JsonDocument doc;
  if (deserializeJson(doc, response)) {
    return false;
  }
  pendingEvents = doc["device"]["pending_events"].as<int>();
  return doc["success"].as<bool>();
}

bool fetchEnrollments(EnrollmentInfo* out, size_t maxCount, size_t& outCount) {
  outCount = 0;
  if (out == nullptr || maxCount == 0) {
    return false;
  }
  if (!isWifiConnected()) {
    return false;
  }
  if (!isDeviceAuthenticated() && !authenticateDevice()) {
    return false;
  }

  HTTPClient http;
  http.setTimeout(8000);
  String url = String(API_BASE_URL) + "/api/v1/biometric/enrollments";
  if (!http.begin(url)) {
    http.end();
    return false;
  }
  http.addHeader("X-Device-Token", g_deviceToken);

  int httpCode = http.GET();
  String response = httpCode > 0 ? http.getString() : String("");
  http.end();

  if (httpCode == HTTP_CODE_UNAUTHORIZED || httpCode == HTTP_CODE_FORBIDDEN) {
    invalidateDeviceToken();
    return false;
  }
  if (httpCode != HTTP_CODE_OK) {
    return false;
  }

  JsonDocument doc;
  if (deserializeJson(doc, response)) {
    return false;
  }
  JsonArray items = doc["enrollments"].as<JsonArray>();
  for (JsonObject item : items) {
    if (outCount >= maxCount) {
      break;
    }
    String status = item["status"].as<String>();
    if (status != "enrolled" && status != "verified") {
      continue;
    }
    String deviceUserId = item["device_user_id"].as<String>();
    int slot = -1;
    if (deviceUserId.startsWith("slot:")) {
      slot = deviceUserId.substring(5).toInt();
    } else if (deviceUserId.length() > 0 && isdigit(deviceUserId[0])) {
      slot = deviceUserId.toInt();
    }
    if (slot <= 0) {
      continue;
    }
    EnrollmentInfo& info = out[outCount];
    info.slot = static_cast<uint16_t>(slot);
    info.memberId = item["member_id"].as<int64_t>();
    info.loginId = item["member_login_id"].as<String>();
    info.usn = item["member_usn"].as<String>();
    info.name = item["member_name"].as<String>();
    info.valid = true;
    outCount++;
  }
  return true;
}
