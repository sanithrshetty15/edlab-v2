#pragma once
#include <Arduino.h>
#ifndef DEVICE_AUTH_H
#define DEVICE_AUTH_H

#include <cstdint>
#include <string>

// A backend enrolment mapped onto a local R307S template slot.
struct EnrollmentInfo {
  uint16_t slot = 0;
  int64_t memberId = 0;
  String loginId;
  String usn;
  String name;
  bool valid = false;
};

// An admin-created fingerprint enrollment session owned by this device.
struct EnrollmentSessionInfo {
  String sessionUid;
  String status;
  String memberName;
  String memberLoginId;
  String errorCode;
  String errorMessage;
  int64_t memberId = 0;
  int slot = 0;
  int attempts = 0;
  bool cleanupRequired = false;
  bool active = false;
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

// POST /api/v1/biometric/readers. Reports the single R307S entry reader and
// its last confirmed capture readiness so the backend can offer it for
// fingerprint enrollment.
bool reportDeviceReaders(bool connected, const char* readiness);

// POST /api/v1/biometric/door-access. The backend synchronously verifies that
// the fingerprint slot belongs to an active member on this device and returns
// an explicit authorization decision. The caller must only unlock the door when
// this returns true.
bool authorizeDoorAccess(uint16_t slot, const String& readerKey,
                         bool& duplicate, String& reason);

// POST /api/v1/biometric/enroll-fingerprint. Registers a device template slot
// against a member. The raw template is never persisted by the backend.
bool submitFingerprintEnrollment(int64_t memberId,
                                 const String& templateRef,
                                 const String& deviceUserId);

// Enrollment session endpoints. httpCode is set whenever the request reached
// the backend, including non-2xx responses, so callers can react to conflicts.
bool fetchEnrollmentCommand(EnrollmentSessionInfo& out, String& action, int& httpCode);
bool fetchEnrollmentSession(const String& sessionUid, EnrollmentSessionInfo& out,
                            String& action, int& httpCode);
bool updateEnrollmentProgress(const String& sessionUid, const String& status,
                              int slot, const String& errorCode,
                              const String& errorMessage,
                              EnrollmentSessionInfo& out, int& httpCode);
bool completeEnrollment(const String& sessionUid, int slot,
                        EnrollmentSessionInfo& out, int& httpCode);
bool failEnrollment(const String& sessionUid, int slot, const String& errorCode,
                    const String& errorMessage, EnrollmentSessionInfo& out,
                    int& httpCode);
bool acknowledgeEnrollmentCleanup(const String& sessionUid, int slot,
                                  EnrollmentSessionInfo& out, int& httpCode);

// GET /api/v1/biometric/status. Returns pending event count.
bool fetchDeviceStatus(int& pendingEvents);

// GET /api/v1/biometric/enrollments, cached into local slot -> member mappings.
bool fetchEnrollments(EnrollmentInfo* out, size_t maxCount, size_t& outCount);

// Hex HMAC-SHA256 using mbedTLS. Returns an empty string when no secret is set.
String hmacSha256Hex(const String& data);

#endif // DEVICE_AUTH_H
