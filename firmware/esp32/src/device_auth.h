#ifndef DEVICE_AUTH_H
#define DEVICE_AUTH_H

#include <Arduino.h>

// A backend enrolment mapped onto a local R307S template slot.
struct EnrollmentInfo {
  uint16_t slot = 0;
  int64_t memberId = 0;
  String loginId;
  String usn;
  String name;
  bool valid = false;
};

// Exchange DEVICE_API_KEY for a short-lived X-Device-Token.
bool authenticateDevice();
bool isDeviceAuthenticated();
void invalidateDeviceToken();
int64_t getDeviceId();
const String& getDeviceToken();

// POST /api/v1/biometric/submit-event. Returns true for 200/202 responses.
bool submitBiometricEvent(const String& eventType,
                          const String& memberIdentifier,
                          const String& idempotencyKey,
                          const String& payloadJson = "{}");

// POST /api/v1/biometric/enroll-fingerprint. Registers a device template slot
// against a member. The raw template is never persisted by the backend.
bool submitFingerprintEnrollment(int64_t memberId,
                                 const String& templateRef,
                                 const String& deviceUserId);

// GET /api/v1/biometric/status. Returns pending event count.
bool fetchDeviceStatus(int& pendingEvents);

// GET /api/v1/biometric/enrollments, cached into local slot -> member mappings.
bool fetchEnrollments(EnrollmentInfo* out, size_t maxCount, size_t& outCount);

// Hex HMAC-SHA256 using mbedTLS. Returns an empty string when no secret is set.
String hmacSha256Hex(const String& data);

#endif // DEVICE_AUTH_H
