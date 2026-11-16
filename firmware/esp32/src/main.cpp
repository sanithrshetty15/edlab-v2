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
 *   - The door unlocks only after the backend synchronously authorises the
 *     device, the fingerprint slot is registered to an active member, and the
 *     slot belongs to the entry reader.
 *   - The GPIO33 exit button releases the lock locally, without Wi-Fi or the
 *     backend, and re-locks automatically after DOOR_OPEN_MS.
 *   - The exit button never creates a member logout event.
 */

#include "config.h"

#include <Arduino.h>
#include <HTTPClient.h>
#include <Preferences.h>

#include "R307S.h"
#include "device_auth.h"
#include "enrollment_flow.h"
#include "reader_readiness.h"
#include "wifi_manager.h"

namespace {

enum SystemMode { MODE_IDLE, MODE_ENROLLMENT };

// One physical R307S reader handles entry, attendance and fingerprint
// enrollment. The legacy logout-reader configuration is intentionally unused.
R307S entryReader(LOGIN_UART, LOGIN_RX, LOGIN_TX);

R307S* enrollmentReader =
#if ENROLLMENT_READER == ENROLLMENT_READER_NONE
    nullptr;
#else
    &entryReader;
#endif

ReaderReadiness readerReadiness = ReaderReadiness::NOT_READY;
bool readerReportPending = false;
bool enrollmentDisabledLogged = false;

enum class SensorActivity { NONE, READINESS, ENROLLMENT };

SensorActivity sensorActivity = SensorActivity::NONE;

struct SensorActivityGuard {
  SensorActivity previous;
  explicit SensorActivityGuard(SensorActivity next) : previous(sensorActivity) {
    sensorActivity = next;
  }
  ~SensorActivityGuard() { sensorActivity = previous; }
};

SystemMode mode = MODE_IDLE;

bool doorUnlocked = false;
unsigned long doorOpenedAt = 0;

unsigned long alarmUntil = 0;

unsigned long lastLoginScan = 0;
unsigned long lastLoginScanLog = 0;
const unsigned long LOGIN_SCAN_LOG_INTERVAL = 2000UL;
unsigned long lastHeartbeat = 0;
unsigned long lastAuthAttempt = 0;
unsigned long lastEnrollmentRefresh = 0;
unsigned long lastEnrollmentPoll = 0;
unsigned long lastReaderReport = 0;
unsigned long lastReaderReportAttempt = 0;
unsigned long lastReaderPoll = 0;

EnrollmentInfo enrollments[MAX_ENROLLMENTS];
size_t enrollmentCount = 0;

struct ActiveEnrollment {
  bool active = false;
  String sessionUid;
  String memberName;
  int64_t memberId = 0;
  uint16_t slot = 0;
  bool slotStored = false;
  enrollment_flow::Stage stage = enrollment_flow::Stage::Idle;
  unsigned long lastScanAt = 0;
  unsigned long lastStatusCheck = 0;
  unsigned long lastCompleteAttempt = 0;
  uint8_t scanErrors = 0;
  bool readerPreflightDone = false;
};

ActiveEnrollment activeEnrollment;
Preferences enrollmentPrefs;
String persistedEnrollmentUid;
uint16_t persistedEnrollmentSlot = 0;
String persistedEnrollmentAction;
String persistedFailureCode;
String persistedFailureMessage;

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
  if (doorUnlocked && millis() - doorOpenedAt >= DOOR_OPEN_MS) {
    lockDoor();
  }
}

// ---------------------------------------------------------------------------
// Push-button exit (independent of Wi-Fi and backend)
// ---------------------------------------------------------------------------

void checkExitButton() {
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
    // Release the fail-safe lock immediately, before any backend call. The
    // normal updateDoor() timer re-locks the door after DOOR_OPEN_MS.
    unlockDoor();
    digitalWrite(GREEN_LED, HIGH);
    digitalWrite(RED_LED, LOW);
    buzzPattern(1);
    Serial.println("EXIT BUTTON: lock released locally");
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
  static EnrollmentInfo fresh[MAX_ENROLLMENTS];
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
  Serial.print(static_cast<unsigned>(count));
  Serial.print(", free heap: ");
  Serial.print(ESP.getFreeHeap());
  Serial.print(", stack headroom: ");
  Serial.print(static_cast<unsigned>(uxTaskGetStackHighWaterMark(nullptr) * sizeof(StackType_t)));
  Serial.println(" bytes");
}

bool slotIsFree(uint16_t slot, bool& freeSlot) {
  freeSlot = false;
  if (enrollmentReader == nullptr || slot < 1 || slot > 127) {
    return false;
  }
  bool exists = false;
  if (!enrollmentReader->templateExists(slot, exists)) {
    return false;
  }
  if (exists) {
    return true;
  }
  for (size_t i = 0; i < enrollmentCount; i++) {
    if (enrollments[i].valid && enrollments[i].slot == slot) {
      return true;
    }
  }
  freeSlot = true;
  return true;
}

