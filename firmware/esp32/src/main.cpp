/*
 * Edwins Lab ESP32 biometric attendance and door access controller.
 *
 * Target: ESP32-WROOM-32UE (ALF V1.0) / PlatformIO esp32dev / Arduino.
 *
 * This is the only compiled firmware entry point: setup() and loop() live
 * here. The R307S driver, Wi-Fi manager and device-auth client are reusable
 * modules under src/.
 *
 * Safety rules enforced below:
 *   - A local fingerprint match never unlocks the door by itself.
 *   - The door unlocks only after the backend authorises the device, the
 *     scanned template is registered to a member on the backend, and the
 *     fingerprint_scan event is accepted.
 *   - The emergency exit releases the lock locally, without Wi-Fi or backend.
 *   - Emergency exit never creates a member logout event.
 */

#include "config.h"

#include <Arduino.h>

#include "R307S.h"
#include "device_auth.h"
#include "wifi_manager.h"

namespace {

enum SystemMode { MODE_IDLE, MODE_ENROLLMENT };

R307S loginReader(LOGIN_UART, LOGIN_RX, LOGIN_TX);
R307S logoutReader(LOGOUT_UART, LOGOUT_RX, LOGOUT_TX);

SystemMode mode = MODE_IDLE;

bool doorUnlocked = false;
bool emergencyActive = false;
unsigned long doorOpenedAt = 0;

unsigned long alarmUntil = 0;

unsigned long lastLoginScan = 0;
unsigned long lastLogoutScan = 0;
unsigned long lastHeartbeat = 0;
unsigned long lastAuthAttempt = 0;
unsigned long lastEnrollmentRefresh = 0;

EnrollmentInfo enrollments[MAX_ENROLLMENTS];
size_t enrollmentCount = 0;

int enrollMemberId = 0;
uint16_t enrollSlot = 0;
int enrollPhase = 0;
unsigned long lastEnrollScan = 0;

// ---------------------------------------------------------------------------
// Indicators and lock
// ---------------------------------------------------------------------------

void setLEDStatus(bool green, bool red, bool buzzer) {
  digitalWrite(GREEN_LED, green ? HIGH : LOW);
  digitalWrite(RED_LED, red ? HIGH : LOW);
  digitalWrite(BUZZER_PIN, buzzer ? HIGH : LOW);
}

void triggerAlarm(unsigned long durationMs) {
  digitalWrite(BUZZER_PIN, HIGH);
  alarmUntil = millis() + durationMs;
}

void updateAlarm() {
  if (alarmUntil != 0 && static_cast<long>(millis() - alarmUntil) >= 0) {
    digitalWrite(BUZZER_PIN, LOW);
    alarmUntil = 0;
  }
}

void buzzPattern(int pattern) {
  switch (pattern) {
    case 1: // short acknowledgement
      digitalWrite(BUZZER_PIN, HIGH);
      delay(70);
      digitalWrite(BUZZER_PIN, LOW);
      break;
    case 2: // success: two short beeps
      for (int i = 0; i < 2; i++) {
        digitalWrite(BUZZER_PIN, HIGH);
        delay(60);
        digitalWrite(BUZZER_PIN, LOW);
        delay(40);
      }
      break;
    case 3: // error: three short beeps
      for (int i = 0; i < 3; i++) {
        digitalWrite(BUZZER_PIN, HIGH);
        delay(60);
        digitalWrite(BUZZER_PIN, LOW);
        delay(40);
      }
      break;
    case 4: // long emergency tone
      triggerAlarm(500);
      break;
    default:
      break;
  }
}

void successSignal() {
  digitalWrite(GREEN_LED, HIGH);
  digitalWrite(RED_LED, LOW);
  buzzPattern(2);
}

void errorSignal() {
  digitalWrite(RED_LED, HIGH);
  digitalWrite(GREEN_LED, LOW);
  buzzPattern(3);
}

void lockDoor() {
  if (emergencyActive) {
    // After an emergency the lock stays released until manual intervention.
    return;
  }
  digitalWrite(RELAY_PIN, RELAY_LOCKED_LEVEL);
  doorUnlocked = false;
  Serial.println("Door locked");
}

void unlockDoor() {
  digitalWrite(RELAY_PIN, RELAY_UNLOCK_LEVEL);
  doorUnlocked = true;
  doorOpenedAt = millis();
  Serial.println("Door unlocked");
}

void updateDoor() {
  if (emergencyActive) {
    return;
  }
  if (doorUnlocked && millis() - doorOpenedAt >= DOOR_OPEN_MS) {
    lockDoor();
  }
}

// ---------------------------------------------------------------------------
// Emergency exit (independent of Wi-Fi and backend)
// ---------------------------------------------------------------------------

void checkEmergencyExit() {
  static bool lastReading = HIGH;
  static unsigned long lastChange = 0;
  static bool handled = false;

  bool reading = digitalRead(EXIT_BUTTON);
  if (reading != lastReading) {
    lastReading = reading;
    lastChange = millis();
    return;
  }
  if (millis() - lastChange < EXIT_DEBOUNCE_MS) {
    return;
  }

  if (reading == LOW && !handled) {
    handled = true;
    emergencyActive = true;
    // Release the fail-safe lock immediately, before any backend call.
    digitalWrite(RELAY_PIN, RELAY_UNLOCK_LEVEL);
    doorUnlocked = true;
    doorOpenedAt = millis();
    digitalWrite(GREEN_LED, LOW);
    digitalWrite(RED_LED, HIGH);
    triggerAlarm(3000);
    Serial.println("EMERGENCY EXIT: lock released locally");
    // Deliberately no biometric event: an exit is not a member logout.
  } else if (reading == HIGH) {
    handled = false;
  }
}

// ---------------------------------------------------------------------------
// Enrollment mapping cache
// ---------------------------------------------------------------------------

void refreshEnrollmentsIfDue(bool force) {
  if (!isWifiConnected() || !isDeviceAuthenticated()) {
    return;
  }
  if (!force && millis() - lastEnrollmentRefresh < HEARTBEAT_INTERVAL) {
    return;
  }
  EnrollmentInfo fresh[MAX_ENROLLMENTS];
  size_t count = 0;
  if (!fetchEnrollments(fresh, MAX_ENROLLMENTS, count)) {
    return;
  }
  for (size_t i = 0; i < count; i++) {
    enrollments[i] = fresh[i];
  }
  enrollmentCount = count;
  lastEnrollmentRefresh = millis();
  Serial.print("Cached enrollments: ");
  Serial.println(static_cast<unsigned>(count));
}

EnrollmentInfo* findEnrollment(uint16_t slot) {
  for (size_t i = 0; i < enrollmentCount; i++) {
    if (enrollments[i].valid && enrollments[i].slot == slot) {
      return &enrollments[i];
    }
  }
  return nullptr;
}

uint16_t nextFreeSlot() {
  for (uint16_t slot = 1; slot <= 127; slot++) {
    bool used = false;
    for (size_t i = 0; i < enrollmentCount; i++) {
      if (enrollments[i].valid && enrollments[i].slot == slot) {
        used = true;
        break;
      }
    }
    if (!used) {
      return slot;
    }
  }
  return 1;
}

void handleHeartbeat() {
  int pending = 0;
  if (!fetchDeviceStatus(pending)) {
    if (!isDeviceAuthenticated()) {
      Serial.println("Heartbeat: device not authenticated");
    }
    return;
  }
  Serial.print("Heartbeat OK, pending events: ");
  Serial.println(pending);
}

// ---------------------------------------------------------------------------
// Fingerprint scanning
// ---------------------------------------------------------------------------

bool scanFingerprintReader(R307S& reader, uint16_t& slot) {
  return reader.search(slot);
}

void recordLoginScan() {
  uint16_t slot = 0;
  if (!scanFingerprintReader(loginReader, slot)) {
    return;
  }
  if (millis() - lastLoginScan < SCAN_DEBOUNCE_MS) {
    return;
  }
  lastLoginScan = millis();

  if (!isDeviceAuthenticated()) {
    errorSignal();
    Serial.println("Login denied: device not authenticated");
    return;
  }

  EnrollmentInfo* enrollment = findEnrollment(slot);
  if (enrollment == nullptr) {
    // A local match that is not registered on the backend is not authorised.
    errorSignal();
    Serial.println("Login denied: template not enrolled on backend");
    return;
  }

  String memberIdentifier = enrollment->loginId.length() > 0 ? enrollment->loginId : enrollment->usn;
  if (memberIdentifier.length() == 0) {
    errorSignal();
    Serial.println("Login denied: enrollment has no member identifier");
    return;
  }

  String key = String("login_") + String(static_cast<unsigned long>(millis()));
  if (submitBiometricEvent("fingerprint_scan", memberIdentifier, key, "{}")) {
    unlockDoor();
    successSignal();
    Serial.println("LOGIN AUTHORISED");
  } else {
    errorSignal();
    Serial.println("Login denied by backend");
  }
}

void recordLogoutScan() {
  uint16_t slot = 0;
  if (!scanFingerprintReader(logoutReader, slot)) {
    return;
  }
  if (millis() - lastLogoutScan < SCAN_DEBOUNCE_MS) {
    return;
  }
  lastLogoutScan = millis();

  if (!isDeviceAuthenticated()) {
    errorSignal();
    Serial.println("Logout denied: device not authenticated");
    return;
  }

  EnrollmentInfo* enrollment = findEnrollment(slot);
  if (enrollment == nullptr) {
    errorSignal();
    Serial.println("Logout denied: template not enrolled on backend");
    return;
  }

  String memberIdentifier = enrollment->loginId.length() > 0 ? enrollment->loginId : enrollment->usn;
  if (memberIdentifier.length() == 0) {
    errorSignal();
    return;
  }

  String key = String("logout_") + String(static_cast<unsigned long>(millis()));
  if (submitBiometricEvent("fingerprint_scan", memberIdentifier, key, "{}")) {
    successSignal();
    Serial.println("LOGOUT RECORDED");
  } else {
    errorSignal();
    Serial.println("Logout denied by backend");
  }
}

// ---------------------------------------------------------------------------
// Enrollment
// ---------------------------------------------------------------------------

void enrollFingerprintComplete() {
  String slotRef = String("slot:") + String(static_cast<unsigned>(enrollSlot));
  String templateRef = String("r307s:") + slotRef;
  if (submitFingerprintEnrollment(enrollMemberId, templateRef, slotRef)) {
    successSignal();
    refreshEnrollmentsIfDue(true);
    Serial.println("Enrollment confirmed by backend");
  } else {
    errorSignal();
    Serial.println("Enrollment rejected by backend");
  }
  mode = MODE_IDLE;
  enrollPhase = 0;
}

void enrollNextScan() {
  if (mode != MODE_ENROLLMENT) {
    return;
  }
  if (millis() - lastEnrollScan < SCAN_DEBOUNCE_MS) {
    return;
  }

  if (enrollPhase == 0) {
    if (!loginReader.getImage(nullptr)) {
      return;
    }
    if (!loginReader.img2tz(static_cast<uint8_t>(1))) {
      errorSignal();
      return;
    }
    lastEnrollScan = millis();
    enrollPhase = 1;
    buzzPattern(1);
    Serial.println("First scan captured; scan the same finger again");
    return;
  }

  if (!loginReader.getImage(nullptr)) {
    return;
  }
  if (!loginReader.img2tz(static_cast<uint8_t>(2))) {
    errorSignal();
    return;
  }
  if (!loginReader.regModel()) {
    errorSignal();
    enrollPhase = 0;
    return;
  }
  if (!loginReader.storeModel(enrollSlot)) {
    errorSignal();
    enrollPhase = 0;
    return;
  }
  lastEnrollScan = millis();
  enrollFingerprintComplete();
}

} // namespace