bool findFreeSlot(uint16_t& slot) {
  for (uint16_t candidate = 1; candidate <= 127; candidate++) {
    bool freeSlot = false;
    if (!slotIsFree(candidate, freeSlot)) {
      return false;
    }
    if (freeSlot) {
      slot = candidate;
      return true;
    }
  }
  return false;
}

// ---------------------------------------------------------------------------
// Enrollment NVS recovery
// ---------------------------------------------------------------------------

bool hasPersistedEnrollment() {
  return persistedEnrollmentUid.length() > 0 && persistedEnrollmentSlot >= 1;
}

void clearPersistedEnrollment();

void loadPersistedEnrollment() {
  persistedEnrollmentUid = "";
  persistedEnrollmentSlot = 0;
  persistedEnrollmentAction = "";
  persistedFailureCode = "";
  persistedFailureMessage = "";
  if (!enrollmentPrefs.begin("enrollment", true)) {
    return;
  }
  persistedEnrollmentUid = enrollmentPrefs.getString("session_uid", "");
  persistedEnrollmentSlot = enrollmentPrefs.getUShort("slot", 0);
  persistedEnrollmentAction = enrollmentPrefs.getString("action", "");
  persistedFailureCode = enrollmentPrefs.getString("error_code", "");
  persistedFailureMessage = enrollmentPrefs.getString("error_message", "");
  enrollmentPrefs.end();
  if (persistedEnrollmentSlot > 127) {
    persistedEnrollmentSlot = 0;
  }
  if (persistedEnrollmentUid.length() > 0 && persistedEnrollmentSlot >= 1) {
    if (persistedEnrollmentAction != "complete" && persistedEnrollmentAction != "fail") {
      persistedEnrollmentAction = "fail";
      persistedFailureCode = "device_recovery";
      persistedFailureMessage = "Enrollment was interrupted before its outcome was confirmed";
    }
    Serial.print("Pending enrollment recovery: ");
    Serial.print(persistedEnrollmentUid);
    Serial.print(" slot ");
    Serial.print(static_cast<unsigned>(persistedEnrollmentSlot));
    Serial.print(" action ");
    Serial.println(persistedEnrollmentAction);
  } else if (persistedEnrollmentUid.length() > 0 || persistedEnrollmentSlot > 0) {
    clearPersistedEnrollment();
  } else {
    persistedEnrollmentUid = "";
    persistedEnrollmentSlot = 0;
  }
}

bool savePersistedEnrollment(const String& sessionUid, uint16_t slot,
                             const String& action = "complete",
                             const String& errorCode = "",
                             const String& errorMessage = "") {
  if (!enrollmentPrefs.begin("enrollment", false)) {
    return false;
  }
  bool ok = enrollmentPrefs.putString("session_uid", sessionUid) == sessionUid.length();
  ok = enrollmentPrefs.putUShort("slot", slot) == sizeof(uint16_t) && ok;
  ok = enrollmentPrefs.putString("action", action) == action.length() && ok;
  ok = enrollmentPrefs.putString("error_code", errorCode) == errorCode.length() && ok;
  ok = enrollmentPrefs.putString("error_message", errorMessage) == errorMessage.length() && ok;
  if (!ok) {
    enrollmentPrefs.remove("session_uid");
    enrollmentPrefs.remove("slot");
    enrollmentPrefs.remove("action");
    enrollmentPrefs.remove("error_code");
    enrollmentPrefs.remove("error_message");
  }
  enrollmentPrefs.end();
  if (ok) {
    persistedEnrollmentUid = sessionUid;
    persistedEnrollmentSlot = slot;
    persistedEnrollmentAction = action;
    persistedFailureCode = errorCode;
    persistedFailureMessage = errorMessage;
  }
  return ok;
}

void clearPersistedEnrollment() {
  persistedEnrollmentUid = "";
  persistedEnrollmentSlot = 0;
  persistedEnrollmentAction = "";
  persistedFailureCode = "";
  persistedFailureMessage = "";
  if (!enrollmentPrefs.begin("enrollment", false)) {
    return;
  }
  enrollmentPrefs.remove("session_uid");
  enrollmentPrefs.remove("slot");
  enrollmentPrefs.remove("action");
  enrollmentPrefs.remove("error_code");
  enrollmentPrefs.remove("error_message");
  enrollmentPrefs.end();
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

void setReaderReadiness(ReaderReadiness next) {
  if (readerReadiness == next) {
    return;
  }
  readerReadiness = next;
  readerReportPending = true;
  Serial.print("Reader readiness: ");
  Serial.println(readerReadinessStatus(readerReadiness));
}

void observeReaderImage(bool captured, uint8_t confirmation) {
  if (captured) {
    setReaderReadiness(ReaderReadiness::FINGER_DETECTED);
    return;
  }
  if (confirmation == R307S_OK) {
    setReaderReadiness(ReaderReadiness::FINGER_DETECTED);
    return;
  }
  if (confirmation == R307S_ERR_NOFINGER) {
    setReaderReadiness(ReaderReadiness::WAITING_FOR_FINGER);
    return;
  }
  if (confirmation == R307S_ERR_FLASH) {
    setReaderReadiness(ReaderReadiness::CAPTURE_FAILED);
    return;
  }
  if (confirmation == R307S_ERR_RECV) {
    setReaderReadiness(ReaderReadiness::CAPTURE_FAILED);
    return;
  }
  if (confirmation == R307S_NO_CONFIRMATION) {
    setReaderReadiness(ReaderReadiness::CAPTURE_FAILED);
    return;
  }
  // Other error codes: report sensor errors
  setReaderReadiness(ReaderReadiness::CAPTURE_FAILED);
}

void reportReaderIfDue(bool force) {
  if (enrollmentReader == nullptr) {
    return;
  }

  unsigned long now = millis();
  bool intervalDue = now - lastReaderReport >= READER_REPORT_INTERVAL;
  if (!force && !readerReportPending && !intervalDue) {
    return;
  }
  if (!force && readerReportPending &&
      now - lastReaderReportAttempt < READER_REPORT_RETRY_INTERVAL) {
    return;
  }

  lastReaderReportAttempt = now;
  if (!reportDeviceReaders(readerReadinessIsConnected(readerReadiness),
                           readerReadinessStatus(readerReadiness))) {
    lastReaderReport = now;
    Serial.println("Reader report failed; will retry");
    return;
  }
  lastReaderReport = now;
  readerReportPending = false;
  Serial.print("Reader reported: login, UART ");
  Serial.print(LOGIN_UART);
  Serial.print(", readiness=");
  Serial.println(readerReadinessStatus(readerReadiness));
}

// ---------------------------------------------------------------------------
// Fingerprint scanning
// ---------------------------------------------------------------------------

void logLoginScanIssue(const char* message, uint8_t confirmation) {
  unsigned long now = millis();
  if (lastLoginScanLog != 0 && now - lastLoginScanLog < LOGIN_SCAN_LOG_INTERVAL) {
    return;
  }
  lastLoginScanLog = now;
  Serial.print("Login scan: ");
  Serial.print(message);
  Serial.print(" (confirmation 0x");
  Serial.print(confirmation, HEX);
  Serial.println(")");
}

bool processCapturedFingerprint(R307S& reader, uint16_t& slot) {
  slot = 0;
  if (!reader.img2tz(static_cast<uint8_t>(1))) {
    setReaderReadiness(ReaderReadiness::CAPTURE_FAILED);
    logLoginScanIssue("image conversion failed", reader.lastConfirmation());
    return false;
  }
  if (!reader.search(slot)) {
    if (reader.lastConfirmation() == R307S_ERR_NOTFOUND) {
      logLoginScanIssue("unregistered fingerprint", R307S_ERR_NOTFOUND);
      setReaderReadiness(ReaderReadiness::READY);
    } else {
      logLoginScanIssue("template search failed", reader.lastConfirmation());
      setReaderReadiness(ReaderReadiness::CAPTURE_FAILED);
    }
    return false;
  }
  return true;
}

void processLoginCapture() {
  uint16_t slot = 0;
  if (!processCapturedFingerprint(entryReader, slot)) {
    return;
  }

  unsigned long now = millis();
  if (now - lastLoginScan < SCAN_DEBOUNCE_MS) {
    setReaderReadiness(ReaderReadiness::READY);
    return;
  }
  lastLoginScan = now;

  Serial.print("Fingerprint matched slot ");
  Serial.println(static_cast<unsigned>(slot));

  if (!isDeviceAuthenticated()) {
    errorSignal();
    setReaderReadiness(ReaderReadiness::READY);
    Serial.println("Entry denied: device not authenticated");
    return;
  }

  bool duplicate = false;
  String reason;
  if (authorizeDoorAccess(slot, "login", duplicate, reason)) {
    if (!duplicate) {
      unlockDoor();
    }
    successSignal();
    Serial.print("ENTRY AUTHORISED");
    if (duplicate) {
      Serial.print(" (duplicate scan ignored)");
    }
    Serial.println();
  } else {
    errorSignal();
    Serial.print("Entry denied by backend: ");
    Serial.println(reason.length() > 0 ? reason : String("not authorised"));
  }
  setReaderReadiness(ReaderReadiness::READY);
}

void pollReaderForLogin() {
  if (sensorActivity != SensorActivity::NONE) {
    return;
  }
  unsigned long now = millis();
  if (now - lastReaderPoll < READER_POLL_INTERVAL) {
    return;
  }
  if (now - lastLoginScan < SCAN_DEBOUNCE_MS) {
    return;
  }
  lastReaderPoll = now;

  SensorActivityGuard guard(SensorActivity::READINESS);
  if (!entryReader.getImage(nullptr)) {
    uint8_t confirmation = entryReader.lastConfirmation();
    observeReaderImage(false, confirmation);
    if (confirmation != R307S_ERR_NOFINGER) {
      logLoginScanIssue("image capture failed", confirmation);
    }
    return;
  }

  setReaderReadiness(ReaderReadiness::FINGER_DETECTED);
  processLoginCapture();
}

// ---------------------------------------------------------------------------
// Enrollment
// ---------------------------------------------------------------------------

bool deleteEnrollmentSlot(uint16_t slot) {
  if (slot < 1 || slot > 127) {
    return true;
  }
  if (enrollmentReader == nullptr) {
    return false;
  }
  if (!enrollmentReader->deleteModel(slot)) {
    Serial.print("Enrollment cleanup failed, sensor confirmation ");
    Serial.println(enrollmentReader->lastConfirmation());
    return false;
  }
  return true;
}

bool acknowledgeCleanup(const String& sessionUid, uint16_t slot) {
  if (sessionUid.length() == 0) {
    return false;
  }
  EnrollmentSessionInfo info;
  int httpCode = 0;
  if (!acknowledgeEnrollmentCleanup(sessionUid, slot, info, httpCode)) {
    return false;
  }
  return httpCode == HTTP_CODE_OK;
}

void stopEnrollment(bool success, const String& reason) {
  if (success) {
    successSignal();
  } else {
    errorSignal();
  }
  if (reason.length() > 0) {
    Serial.print("Enrollment stopped: ");
    Serial.println(reason);
  }
  if (success) {
    setReaderReadiness(ReaderReadiness::READY);
  }
  activeEnrollment = ActiveEnrollment();
  mode = MODE_IDLE;
  lastEnrollmentPoll = millis();
}

void handleTerminalEnrollment(const EnrollmentSessionInfo& info, bool deleteLocalTemplate) {
  if (info.status == "finalized") {
    clearPersistedEnrollment();
    refreshEnrollmentsIfDue(true);
    stopEnrollment(true, "finalized");
    return;
  }

  bool shouldDelete = deleteLocalTemplate || info.cleanupRequired;
  if (info.status == "failed" && info.errorCode == "slot_conflict") {
    shouldDelete = false;
  }

  uint16_t slot = activeEnrollment.slotStored ? activeEnrollment.slot : persistedEnrollmentSlot;
  String sessionUid = activeEnrollment.sessionUid.length() > 0
                          ? activeEnrollment.sessionUid
                          : persistedEnrollmentUid;

  if (shouldDelete && slot >= 1) {
    if (!deleteEnrollmentSlot(slot)) {
      stopEnrollment(false, "cleanup pending");
      return;
    }
    if (info.cleanupRequired) {
      acknowledgeCleanup(sessionUid, slot);
    }
  }

  clearPersistedEnrollment();
  String reason = info.status.length() > 0 ? info.status : String("rejected");
  if (info.errorCode.length() > 0) {
    reason += " (";
    reason += info.errorCode;
    reason += ")";
  }
  stopEnrollment(false, reason);
}

void reconcileActiveEnrollment() {
  EnrollmentSessionInfo info;
  String action;
  int httpCode = 0;
  if (!fetchEnrollmentSession(activeEnrollment.sessionUid, info, action, httpCode)) {
    if (httpCode == HTTP_CODE_NOT_FOUND) {
      handleTerminalEnrollment(info, true);
    }
    return;
  }
  if (action == "finalized" || info.status == "finalized") {
    handleTerminalEnrollment(info, false);
    return;
  }
  if (info.cleanupRequired || action == "cleanup") {
    handleTerminalEnrollment(info, true);
    return;
  }
  if (!info.valid) {
    return;
  }
  if (!info.active || info.status == "cancelled" || info.status == "expired" ||
      info.status == "failed") {
    handleTerminalEnrollment(info, true);
  }
}

bool reportActiveProgress(const char* status, int slot) {
  EnrollmentSessionInfo info;
  int httpCode = 0;
  if (!updateEnrollmentProgress(activeEnrollment.sessionUid, status, slot, "", "", info, httpCode)) {
    return false;
  }
  if (httpCode == HTTP_CODE_OK) {
    return true;
  }
  if (httpCode == HTTP_CODE_CONFLICT && info.valid) {
    handleTerminalEnrollment(info, true);
  }
  return false;
}

void attemptEnrollmentCompletion() {
  if (!activeEnrollment.active || activeEnrollment.stage != enrollment_flow::Stage::Finalizing) {
    return;
  }
  if (millis() - activeEnrollment.lastCompleteAttempt < ENROLLMENT_RECONCILE_INTERVAL) {
    return;
  }
  activeEnrollment.lastCompleteAttempt = millis();

  EnrollmentSessionInfo info;
  int httpCode = 0;
  if (!completeEnrollment(activeEnrollment.sessionUid, activeEnrollment.slot, info, httpCode)) {
    Serial.println("Enrollment completion request failed; will retry");
    return;
  }
  if (httpCode == HTTP_CODE_OK) {
    clearPersistedEnrollment();
    refreshEnrollmentsIfDue(true);
    stopEnrollment(true, "finalized");
    return;
  }
  if (httpCode == HTTP_CODE_UNAUTHORIZED || httpCode == HTTP_CODE_FORBIDDEN) {
    return;
  }
  if (httpCode == HTTP_CODE_CONFLICT) {
    if (info.valid) {
      handleTerminalEnrollment(info, true);
    } else {
      reconcileActiveEnrollment();
    }
    return;
  }
  if (httpCode == HTTP_CODE_NOT_FOUND) {
    handleTerminalEnrollment(info, true);
    return;
  }
  Serial.print("Enrollment completion HTTP ");
  Serial.println(httpCode);
}

void failActiveEnrollment(const String& errorCode, const String& message) {
  bool failureStateSaved = true;
  if (activeEnrollment.slotStored) {
    failureStateSaved = savePersistedEnrollment(activeEnrollment.sessionUid, activeEnrollment.slot,
                                                "fail", errorCode, message);
  }

  EnrollmentSessionInfo info;
  int httpCode = 0;
  int slot = activeEnrollment.slotStored ? activeEnrollment.slot : 0;
  bool reported = failEnrollment(activeEnrollment.sessionUid, slot, errorCode, message, info, httpCode);
  bool cleared = true;
  if (activeEnrollment.slotStored) {
    cleared = deleteEnrollmentSlot(activeEnrollment.slot);
    if (cleared && reported &&
        (httpCode == HTTP_CODE_OK || (httpCode == HTTP_CODE_CONFLICT && info.cleanupRequired))) {
      acknowledgeCleanup(activeEnrollment.sessionUid, activeEnrollment.slot);
    }
  }
  if (cleared) {
    clearPersistedEnrollment();
  } else if (!failureStateSaved) {
    clearPersistedEnrollment();
    Serial.println("Enrollment cleanup failed without recovery state; slot may be orphaned");
  }
  stopEnrollment(false, errorCode);
}

bool detectDuplicateFinger(uint16_t ownSlot, bool& duplicate) {
  duplicate = false;
  uint16_t foundSlot = 0;
  if (ownSlot > 1) {
    if (enrollmentReader->searchRange(1, static_cast<uint16_t>(ownSlot - 1), foundSlot)) {
      duplicate = true;
      return true;
    }
    if (enrollmentReader->lastConfirmation() != R307S_ERR_NOTFOUND) {
      return false;
    }
  }
  if (ownSlot < 127) {
    if (enrollmentReader->searchRange(static_cast<uint16_t>(ownSlot + 1),
                                      static_cast<uint16_t>(127 - ownSlot), foundSlot)) {
      duplicate = true;
      return true;
    }
    if (enrollmentReader->lastConfirmation() != R307S_ERR_NOTFOUND) {
      return false;
    }
  }
  return true;
}

void processActiveEnrollment() {
  if (!activeEnrollment.active) {
    mode = MODE_IDLE;
    return;
  }
  if (sensorActivity != SensorActivity::NONE) {
    return;
  }
  SensorActivityGuard guard(SensorActivity::ENROLLMENT);

  if (millis() - activeEnrollment.lastStatusCheck >= ENROLLMENT_RECONCILE_INTERVAL) {
    activeEnrollment.lastStatusCheck = millis();
    reconcileActiveEnrollment();
    if (!activeEnrollment.active) {
      return;
    }
  }

  if (activeEnrollment.stage == enrollment_flow::Stage::Finalizing) {
    attemptEnrollmentCompletion();
    return;
  }

  if (!activeEnrollment.readerPreflightDone &&
      activeEnrollment.stage == enrollment_flow::Stage::WaitingFirstScan) {
    if (millis() - activeEnrollment.lastScanAt < READER_POLL_INTERVAL) {
      return;
    }
    activeEnrollment.lastScanAt = millis();
    if (enrollmentReader->getImage(nullptr)) {
      setReaderReadiness(ReaderReadiness::FINGER_DETECTED);
      activeEnrollment.scanErrors = 0;
      return;
    }

    uint8_t confirmation = enrollmentReader->lastConfirmation();
    if (confirmation == R307S_ERR_NOFINGER) {
      activeEnrollment.scanErrors = 0;
      activeEnrollment.readerPreflightDone = true;
      setReaderReadiness(ReaderReadiness::WAITING_FOR_FINGER);
      reportActiveProgress(enrollment_flow::progressStatus(activeEnrollment.stage), 0);
      return;
    }

    activeEnrollment.scanErrors++;
    setReaderReadiness(ReaderReadiness::CAPTURE_FAILED);
    if (activeEnrollment.scanErrors >= 3) {
      failActiveEnrollment("sensor_error", "Enrollment reader is not ready");
    }
    return;
  }

  if (millis() - activeEnrollment.lastScanAt < ENROLLMENT_SCAN_INTERVAL) {
    return;
  }
  activeEnrollment.lastScanAt = millis();

  switch (activeEnrollment.stage) {
    case enrollment_flow::Stage::WaitingFirstScan: {
      if (!enrollmentReader->getImage(nullptr)) {
        if (enrollmentReader->lastConfirmation() == R307S_ERR_NOFINGER) {
          activeEnrollment.scanErrors = 0;
          setReaderReadiness(ReaderReadiness::WAITING_FOR_FINGER);
          return;
        }
        activeEnrollment.scanErrors++;
        setReaderReadiness(ReaderReadiness::CAPTURE_FAILED);
        if (activeEnrollment.scanErrors >= 3) {
          failActiveEnrollment("sensor_error", "Unable to capture the first fingerprint image");
        }
        return;
      }
      activeEnrollment.scanErrors = 0;
      setReaderReadiness(ReaderReadiness::FINGER_DETECTED);
      if (!enrollmentReader->img2tz(static_cast<uint8_t>(1))) {
        setReaderReadiness(ReaderReadiness::CAPTURE_FAILED);
        failActiveEnrollment("first_scan_failed", "Unable to process the first fingerprint image");
        return;
      }
      activeEnrollment.stage =
          enrollment_flow::next(activeEnrollment.stage, enrollment_flow::Event::FirstCaptured);
      buzzPattern(1);
      reportActiveProgress(enrollment_flow::progressStatus(activeEnrollment.stage), 0);
      return;
    }
    case enrollment_flow::Stage::FirstScanCaptured: {
      activeEnrollment.stage = enrollment_flow::next(
          activeEnrollment.stage, enrollment_flow::Event::FirstCaptureReported);
      reportActiveProgress(enrollment_flow::progressStatus(activeEnrollment.stage), 0);
      return;
    }
    case enrollment_flow::Stage::WaitingFingerRemoval: {
      if (enrollmentReader->getImage(nullptr)) {
        setReaderReadiness(ReaderReadiness::FINGER_DETECTED);
        return;
      }
      if (enrollmentReader->lastConfirmation() != R307S_ERR_NOFINGER) {
        activeEnrollment.scanErrors++;
        setReaderReadiness(ReaderReadiness::CAPTURE_FAILED);
        if (activeEnrollment.scanErrors >= 3) {
          failActiveEnrollment("sensor_error", "Finger removal could not be detected");
        }
        return;
      }
      activeEnrollment.scanErrors = 0;
      setReaderReadiness(ReaderReadiness::WAITING_FOR_FINGER);
      activeEnrollment.stage = enrollment_flow::next(
          activeEnrollment.stage, enrollment_flow::Event::FingerRemoved);
      buzzPattern(1);
      reportActiveProgress(enrollment_flow::progressStatus(activeEnrollment.stage), 0);
      return;
    }
    case enrollment_flow::Stage::WaitingSecondScan: {
      if (!enrollmentReader->getImage(nullptr)) {
        if (enrollmentReader->lastConfirmation() == R307S_ERR_NOFINGER) {
          activeEnrollment.scanErrors = 0;
          setReaderReadiness(ReaderReadiness::WAITING_FOR_FINGER);
          return;
        }
        activeEnrollment.scanErrors++;
        setReaderReadiness(ReaderReadiness::CAPTURE_FAILED);
        if (activeEnrollment.scanErrors >= 3) {
          failActiveEnrollment("sensor_error", "Unable to capture the second fingerprint image");
        }
        return;
      }
      activeEnrollment.scanErrors = 0;
      setReaderReadiness(ReaderReadiness::FINGER_DETECTED);
      if (!enrollmentReader->img2tz(static_cast<uint8_t>(2))) {
        setReaderReadiness(ReaderReadiness::CAPTURE_FAILED);
        failActiveEnrollment("second_scan_failed", "Unable to process the second fingerprint image");
        return;
      }
      setReaderReadiness(ReaderReadiness::READY);

      activeEnrollment.stage =
          enrollment_flow::next(activeEnrollment.stage, enrollment_flow::Event::SecondCaptured);
      reportActiveProgress(enrollment_flow::progressStatus(activeEnrollment.stage), 0);
      if (!activeEnrollment.active) {
        return;
      }

      uint16_t score = 0;
      if (!enrollmentReader->match(score) || score < 50) {
        failActiveEnrollment("finger_mismatch", "The two scans did not match the same finger");
        return;
      }
      if (!enrollmentReader->regModel()) {
        failActiveEnrollment("template_merge_failed", "Unable to create a fingerprint template");
        return;
      }

      uint16_t slot = 0;
      if (!findFreeSlot(slot)) {
        failActiveEnrollment("no_free_slot", "No free fingerprint slot is available");
        return;
      }
      activeEnrollment.slot = slot;
      if (!savePersistedEnrollment(activeEnrollment.sessionUid, slot)) {
        failActiveEnrollment("recovery_state_failed", "Unable to persist enrollment recovery state");
        return;
      }
      if (!enrollmentReader->storeModel(slot)) {
        failActiveEnrollment("sensor_store_failed", "Unable to store the fingerprint template");
        return;
      }
      activeEnrollment.slotStored = true;

      bool duplicate = false;
      if (!detectDuplicateFinger(slot, duplicate)) {
        failActiveEnrollment("duplicate_check_failed", "Unable to check for duplicate fingerprints");
        return;
      }
      if (duplicate) {
        failActiveEnrollment("duplicate_fingerprint", "This fingerprint is already enrolled");
        return;
      }

      activeEnrollment.stage =
          enrollment_flow::next(activeEnrollment.stage, enrollment_flow::Event::TemplateReady);
      reportActiveProgress(enrollment_flow::progressStatus(activeEnrollment.stage), slot);
      if (!activeEnrollment.active) {
        return;
      }
      activeEnrollment.stage = enrollment_flow::Stage::Finalizing;
      activeEnrollment.lastCompleteAttempt = 0;
      attemptEnrollmentCompletion();
      return;
    }
    default:
      return;
  }
}

void startActiveEnrollment(const EnrollmentSessionInfo& info) {
  activeEnrollment = ActiveEnrollment();
  activeEnrollment.active = true;
  activeEnrollment.sessionUid = info.sessionUid;
  activeEnrollment.memberId = info.memberId;
  activeEnrollment.memberName = info.memberName;
  activeEnrollment.lastScanAt = millis() - READER_POLL_INTERVAL;
  activeEnrollment.lastStatusCheck = millis();
  activeEnrollment.stage = enrollment_flow::Stage::WaitingFirstScan;
  mode = MODE_ENROLLMENT;
  setLEDStatus(true, false, false);

  Serial.print("Enrollment started for ");
  Serial.print(info.memberName.length() > 0 ? info.memberName : String("member"));
  Serial.print(", session ");
  Serial.println(info.sessionUid);

  if (info.slot >= 1 && info.slot <= 127) {
    bool exists = false;
    if (enrollmentReader->templateExists(static_cast<uint16_t>(info.slot), exists) && exists) {
      activeEnrollment.slot = static_cast<uint16_t>(info.slot);
      activeEnrollment.slotStored = true;
      activeEnrollment.stage = enrollment_flow::Stage::Finalizing;
      activeEnrollment.readerPreflightDone = true;
      setReaderReadiness(ReaderReadiness::READY);
      savePersistedEnrollment(activeEnrollment.sessionUid, activeEnrollment.slot);
      reportActiveProgress(enrollment_flow::progressStatus(activeEnrollment.stage), activeEnrollment.slot);
      return;
    }
  }
}

bool reconcilePersistedEnrollment() {
  EnrollmentSessionInfo info;
  String action;
  int httpCode = 0;
  if (!fetchEnrollmentSession(persistedEnrollmentUid, info, action, httpCode)) {
    if (httpCode == HTTP_CODE_NOT_FOUND) {
      if (!deleteEnrollmentSlot(persistedEnrollmentSlot)) {
        return false;
      }
      clearPersistedEnrollment();
      return true;
    }
    return false;
  }
  if (action == "finalized" || info.status == "finalized") {
    clearPersistedEnrollment();
    return true;
  }
  if (info.cleanupRequired || action == "cleanup") {
    if (!deleteEnrollmentSlot(persistedEnrollmentSlot)) {
      return false;
    }
    acknowledgeCleanup(persistedEnrollmentUid, persistedEnrollmentSlot);
    clearPersistedEnrollment();
    return true;
  }

  if (info.valid && info.active) {
    bool exists = false;
    if (!enrollmentReader->templateExists(persistedEnrollmentSlot, exists)) {
      return false;
    }
    if (!exists) {
      EnrollmentSessionInfo ignored;
      int failCode = 0;
      failEnrollment(persistedEnrollmentUid, 0, "local_template_missing",
                     "Stored fingerprint template is missing from the sensor",
                     ignored, failCode);
      clearPersistedEnrollment();
      return true;
    }

    if (persistedEnrollmentAction == "fail") {
      String errorCode = persistedFailureCode.length() > 0 ? persistedFailureCode : String("device_recovery");
      String errorMessage = persistedFailureMessage.length() > 0
                                ? persistedFailureMessage
                                : String("Enrollment was interrupted on the device");
      EnrollmentSessionInfo failed;
      int failCode = 0;
      bool reported = failEnrollment(persistedEnrollmentUid, persistedEnrollmentSlot,
                                     errorCode, errorMessage, failed, failCode);
      if (!reported) {
        return false;
      }
      if (failCode == HTTP_CODE_UNAUTHORIZED || failCode == HTTP_CODE_FORBIDDEN) {
        return false;
      }
      if (failCode == HTTP_CODE_NOT_FOUND) {
        if (!deleteEnrollmentSlot(persistedEnrollmentSlot)) {
          return false;
        }
        clearPersistedEnrollment();
        return true;
      }
      if (failCode != HTTP_CODE_OK && failCode != HTTP_CODE_CONFLICT) {
        return false;
      }
      if (!deleteEnrollmentSlot(persistedEnrollmentSlot)) {
        return false;
      }
      if (failed.cleanupRequired) {
        acknowledgeCleanup(persistedEnrollmentUid, persistedEnrollmentSlot);
      }
      clearPersistedEnrollment();
      return true;
    }

    activeEnrollment = ActiveEnrollment();
    activeEnrollment.active = true;
    activeEnrollment.sessionUid = persistedEnrollmentUid;
    activeEnrollment.memberId = info.memberId;
    activeEnrollment.memberName = info.memberName;
    activeEnrollment.slot = persistedEnrollmentSlot;
    activeEnrollment.slotStored = true;
    activeEnrollment.stage = enrollment_flow::Stage::Finalizing;
    activeEnrollment.readerPreflightDone = true;
    activeEnrollment.lastStatusCheck = millis();
    mode = MODE_ENROLLMENT;
    setLEDStatus(true, false, false);
    reportActiveProgress(enrollment_flow::progressStatus(activeEnrollment.stage),
                         activeEnrollment.slot);
    return true;
  }

  if (!info.valid) {
    return false;
  }

  bool deleteLocal = !(info.status == "failed" && info.errorCode == "slot_conflict");
  if (deleteLocal) {
    if (!deleteEnrollmentSlot(persistedEnrollmentSlot)) {
      return false;
    }
  }
  clearPersistedEnrollment();
  return true;
}

void processEnrollmentCommand() {
  if (mode != MODE_IDLE) {
    return;
  }
  if (enrollmentReader == nullptr) {
    if (!enrollmentDisabledLogged) {
      enrollmentDisabledLogged = true;
      Serial.println("Enrollment reader is not configured; enrollment disabled");
    }
    return;
  }
  if (!readerReadinessCanStartEnrollment(readerReadiness)) {
    return;
  }
  if (!isWifiConnected() || !isDeviceAuthenticated()) {
    return;
  }
  if (millis() - lastEnrollmentPoll < ENROLLMENT_POLL_INTERVAL) {
    return;
  }
  if (sensorActivity != SensorActivity::NONE) {
    return;
  }
  lastEnrollmentPoll = millis();
  SensorActivityGuard guard(SensorActivity::ENROLLMENT);

  if (hasPersistedEnrollment()) {
    reconcilePersistedEnrollment();
    return;
  }

  EnrollmentSessionInfo info;
  String action;
  int httpCode = 0;
  if (!fetchEnrollmentCommand(info, action, httpCode)) {
    return;
  }
  if (action == "cleanup") {
    if (info.slot >= 1 && !deleteEnrollmentSlot(static_cast<uint16_t>(info.slot))) {
      return;
    }
    acknowledgeCleanup(info.sessionUid, info.slot >= 1 ? static_cast<uint16_t>(info.slot) : 0);
    return;
  }
  if (action == "enroll" && info.active) {
    startActiveEnrollment(info);
  }
}

} // namespace

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

  bool entryReady = entryReader.begin();
  readerReadiness = entryReady ? ReaderReadiness::INITIALIZING : ReaderReadiness::NOT_READY;
  Serial.print("Entry reader: ");
  Serial.println(entryReady ? "password verified; capture readiness pending" : "not detected");

  loadPersistedEnrollment();
#if ENROLLMENT_READER == ENROLLMENT_READER_NONE
  Serial.println("Enrollment reader: disabled");
#else
  Serial.println("Enrollment reader: configured; capture readiness pending");
#endif

  wifiBegin();
  Serial.println("Wi-Fi connection started");
  lastAuthAttempt = millis() - AUTH_RETRY_INTERVAL;
}

void loop() {
  updateAlarm();

  // Push-button exit is checked first and never depends on Wi-Fi or the backend.
  checkExitButton();

  updateDoor();
  wifiMaintain();

  if (!isWifiConnected()) {
    delay(5);
    return;
  }

  if (!isDeviceAuthenticated()) {
    if (millis() - lastAuthAttempt >= AUTH_RETRY_INTERVAL) {
      lastAuthAttempt = millis();
      Serial.println("Attempting backend device authentication...");
      if (authenticateDevice()) {
        reportReaderIfDue(true);
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
  reportReaderIfDue(false);

  if (mode == MODE_ENROLLMENT) {
    processActiveEnrollment();
  } else {
    pollReaderForLogin();
    if (mode == MODE_IDLE) {
      processEnrollmentCommand();
    }
  }

  delay(5);
}