// Starts enrollment for a member. Call this from an enrolment trigger (for
// example an admin provisioning command) when the member should be enrolled.
// Kept with external linkage so it can be invoked from future integrations.
void enrollFingerprintBegin(int memberId) {
  enrollMemberId = memberId;
  enrollSlot = nextFreeSlot();
  enrollPhase = 0;
  mode = MODE_ENROLLMENT;
  setLEDStatus(true, false, false);
  Serial.print("Enrollment started for member ");
  Serial.print(memberId);
  Serial.print(" into slot ");
  Serial.println(enrollSlot);
}

// ---------------------------------------------------------------------------
// Arduino entry points
// ---------------------------------------------------------------------------

void setup() {
  Serial.begin(115200);
  delay(200);
  Serial.println();
  Serial.println("Edwins Lab ESP32 controller starting");

  pinMode(RELAY_PIN, OUTPUT);
  pinMode(EXIT_BUTTON, INPUT_PULLUP);
  pinMode(GREEN_LED, OUTPUT);
  pinMode(RED_LED, OUTPUT);
  pinMode(BUZZER_PIN, OUTPUT);

  // Fail-safe default: de-energised relay is the locked state for this board's
  // wiring, and stays locked until the backend authorises a login.
  digitalWrite(RELAY_PIN, RELAY_LOCKED_LEVEL);
  digitalWrite(GREEN_LED, LOW);
  digitalWrite(RED_LED, LOW);
  digitalWrite(BUZZER_PIN, LOW);

  bool loginReady = loginReader.begin();
  bool logoutReady = logoutReader.begin();
  Serial.print("Login reader: ");
  Serial.println(loginReady ? "ready" : "not detected");
  Serial.print("Logout reader: ");
  Serial.println(logoutReady ? "ready" : "not detected");

  wifiBegin();
  Serial.println("Wi-Fi connection started");
}

void loop() {
  updateAlarm();

  // Emergency exit is checked first and never depends on Wi-Fi or the backend.
  checkEmergencyExit();

  updateDoor();
  wifiMaintain();

  if (!isWifiConnected()) {
    delay(5);
    return;
  }

  if (!isDeviceAuthenticated()) {
    if (millis() - lastAuthAttempt >= AUTH_RETRY_INTERVAL) {
      lastAuthAttempt = millis();
      if (authenticateDevice()) {
        refreshEnrollmentsIfDue(true);
        successSignal();
      }
    }
    delay(5);
    return;
  }

  if (millis() - lastHeartbeat >= HEARTBEAT_INTERVAL) {
    lastHeartbeat = millis();
    handleHeartbeat();
    refreshEnrollmentsIfDue(false);
  }

  if (mode == MODE_ENROLLMENT) {
    enrollNextScan();
  } else {
    recordLoginScan();
    recordLogoutScan();
  }

  delay(5);
}
