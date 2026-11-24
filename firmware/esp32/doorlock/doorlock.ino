/*
 * Edwins Lab biometric door lock firmware
 * ======================================
 *
 * Single-file Arduino IDE sketch for the ALF V1.0 controller.
 *
 * Hardware:
 *   Controller .......... ESP32-WROOM-32UE on an ALF V1.0 board
 *   Fingerprint reader .. R307S on UART2, RX GPIO16, TX GPIO17, 57600 baud
 *   Door relay .......... GPIO25, active LOW (LOW = unlocked, HIGH = locked)
 *   Exit push button .... GPIO33, active LOW (INPUT_PULLUP)
 *   Green LED ........... GPIO23
 *   Red LED ............. GPIO22
 *   Buzzer .............. GPIO21
 *
 * Arduino IDE: board "ESP32 Dev Module" (or your ALF V1.0 definition), upload
 * speed 921600, serial monitor 115200.
 *
 * Design rules enforced by this sketch:
 *   - The R307S stores the fingerprint templates. The ESP32 only ever learns
 *     a slot number from the sensor and stores no member personal data.
 *   - A local fingerprint match never unlocks the door by itself. The backend
 *     is the only authority: it maps the slot to a member, decides whether the
 *     member may enter, and records the access attempt.
 *   - Enrollment is driven by admin sessions created in the backend. The
 *     backend allocates the slot, and a fingerprint is only "enrolled" once the
 *     sensor stored the template AND the backend confirmed the slot mapping.
 *   - The exit push button releases the lock locally, without Wi-Fi or the
 *     backend, and the lock re-engages after DOOR_OPEN_MS.
 *   - The single reader is shared: login scanning is suspended for the whole
 *     enrollment / cleanup state machine.
 *
 * Required libraries (Library Manager):
 *   - ArduinoJson (Benoit Blanchon) 7.x
 *   - WiFi, HTTPClient and Preferences ship with the ESP32 Arduino core.
 */

// ===========================================================================
// 1. CONFIGURATION
// ===========================================================================
//
// Never commit real credentials. Either edit the placeholders below locally,
// or create a sibling file "doorlock_secrets.h" next to this sketch (and add
// it to .gitignore) that defines any of the macros below; it is included
// automatically when present.

#if __has_include("doorlock_secrets.h")
#include "doorlock_secrets.h"
#endif

#ifndef WIFI_SSID
#define WIFI_SSID "Edwins"
#endif
#ifndef WIFI_PASSWORD
#define WIFI_PASSWORD "OpenSource@2008"
#endif

// Backend base URL, without a trailing slash.
#ifndef API_BASE_URL
#define API_BASE_URL "https://edwinslab.onrender.com"
#endif

// Device API key provisioned by an administrator through POST /api/v1/devices.
// The backend exchanges it for a short-lived X-Device-Token.
#ifndef DEVICE_API_KEY
#define DEVICE_API_KEY "CHANGE_ME_DEVICE_API_KEY"
#endif

// Friendly name reported during device authentication.
#ifndef DEVICE_NAME
#define DEVICE_NAME "Edwins Lab ESP32 (ALF V1.0)"
#endif

// ===========================================================================
// 2. HARDWARE (ALF V1.0 pinout)
// ===========================================================================

// Fingerprint reader on UART2.
const int LOGIN_UART = 2;
const int LOGIN_RX = 16;
const int LOGIN_TX = 17;
const unsigned long FINGERPRINT_BAUD = 57600;

// Fail-safe relay wiring: energised while locked, released when unlocked.
// RELAY_UNLOCK_LEVEL is the relay's active (energised) level on this board.
const int RELAY_PIN = 25;
const int RELAY_LOCKED_LEVEL = HIGH;
const int RELAY_UNLOCK_LEVEL = LOW;

// Exit push button, active LOW with the internal pull-up enabled.
const int EXIT_BUTTON = 33;

// Status indicators.
const int GREEN_LED = 23;
const int RED_LED = 22;
const int BUZZER_PIN = 21;

// ===========================================================================
// 3. TIMING
// ===========================================================================

const unsigned long DOOR_OPEN_MS = 5000UL;          // unlock period
const unsigned long EXIT_DEBOUNCE_MS = 500UL;
const unsigned long SCAN_DEBOUNCE_MS = 750UL;       // repeated login scans
const unsigned long READER_POLL_INTERVAL = 1000UL;  // idle finger probe
const unsigned long HEARTBEAT_INTERVAL = 30000UL;
const unsigned long WIFI_RETRY_INTERVAL = 15000UL;
const unsigned long AUTH_RETRY_INTERVAL = 60000UL;

// Reader availability reporting (the backend treats it as stale after 90s).
const unsigned long READER_REPORT_INTERVAL = 30000UL;
const unsigned long READER_REPORT_RETRY_INTERVAL = 5000UL;

// Enrollment polling and state-machine cadence.
const unsigned long ENROLLMENT_POLL_INTERVAL = 1000UL;
const unsigned long ENROLLMENT_IDLE_POLL_INTERVAL = 2000UL;
const unsigned long ENROLLMENT_RECONCILE_INTERVAL = 10000UL;
const unsigned long ENROLLMENT_COMPLETE_RETRY_INTERVAL = 3000UL;
const unsigned long ENROLLMENT_CLEANUP_RETRY_INTERVAL = 3000UL;
const unsigned long ENROLLMENT_CLAIM_RETRY_INTERVAL = 1000UL;
const unsigned long ENROLLMENT_RESUME_COOLDOWN_MS = 1500UL;
const unsigned long ENROLLMENT_SCAN_INTERVAL = 200UL;
const unsigned long ENROLLMENT_PREFLIGHT_INTERVAL = 250UL;
const int ENROLLMENT_HTTP_TIMEOUT_MS = 4000;

// R307S slot range (verified capacity of the module family).
const int SLOT_MIN = 1;
const int SLOT_MAX = 127;

// ===========================================================================
// 4. INCLUDES
// ===========================================================================

#include <Arduino.h>
#include <ArduinoJson.h>
#include <HTTPClient.h>
#include <Preferences.h>
#include <WiFi.h>

// ===========================================================================
// 5. SHARED TYPES
// ---------------------------------------------------------------------------
// Every type is declared before the first function in this sketch: the Arduino
// IDE and PlatformIO both insert generated function prototypes before the first
// function body, so anything used in a signature has to exist by then.
// ===========================================================================

// Reader availability as reported to the backend. A reader is only ready when
// the sensor itself confirms capture readiness, not just a password handshake.
enum class ReaderReadiness : uint8_t {
  NOT_READY,
  INITIALIZING,
  READY,
  WAITING_FOR_FINGER,
  FINGER_DETECTED,
  CAPTURE_FAILED,
  ENROLLING,
};

// Exclusive sensor ownership. Login scanning is suspended whenever enrollment
// or enrollment cleanup owns the reader.
enum class SensorMode : uint8_t {
  IDLE,
  LOGIN,
  ENROLLMENT,
  ENROLLMENT_CLEANUP,
};

enum class SensorActivity : uint8_t {
  NONE,
  READINESS,
  ENROLLMENT,
};

// Exact two-scan enrollment sequence. Each stage has a single responsibility so
// the serial log and the backend progress always identify where it stopped.
enum class Stage : uint8_t {
  Idle,
  ClaimingSession,
  SensorInitialization,
  WaitingForFirstFinger,
  CapturingFirstImage,
  ConvertingFirstImage,
  WaitingForFingerRemoval,
  WaitingForSecondFinger,
  CapturingSecondImage,
  ConvertingSecondImage,
  CreatingModel,
  StoringTemplate,
  BackendFinalization,
  EnrollmentCleanup,
  Completed,
  Failed,
};

enum class Event : uint8_t {
  Start,
  CommandClaimed,
  SensorReady,
  FirstFingerDetected,
  FirstImageCaptured,
  FirstImageConverted,
  FingerRemoved,
  SecondFingerDetected,
  SecondImageCaptured,
  SecondImageConverted,
  ModelCreated,
  TemplateStored,
  FinalizeRequested,
  BackendFinalized,
  CleanupRequired,
  CleanupCompleted,
  Failed,
  Reset,
};

// An admin-created fingerprint enrollment session owned by this device. The
// backend allocates the fingerprint slot; the device never chooses one.
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

struct ActiveEnrollment {
  bool active = false;
  String sessionUid;
  String memberName;
  int64_t memberId = 0;
  uint16_t slot = 0;
  bool slotStored = false;
  Stage stage = Stage::Idle;
  unsigned long lastScanAt = 0;
  unsigned long lastStatusCheck = 0;
  unsigned long lastClaimAttempt = 0;
  unsigned long lastCompleteAttempt = 0;
  uint8_t scanErrors = 0;
  bool cleanupPending = false;
  String terminalReason;
};

// R307S command response trace, used to diagnose a misbehaving sensor.
struct R307SResponseDiagnostics {
  unsigned long rxBytes = 0;
  bool headerDetected = false;
  bool headerComplete = false;
  bool packetInfoAvailable = false;
  uint8_t packetId = 0;
  uint16_t length = 0;
  bool payloadComplete = false;
  bool checksumChecked = false;
  bool checksumValid = false;
  uint8_t confirmation = 0xFF;
};

// Tracks which activity currently has sensor access. Declared last, together
// with its RAII guard, so every type above exists before any global.
SensorActivity sensorActivity = SensorActivity::NONE;

struct SensorActivityGuard {
  SensorActivity previous;
  explicit SensorActivityGuard(SensorActivity next) : previous(sensorActivity) {
    sensorActivity = next;
  }
  ~SensorActivityGuard() { sensorActivity = previous; }
};

// ===========================================================================
// 6. GLOBAL STATE
// ===========================================================================

// The ALF V1.0 board has exactly one reader, used for both login and
// enrollment. Only one activity may talk to the sensor at a time.
HardwareSerial r307sSerial(LOGIN_UART);
uint8_t r307sConfirmation = 0xFF; // confirmation of the last command

SensorMode currentMode = SensorMode::IDLE;

ReaderReadiness readerReadiness = ReaderReadiness::NOT_READY;
bool readerReportPending = false;

bool doorUnlocked = false;
unsigned long doorOpenedAt = 0;

unsigned long lastLoginScan = 0;
unsigned long lastLoginScanLog = 0;
const unsigned long LOGIN_SCAN_LOG_INTERVAL = 2000UL;

unsigned long lastHeartbeat = 0;
unsigned long lastAuthAttempt = 0;
unsigned long lastWifiAttempt = 0;
unsigned long lastEnrollmentPoll = 0;
unsigned long lastEnrollmentIdlePollLog = 0;
unsigned long enrollmentPollInterval = ENROLLMENT_POLL_INTERVAL;
unsigned long lastReaderReport = 0;
unsigned long lastReaderReportAttempt = 0;
unsigned long lastReaderPoll = 0;
unsigned long sensorResumeAt = 0;

ActiveEnrollment activeEnrollment;

Preferences enrollmentPrefs;
String persistedEnrollmentUid;
uint16_t persistedEnrollmentSlot = 0;
String persistedEnrollmentAction;
String persistedFailureCode;
String persistedFailureMessage;

// Short-lived device token issued by POST /api/v1/biometric/authenticate-device.
String deviceToken;
int64_t deviceId = 0;

// ===========================================================================
// 7. FORWARD DECLARATIONS
// ===========================================================================

void failActiveEnrollment(const String& errorCode, const String& message);
void handleTerminalEnrollment(const EnrollmentSessionInfo& info, bool deleteLocalTemplate);
void reconcileActiveEnrollment();
bool reportActiveProgress(const char* status, int slot);
void attemptEnrollmentCompletion();
void clearPersistedEnrollment();
void stopEnrollment(bool success, const String& reason);
void setReaderReadiness(ReaderReadiness next);
void reportReaderIfDue(bool force);
bool deleteEnrollmentSlot(uint16_t slot);
bool acknowledgeCleanup(const String& sessionUid, uint16_t slot);
void handleHeartbeat();

// ===========================================================================
// 8. R307S FINGERPRINT DRIVER (UART, standard R30x protocol)
// ===========================================================================

// Instruction codes.
#define R307S_GENIMG 0x01
#define R307S_IMG2TZ 0x02
#define R307S_MATCH 0x03
#define R307S_SEARCH 0x04
#define R307S_REGMODEL 0x05
#define R307S_STORE 0x06
#define R307S_LOADCHAR 0x07
#define R307S_DELCHAR 0x0C
#define R307S_VERIFYPWD 0x13

// Confirmation codes.
#define R307S_OK 0x00
#define R307S_ERR_RECV 0x01
#define R307S_ERR_NOFINGER 0x02
#define R307S_ERR_ENROLL 0x03
#define R307S_ERR_IMAGE 0x06
#define R307S_ERR_MATCH 0x08
#define R307S_ERR_NOTFOUND 0x09
#define R307S_ERR_EMPTY 0x0A
#define R307S_ERR_FLASH 0x18
#define R307S_NO_CONFIRMATION 0xFF

// Factory default module password.
#define R307S_DEFAULT_PASSWORD 0x00000000UL

// Per-command response timeout. A flat timeout makes a full library search
// unreliable without making real failures any faster.
unsigned long r307sResponseTimeoutFor(uint8_t command) {
  switch (command) {
    case R307S_SEARCH:
      return 4000; // full 127-template library search
    case R307S_REGMODEL:
    case R307S_STORE:
    case R307S_DELCHAR:
      return 1500;
    default:
      return 1000;
  }
}

void logGenImgResponse(const R307SResponseDiagnostics& diag, const char* reason) {
  static unsigned long lastLog = 0;
  unsigned long now = millis();
  if (lastLog != 0 && now - lastLog < 5000UL) {
    return;
  }
  lastLog = now;

  Serial.print("R307S GenImg: ");
  Serial.print(reason);
  Serial.print(", bytes=");
  Serial.print(diag.rxBytes);
  Serial.print(", header=");
  Serial.print(diag.headerDetected ? "yes" : "no");
  Serial.print(", header7=");
  Serial.print(diag.headerComplete ? "yes" : "no");
  Serial.print(", pid=");
  if (diag.packetInfoAvailable) {
    Serial.print("0x");
    Serial.print(diag.packetId, HEX);
  } else {
    Serial.print("n/a");
  }
  Serial.print(", length=");
  if (diag.packetInfoAvailable) {
    Serial.print(diag.length);
  } else {
    Serial.print("n/a");
  }
  Serial.print(", payload=");
  Serial.print(diag.payloadComplete ? "yes" : "no");
  Serial.print(", checksum=");
  if (!diag.checksumChecked) {
    Serial.print("skipped");
  } else {
    Serial.print(diag.checksumValid ? "valid" : "invalid");
  }
  Serial.print(", confirmation=0x");
  Serial.println(diag.confirmation, HEX);
}

bool r307sValidSlot(uint16_t slot) { return slot >= SLOT_MIN && slot <= SLOT_MAX; }

uint16_t r307sChecksum16(const uint8_t* data, size_t len) {
  uint16_t sum = 0;
  for (size_t i = 0; i < len; i++) {
    sum = static_cast<uint16_t>(sum + data[i]);
  }
  return sum;
}

// Read one response packet: 0xEF 0x01 | address(4) | package id | length(2) |
// confirmation | data | checksum(2). Data bytes are copied into `out`.
bool r307sReadResponse(uint8_t* out, size_t maxOut, size_t& outLen, bool diagnostics,
                       unsigned long timeoutMs) {
  outLen = 0;
  r307sConfirmation = R307S_NO_CONFIRMATION;
  R307SResponseDiagnostics diag;

  r307sSerial.setTimeout(timeoutMs);

  // Wait for the 0xEF 0x01 header.
  unsigned long start = millis();
  bool haveHeader = false;
  while (millis() - start < timeoutMs) {
    if (!r307sSerial.available()) {
      // Yield to the Wi-Fi/RTOS tasks instead of spinning the CPU.
      delay(1);
      continue;
    }
    uint8_t b = static_cast<uint8_t>(r307sSerial.read());
    diag.rxBytes++;
    if (b != 0xEF) {
      continue;
    }
    unsigned long inner = millis();
    while (millis() - inner < 100 && !r307sSerial.available()) {
      // brief wait for the second header byte
    }
    if (!r307sSerial.available()) {
      continue;
    }
    uint8_t second = static_cast<uint8_t>(r307sSerial.read());
    diag.rxBytes++;
    if (second == 0x01) {
      haveHeader = true;
      diag.headerDetected = true;
      break;
    }
  }
  if (!haveHeader) {
    if (diagnostics) {
      logGenImgResponse(diag, diag.rxBytes == 0 ? "timeout_no_bytes" : "timeout_no_header");
    }
    return false;
  }

  // Address (4) + package id (1) + length (2).
  uint8_t head[7];
  size_t headBytes = r307sSerial.readBytes(head, sizeof(head));
  diag.rxBytes += headBytes;
  if (headBytes != sizeof(head)) {
    if (diagnostics) {
      logGenImgResponse(diag, "header_incomplete");
    }
    return false;
  }
  diag.headerComplete = true;

  uint8_t packageId = head[4];
  uint16_t length = static_cast<uint16_t>((head[5] << 8) | head[6]);
  diag.packetInfoAvailable = true;
  diag.packetId = packageId;
  diag.length = length;
  if (length < 3 || length > 64) {
    if (diagnostics) {
      logGenImgResponse(diag, "malformed_length");
    }
    return false;
  }

  uint8_t payload[64];
  size_t payloadBytes = r307sSerial.readBytes(payload, length);
  diag.rxBytes += payloadBytes;
  if (payloadBytes != length) {
    if (diagnostics) {
      logGenImgResponse(diag, "payload_incomplete");
    }
    return false;
  }
  diag.payloadComplete = true;

  uint16_t expected = static_cast<uint16_t>((payload[length - 2] << 8) | payload[length - 1]);
  uint8_t checkBuf[64];
  size_t checkLen = 0;
  checkBuf[checkLen++] = packageId;
  checkBuf[checkLen++] = head[5];
  checkBuf[checkLen++] = head[6];
  for (size_t i = 0; i + 2 < length; i++) {
    checkBuf[checkLen++] = payload[i];
  }
  diag.confirmation = payload[0];
  diag.checksumChecked = true;
  diag.checksumValid = r307sChecksum16(checkBuf, checkLen) == expected;
  if (!diag.checksumValid) {
    if (diagnostics) {
      logGenImgResponse(diag, "checksum_mismatch");
    }
    return false;
  }

  uint8_t confirmation = payload[0];
  r307sConfirmation = confirmation;
  size_t dataLen = static_cast<size_t>(length) - 3; // minus confirmation + checksum
  if (out != nullptr && maxOut > 0) {
    size_t copy = dataLen < maxOut ? dataLen : maxOut;
    for (size_t i = 0; i < copy; i++) {
      out[i] = payload[1 + i];
    }
    outLen = copy;
  }
  if (diagnostics && confirmation != R307S_OK && confirmation != R307S_ERR_NOFINGER) {
    logGenImgResponse(diag, "sensor_error");
  }
  return confirmation == R307S_OK;
}

// Send one instruction packet and read its response.
bool r307sExecute(uint8_t command, const uint8_t* param, size_t paramLen, uint8_t* out,
                  size_t maxOut, size_t& outLen) {
  r307sConfirmation = R307S_NO_CONFIRMATION;

  // Command packet:
  // 0xEF 0x01 | 4-byte address | package id | length(2) | command | params | checksum(2)
  const size_t maxPacket = 64;
  if (paramLen + 12 > maxPacket) {
    return false;
  }
  uint8_t packet[maxPacket];
  size_t idx = 0;
  packet[idx++] = 0xEF;
  packet[idx++] = 0x01;
  packet[idx++] = 0xFF;
  packet[idx++] = 0xFF;
  packet[idx++] = 0xFF;
  packet[idx++] = 0xFF;
  packet[idx++] = 0x01;
  uint16_t length = static_cast<uint16_t>(paramLen + 3); // command + checksum
  packet[idx++] = static_cast<uint8_t>((length >> 8) & 0xFF);
  packet[idx++] = static_cast<uint8_t>(length & 0xFF);
  packet[idx++] = command;
  for (size_t i = 0; i < paramLen; i++) {
    packet[idx++] = param[i];
  }
  uint16_t sum = r307sChecksum16(packet + 6, idx - 6);
  packet[idx++] = static_cast<uint8_t>((sum >> 8) & 0xFF);
  packet[idx++] = static_cast<uint8_t>(sum & 0xFF);

  r307sSerial.write(packet, idx);
  r307sSerial.flush();

  return r307sReadResponse(out, maxOut, outLen, command == R307S_GENIMG,
                           r307sResponseTimeoutFor(command));
}

bool r307sVerifyPassword(uint32_t password) {
  uint8_t param[4] = {
      static_cast<uint8_t>((password >> 24) & 0xFF),
      static_cast<uint8_t>((password >> 16) & 0xFF),
      static_cast<uint8_t>((password >> 8) & 0xFF),
      static_cast<uint8_t>(password & 0xFF),
  };
  size_t outLen = 0;
  return r307sExecute(R307S_VERIFYPWD, param, sizeof(param), nullptr, 0, outLen);
}

// Start the UART at FINGERPRINT_BAUD and verify the module password.
bool r307sBegin() {
  r307sSerial.begin(FINGERPRINT_BAUD, SERIAL_8N1, LOGIN_RX, LOGIN_TX);
  delay(200);
  return r307sVerifyPassword(R307S_DEFAULT_PASSWORD);
}

// Capture an image from the sensor into the module's image buffer.
bool r307sGetImage() {
  size_t outLen = 0;
  return r307sExecute(R307S_GENIMG, nullptr, 0, nullptr, 0, outLen);
}

// Convert the stored image into character buffer 1 or 2.
bool r307sImg2tz(uint8_t bufferId) {
  uint8_t param[1] = {bufferId};
  size_t outLen = 0;
  return r307sExecute(R307S_IMG2TZ, param, sizeof(param), nullptr, 0, outLen);
}

// Combine character buffers 1 and 2 into one template.
bool r307sRegModel() {
  size_t outLen = 0;
  return r307sExecute(R307S_REGMODEL, nullptr, 0, nullptr, 0, outLen);
}

// Persist the template held in character buffer 1 into a module slot.
bool r307sStoreModel(uint16_t slot) {
  if (!r307sValidSlot(slot)) {
    return false;
  }
  uint8_t param[3] = {
      0x01, // source character buffer
      static_cast<uint8_t>((slot >> 8) & 0xFF),
      static_cast<uint8_t>(slot & 0xFF),
  };
  size_t outLen = 0;
  return r307sExecute(R307S_STORE, param, sizeof(param), nullptr, 0, outLen);
}

// Search a slot sub-range for a match with character buffer 1.
bool r307sSearchRange(uint16_t startSlot, uint16_t slotCount, uint16_t& slot) {
  slot = 0;
  if (!r307sValidSlot(startSlot) || slotCount == 0 || slotCount > SLOT_MAX ||
      static_cast<uint32_t>(startSlot) + slotCount > SLOT_MAX + 1) {
    return false;
  }
  uint8_t param[5] = {
      0x01, // character buffer 1
      static_cast<uint8_t>((startSlot >> 8) & 0xFF),
      static_cast<uint8_t>(startSlot & 0xFF),
      static_cast<uint8_t>((slotCount >> 8) & 0xFF),
      static_cast<uint8_t>(slotCount & 0xFF),
  };
  uint8_t data[4];
  size_t outLen = 0;
  bool ok = r307sExecute(R307S_SEARCH, param, sizeof(param), data, sizeof(data), outLen);
  if (outLen >= 2) {
    slot = static_cast<uint16_t>((data[0] << 8) | data[1]);
  }
  return ok && r307sValidSlot(slot);
}

// Search the whole library for a match with character buffer 1.
bool r307sSearch(uint16_t& slot) { return r307sSearchRange(SLOT_MIN, SLOT_MAX, slot); }

// Compare character buffers 1 and 2. Returns the match score.
bool r307sMatch(uint16_t& score) {
  score = 0;
  uint8_t data[2];
  size_t outLen = 0;
  bool ok = r307sExecute(R307S_MATCH, nullptr, 0, data, sizeof(data), outLen);
  if (outLen >= 2) {
    score = static_cast<uint16_t>((data[0] << 8) | data[1]);
  }
  return ok;
}

// Load a stored template into character buffer 1 or 2.
bool r307sLoadChar(uint8_t bufferId, uint16_t slot) {
  if (!r307sValidSlot(slot)) {
    return false;
  }
  uint8_t param[3] = {
      bufferId,
      static_cast<uint8_t>((slot >> 8) & 0xFF),
      static_cast<uint8_t>(slot & 0xFF),
  };
  size_t outLen = 0;
  return r307sExecute(R307S_LOADCHAR, param, sizeof(param), nullptr, 0, outLen);
}

// Check whether a slot holds a template. Returns false only when the sensor
// could not be queried reliably; a failed query is never treated as "empty".
bool r307sTemplateExists(uint16_t slot, bool& exists) {
  exists = false;
  if (!r307sValidSlot(slot)) {
    return false;
  }
  if (r307sLoadChar(2, slot)) {
    exists = true;
    return true;
  }
  if (r307sConfirmation == R307S_ERR_EMPTY || r307sConfirmation == R307S_ERR_NOTFOUND) {
    return true;
  }
  return false;
}

// Delete the template in a slot. An already empty slot counts as success.
bool r307sDeleteModel(uint16_t slot) {
  if (!r307sValidSlot(slot)) {
    return false;
  }
  uint8_t param[2] = {
      static_cast<uint8_t>((slot >> 8) & 0xFF),
      static_cast<uint8_t>(slot & 0xFF),
  };
  size_t outLen = 0;
  bool ok = r307sExecute(R307S_DELCHAR, param, sizeof(param), nullptr, 0, outLen);
  return ok || r307sConfirmation == R307S_ERR_EMPTY;
}

uint8_t r307sLastConfirmation() { return r307sConfirmation; }

// ===========================================================================
// 9. WI-FI
// ===========================================================================

bool isWifiConnected() { return WiFi.status() == WL_CONNECTED; }

void wifiBegin() {
  WiFi.mode(WIFI_STA);
  WiFi.setAutoReconnect(true);

  Serial.print("Connecting to Wi-Fi: ");
  Serial.println(WIFI_SSID);

  WiFi.begin(WIFI_SSID, WIFI_PASSWORD);
  lastWifiAttempt = millis();
  lastAuthAttempt = millis() - AUTH_RETRY_INTERVAL;
}

// Reconnect when disconnected, without blocking the main loop.
void wifiMaintain() {
  static bool connectionReported = false;

  if (WiFi.status() == WL_CONNECTED) {
    if (!connectionReported) {
      Serial.println("Wi-Fi connected");
      Serial.print("IP address: ");
      Serial.println(WiFi.localIP());
      connectionReported = true;
    }
    return;
  }

  connectionReported = false;

  if (millis() - lastWifiAttempt < WIFI_RETRY_INTERVAL) {
    return;
  }
  lastWifiAttempt = millis();
  Serial.print("Wi-Fi not connected, status ");
  Serial.print(WiFi.status());
  Serial.println("; retrying");
  WiFi.disconnect(false, false);
  WiFi.begin(WIFI_SSID, WIFI_PASSWORD);
}

// ===========================================================================
// 10. BACKEND DEVICE CLIENT
// ===========================================================================

bool isDeviceAuthenticated() { return deviceToken.length() > 0 && deviceId > 0; }

void invalidateDeviceToken() { deviceToken = ""; }

// Exchange DEVICE_API_KEY for a short-lived X-Device-Token.
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

  deviceId = doc["device"]["id"].as<int64_t>();
  deviceToken = doc["device"]["token"].as<String>();
  if (deviceToken.length() == 0 || deviceId <= 0) {
    deviceToken = "";
    deviceId = 0;
    return false;
  }

  Serial.print("Device authenticated, id=");
  Serial.println(static_cast<long>(deviceId));
  return true;
}

// Authenticated request helper. The token is refreshed when missing, and
// dropped when the backend rejects it, so the next call re-authenticates.
bool performDeviceRequest(const char* method, const String& path, const String& body,
                          String& response, int& httpCode, int timeoutMs) {
  response = "";
  httpCode = 0;
  if (!isWifiConnected()) {
    return false;
  }
  if (!isDeviceAuthenticated() && !authenticateDevice()) {
    return false;
  }

  HTTPClient http;
  http.setTimeout(timeoutMs);
  String url = String(API_BASE_URL) + path;
  if (!http.begin(url)) {
    http.end();
    return false;
  }
  http.addHeader("Content-Type", "application/json");
  http.addHeader("X-Device-Token", deviceToken);

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

bool parseEnrollmentResponse(const String& response, int httpCode, JsonDocument& doc,
                             EnrollmentSessionInfo& out, String* action) {
  if (httpCode <= 0) {
    return false;
  }
  if (deserializeJson(doc, response)) {
    return false;
  }
  if (action != nullptr) {
    *action = doc["action"].as<String>();
  }
  parseEnrollmentSession(doc["enrollment"], out);
  return true;
}

// POST /api/v1/biometric/readers. Reports the single R307S entry reader and its
// last confirmed capture readiness so the backend can offer it for enrollment.
bool reportDeviceReaders(bool connected, const char* readiness) {
  JsonDocument request;
  JsonArray readers = request["readers"].to<JsonArray>();
  JsonObject reader = readers.add<JsonObject>();
  reader["key"] = "login";
  reader["label"] = "Login reader";
  reader["uart"] = LOGIN_UART;
  reader["connected"] = connected;
  reader["readiness"] = readiness != nullptr ? readiness : "not_ready";

  String body;
  serializeJson(request, body);

  String response;
  int httpCode = 0;
  if (!performDeviceRequest("POST", "/api/v1/biometric/readers", body, response, httpCode, 8000)) {
    return false;
  }
  if (httpCode != HTTP_CODE_OK) {
    Serial.print("Reader report HTTP ");
    Serial.print(httpCode);
    Serial.print(": ");
    Serial.println(response);
    return false;
  }

  JsonDocument doc;
  if (deserializeJson(doc, response)) {
    return false;
  }
  return doc["success"].as<bool>();
}

// POST /api/v1/biometric/door-access. The backend synchronously maps the slot to
// a member, decides whether access is granted, and writes the access log. The
// door may only be unlocked when this returns true.
bool authorizeDoorAccess(uint16_t slot, const String& readerKey, bool& duplicate,
                         String& reason) {
  duplicate = false;
  reason = "";
  if (!r307sValidSlot(slot)) {
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
  if (!performDeviceRequest("POST", "/api/v1/biometric/door-access", body, response, httpCode,
                            8000)) {
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

// GET /api/v1/biometric/status. Keeps the device heartbeat alive and reports how
// many device events are still queued for backend processing.
bool fetchDeviceStatus(int& pendingEvents) {
  pendingEvents = 0;
  String response;
  int httpCode = 0;
  if (!performDeviceRequest("GET", "/api/v1/biometric/status", "", response, httpCode, 8000)) {
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

bool fetchEnrollmentCommand(EnrollmentSessionInfo& out, String& action, int& httpCode) {
  out = EnrollmentSessionInfo();
  action = "";
  String response;
  if (!performDeviceRequest("GET", "/api/v1/biometric/enrollment-session", "", response, httpCode,
                            ENROLLMENT_HTTP_TIMEOUT_MS)) {
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

bool fetchEnrollmentSession(const String& sessionUid, EnrollmentSessionInfo& out, String& action,
                            int& httpCode) {
  out = EnrollmentSessionInfo();
  action = "";
  if (sessionUid.length() == 0) {
    return false;
  }
  String response;
  String path = String("/api/v1/biometric/enrollment-sessions/") + sessionUid;
  if (!performDeviceRequest("GET", path, "", response, httpCode, ENROLLMENT_HTTP_TIMEOUT_MS)) {
    return false;
  }
  JsonDocument doc;
  if (!parseEnrollmentResponse(response, httpCode, doc, out, &action)) {
    return false;
  }
  return httpCode == HTTP_CODE_OK && doc["success"].as<bool>();
}

bool claimEnrollmentCommand(const String& sessionUid, EnrollmentSessionInfo& out, int& httpCode) {
  out = EnrollmentSessionInfo();
  if (sessionUid.length() == 0) {
    return false;
  }
  String response;
  String path = String("/api/v1/biometric/enrollment-sessions/") + sessionUid + "/claim";
  if (!performDeviceRequest("POST", path, "{}", response, httpCode, ENROLLMENT_HTTP_TIMEOUT_MS)) {
    return false;
  }
  JsonDocument doc;
  return parseEnrollmentResponse(response, httpCode, doc, out, nullptr);
}

bool updateEnrollmentProgress(const String& sessionUid, const String& status, int slot,
                              const String& errorCode, const String& errorMessage,
                              EnrollmentSessionInfo& out, int& httpCode) {
  out = EnrollmentSessionInfo();
  if (sessionUid.length() == 0 || status.length() == 0) {
    return false;
  }
  JsonDocument request;
  request["status"] = status;
  if (slot >= SLOT_MIN) {
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
  if (!performDeviceRequest("POST", path, body, response, httpCode, ENROLLMENT_HTTP_TIMEOUT_MS)) {
    return false;
  }
  JsonDocument doc;
  return parseEnrollmentResponse(response, httpCode, doc, out, nullptr);
}

bool completeEnrollment(const String& sessionUid, int slot, EnrollmentSessionInfo& out,
                        int& httpCode) {
  out = EnrollmentSessionInfo();
  if (sessionUid.length() == 0 || slot < SLOT_MIN) {
    return false;
  }
  JsonDocument request;
  request["fingerprint_slot"] = slot;
  request["device_user_id"] = String("slot:") + String(slot);
  String body;
  serializeJson(request, body);
  String response;
  String path = String("/api/v1/biometric/enrollment-sessions/") + sessionUid + "/complete";
  if (!performDeviceRequest("POST", path, body, response, httpCode, ENROLLMENT_HTTP_TIMEOUT_MS)) {
    return false;
  }
  JsonDocument doc;
  return parseEnrollmentResponse(response, httpCode, doc, out, nullptr);
}

bool failEnrollment(const String& sessionUid, int slot, const String& errorCode,
                    const String& errorMessage, EnrollmentSessionInfo& out, int& httpCode) {
  out = EnrollmentSessionInfo();
  if (sessionUid.length() == 0) {
    return false;
  }
  JsonDocument request;
  if (slot >= SLOT_MIN) {
    request["fingerprint_slot"] = slot;
  }
  request["error_code"] = errorCode;
  request["error_message"] = errorMessage;
  String body;
  serializeJson(request, body);
  String response;
  String path = String("/api/v1/biometric/enrollment-sessions/") + sessionUid + "/fail";
  if (!performDeviceRequest("POST", path, body, response, httpCode, ENROLLMENT_HTTP_TIMEOUT_MS)) {
    return false;
  }
  JsonDocument doc;
  return parseEnrollmentResponse(response, httpCode, doc, out, nullptr);
}

bool acknowledgeEnrollmentCleanup(const String& sessionUid, int slot,
                                  EnrollmentSessionInfo& out, int& httpCode) {
  out = EnrollmentSessionInfo();
  if (sessionUid.length() == 0) {
    return false;
  }
  JsonDocument request;
  if (slot >= SLOT_MIN) {
    request["fingerprint_slot"] = slot;
  }
  String body;
  serializeJson(request, body);
  String response;
  String path = String("/api/v1/biometric/enrollment-sessions/") + sessionUid + "/cleanup";
  if (!performDeviceRequest("POST", path, body, response, httpCode, ENROLLMENT_HTTP_TIMEOUT_MS)) {
    return false;
  }
  JsonDocument doc;
  return parseEnrollmentResponse(response, httpCode, doc, out, nullptr);
}

// ===========================================================================
// 11. INDICATORS AND LOCK
// ===========================================================================

void setLEDStatus(bool green, bool red, bool buzzer) {
  digitalWrite(GREEN_LED, green ? HIGH : LOW);
  digitalWrite(RED_LED, red ? HIGH : LOW);
  digitalWrite(BUZZER_PIN, buzzer ? HIGH : LOW);
}

// 1 = acknowledgement, 2 = success (two beeps), 3 = error (three beeps).
void buzzPattern(int pattern) {
  int beeps = pattern == 2 ? 2 : (pattern == 3 ? 3 : 1);
  for (int i = 0; i < beeps; i++) {
    digitalWrite(BUZZER_PIN, HIGH);
    delay(60);
    digitalWrite(BUZZER_PIN, LOW);
    if (i + 1 < beeps) {
      delay(40);
    }
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

// Re-lock automatically after DOOR_OPEN_MS. Called on every loop iteration so a
// blocking backend call can never leave the door open.
void updateDoor() {
  if (doorUnlocked && millis() - doorOpenedAt >= DOOR_OPEN_MS) {
    lockDoor();
  }
}

// Exit push button on GPIO33. Releases the lock locally, independent of Wi-Fi
// and the backend. This is an exit request, not an emergency button and not a
// member logout: no biometric event is sent.
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
  } else if (reading == HIGH) {
    handled = false;
  }
}

// ===========================================================================
// 12. READER READINESS REPORTING
// ===========================================================================

const char* readerReadinessStatus(ReaderReadiness readiness) {
  switch (readiness) {
    case ReaderReadiness::NOT_READY:
      return "not_ready";
    case ReaderReadiness::INITIALIZING:
      return "initializing";
    case ReaderReadiness::READY:
      return "ready";
    case ReaderReadiness::WAITING_FOR_FINGER:
      return "waiting_for_finger";
    case ReaderReadiness::FINGER_DETECTED:
      return "finger_detected";
    case ReaderReadiness::CAPTURE_FAILED:
      return "capture_failed";
    case ReaderReadiness::ENROLLING:
      return "enrolling";
  }
  return "not_ready";
}

bool readerReadinessIsConnected(ReaderReadiness readiness) {
  switch (readiness) {
    case ReaderReadiness::INITIALIZING:
    case ReaderReadiness::READY:
    case ReaderReadiness::WAITING_FOR_FINGER:
    case ReaderReadiness::FINGER_DETECTED:
    case ReaderReadiness::ENROLLING:
      return true;
    case ReaderReadiness::NOT_READY:
    case ReaderReadiness::CAPTURE_FAILED:
      return false;
  }
  return false;
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
  if (captured || confirmation == R307S_OK) {
    setReaderReadiness(ReaderReadiness::FINGER_DETECTED);
    return;
  }
  if (confirmation == R307S_ERR_NOFINGER) {
    setReaderReadiness(ReaderReadiness::WAITING_FOR_FINGER);
    return;
  }
  // Any other confirmation, or no answer at all, means capture is unavailable.
  setReaderReadiness(ReaderReadiness::CAPTURE_FAILED);
}

void reportReaderIfDue(bool force) {
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

// ===========================================================================
// 13. ENROLLMENT RECOVERY STATE (NVS)
// ===========================================================================

bool hasPersistedEnrollment() {
  return persistedEnrollmentUid.length() > 0 && persistedEnrollmentSlot >= SLOT_MIN;
}

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
  if (persistedEnrollmentSlot > SLOT_MAX) {
    persistedEnrollmentSlot = 0;
  }
  if (persistedEnrollmentUid.length() > 0 && persistedEnrollmentSlot >= SLOT_MIN) {
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

bool savePersistedEnrollment(const String& sessionUid, uint16_t slot, const String& action,
                             const String& errorCode, const String& errorMessage) {
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

// ===========================================================================
// 14. LOGIN: LOCAL MATCH, BACKEND DECISION
// ===========================================================================

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

// Convert the captured image and search the module library. On success the
// caller only learns the slot number: the member identity lives in the backend.
bool processCapturedFingerprint(uint16_t& slot) {
  slot = 0;
  if (!r307sImg2tz(1)) {
    setReaderReadiness(ReaderReadiness::CAPTURE_FAILED);
    logLoginScanIssue("image conversion failed", r307sLastConfirmation());
    return false;
  }
  if (!r307sSearch(slot)) {
    if (r307sLastConfirmation() == R307S_ERR_NOTFOUND) {
      logLoginScanIssue("unregistered fingerprint", R307S_ERR_NOTFOUND);
      setReaderReadiness(ReaderReadiness::READY);
    } else {
      logLoginScanIssue("template search failed", r307sLastConfirmation());
      setReaderReadiness(ReaderReadiness::CAPTURE_FAILED);
    }
    return false;
  }
  return true;
}

void processLoginCapture() {
  uint16_t slot = 0;
  if (!processCapturedFingerprint(slot)) {
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
    // The backend has already recorded the denied attempt with member details.
  }
  setReaderReadiness(ReaderReadiness::READY);
}

void pollReaderForLogin() {
  // Suspend login scanning whenever enrollment owns the sensor.
  if (currentMode == SensorMode::ENROLLMENT || currentMode == SensorMode::ENROLLMENT_CLEANUP) {
    return;
  }
  if (sensorResumeAt != 0) {
    if (static_cast<long>(millis() - sensorResumeAt) < 0) {
      return;
    }
    sensorResumeAt = 0;
  }
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
  if (!r307sGetImage()) {
    uint8_t confirmation = r307sLastConfirmation();
    observeReaderImage(false, confirmation);
    if (confirmation != R307S_ERR_NOFINGER) {
      logLoginScanIssue("image capture failed", confirmation);
    }
    return;
  }

  setReaderReadiness(ReaderReadiness::FINGER_DETECTED);
  processLoginCapture();
}

// ===========================================================================
// 15. ENROLLMENT: ADMIN SESSION DRIVEN TWO-SCAN FLOW
// ===========================================================================

Stage stageNext(Stage stage, Event event) {
  if (event == Event::Failed) {
    return stage == Stage::Idle ? Stage::Idle : Stage::Failed;
  }
  if (event == Event::Reset) {
    return Stage::Idle;
  }
  switch (stage) {
    case Stage::Idle:
      return event == Event::Start ? Stage::ClaimingSession : Stage::Idle;
    case Stage::ClaimingSession:
      return event == Event::CommandClaimed ? Stage::SensorInitialization : stage;
    case Stage::SensorInitialization:
      if (event == Event::FirstFingerDetected) {
        return Stage::CapturingFirstImage;
      }
      return event == Event::SensorReady ? Stage::WaitingForFirstFinger : stage;
    case Stage::WaitingForFirstFinger:
      return event == Event::FirstFingerDetected ? Stage::CapturingFirstImage : stage;
    case Stage::CapturingFirstImage:
      return event == Event::FirstImageCaptured ? Stage::ConvertingFirstImage : stage;
    case Stage::ConvertingFirstImage:
      return event == Event::FirstImageConverted ? Stage::WaitingForFingerRemoval : stage;
    case Stage::WaitingForFingerRemoval:
      return event == Event::FingerRemoved ? Stage::WaitingForSecondFinger : stage;
    case Stage::WaitingForSecondFinger:
      return event == Event::SecondFingerDetected ? Stage::CapturingSecondImage : stage;
    case Stage::CapturingSecondImage:
      return event == Event::SecondImageCaptured ? Stage::ConvertingSecondImage : stage;
    case Stage::ConvertingSecondImage:
      return event == Event::SecondImageConverted ? Stage::CreatingModel : stage;
    case Stage::CreatingModel:
      return event == Event::ModelCreated ? Stage::StoringTemplate : stage;
    case Stage::StoringTemplate:
      return event == Event::TemplateStored ? Stage::BackendFinalization : stage;
    case Stage::BackendFinalization:
      if (event == Event::CleanupRequired) {
        return Stage::EnrollmentCleanup;
      }
      return event == Event::BackendFinalized ? Stage::Completed : stage;
    case Stage::EnrollmentCleanup:
      return event == Event::CleanupCompleted ? Stage::Failed : stage;
    case Stage::Completed:
    case Stage::Failed:
      return stage;
  }
  return stage;
}

const char* stageName(Stage stage) {
  switch (stage) {
    case Stage::Idle:
      return "idle";
    case Stage::ClaimingSession:
      return "claiming_session";
    case Stage::SensorInitialization:
      return "sensor_initialization";
    case Stage::WaitingForFirstFinger:
      return "waiting_for_first_finger";
    case Stage::CapturingFirstImage:
      return "capturing_first_image";
    case Stage::ConvertingFirstImage:
      return "converting_first_image";
    case Stage::WaitingForFingerRemoval:
      return "waiting_for_finger_removal";
    case Stage::WaitingForSecondFinger:
      return "waiting_for_second_finger";
    case Stage::CapturingSecondImage:
      return "capturing_second_image";
    case Stage::ConvertingSecondImage:
      return "converting_second_image";
    case Stage::CreatingModel:
      return "creating_model";
    case Stage::StoringTemplate:
      return "storing_template";
    case Stage::BackendFinalization:
      return "backend_finalization";
    case Stage::EnrollmentCleanup:
      return "enrollment_cleanup";
    case Stage::Completed:
      return "completed";
    case Stage::Failed:
      return "failed";
  }
  return "unknown";
}

const char* stageProgressStatus(Stage stage) {
  switch (stage) {
    case Stage::SensorInitialization:
      return "sensor_initializing";
    case Stage::WaitingForFirstFinger:
      return "waiting_first_scan";
    case Stage::CapturingFirstImage:
      return "waiting_first_scan";
    case Stage::ConvertingFirstImage:
      return "first_scan_captured";
    case Stage::WaitingForFingerRemoval:
      return "waiting_for_removal";
    case Stage::WaitingForSecondFinger:
      return "waiting_second_scan";
    case Stage::CapturingSecondImage:
      return "waiting_second_scan";
    case Stage::ConvertingSecondImage:
      return "second_scan_captured";
    case Stage::CreatingModel:
      return "verifying";
    case Stage::StoringTemplate:
      return "verifying";
    case Stage::BackendFinalization:
      return "verifying";
    case Stage::Completed:
      return "finalized";
    case Stage::Failed:
      return "failed";
    case Stage::EnrollmentCleanup:
      return "waiting_for_removal";
    case Stage::Idle:
    case Stage::ClaimingSession:
      return "";
  }
  return "";
}

bool deleteEnrollmentSlot(uint16_t slot) {
  if (!r307sValidSlot(slot)) {
    return true;
  }
  if (!r307sDeleteModel(slot)) {
    Serial.print("Enrollment cleanup failed, sensor confirmation ");
    Serial.println(r307sLastConfirmation());
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
    Serial.println("Enrollment: registration successful");
    setReaderReadiness(ReaderReadiness::READY);
  } else {
    errorSignal();
    Serial.print("Enrollment: registration failed - ");
    Serial.println(reason.length() > 0 ? reason : String("unknown reason"));
  }
  activeEnrollment = ActiveEnrollment();
  currentMode = SensorMode::IDLE;
  lastEnrollmentPoll = millis();
  enrollmentPollInterval = ENROLLMENT_POLL_INTERVAL;
  sensorResumeAt = millis() + ENROLLMENT_RESUME_COOLDOWN_MS;
  Serial.println("Sensor ownership: IDLE (login scanning resumes after cooldown)");
}

// Keeps exclusive sensor ownership while a template still has to be removed.
// Login scanning must not resume until cleanup succeeds, so an untracked
// template can never be left behind in the sensor.
void enterEnrollmentCleanup(const String& reason) {
  activeEnrollment.active = true;
  activeEnrollment.cleanupPending = true;
  activeEnrollment.terminalReason = reason;
  activeEnrollment.stage = Stage::EnrollmentCleanup;
  currentMode = SensorMode::ENROLLMENT_CLEANUP;
  Serial.println(
      "Enrollment: cleanup required; login scanning stays suspended until the sensor slot is removed");
}

void handleTerminalEnrollment(const EnrollmentSessionInfo& info, bool deleteLocalTemplate) {
  if (info.status == "finalized") {
    clearPersistedEnrollment();
    stopEnrollment(true, "finalized");
    return;
  }

  bool shouldDelete = deleteLocalTemplate || info.cleanupRequired;

  uint16_t slot = activeEnrollment.slotStored ? activeEnrollment.slot : persistedEnrollmentSlot;
  String sessionUid = activeEnrollment.sessionUid.length() > 0 ? activeEnrollment.sessionUid
                                                              : persistedEnrollmentUid;

  String reason = info.status.length() > 0 ? info.status : String("rejected");
  if (info.errorCode.length() > 0) {
    reason += " (";
    reason += info.errorCode;
    reason += ")";
  }
  if (info.errorMessage.length() > 0) {
    reason += ": ";
    reason += info.errorMessage;
  }

  if (shouldDelete && slot >= SLOT_MIN) {
    if (!deleteEnrollmentSlot(slot)) {
      // The template is still on the sensor. Keep exclusive ownership and
      // retry instead of silently returning to login scanning.
      activeEnrollment.slot = slot;
      activeEnrollment.slotStored = true;
      enterEnrollmentCleanup(reason);
      return;
    }
    if (info.cleanupRequired) {
      acknowledgeCleanup(sessionUid, slot);
    }
  }

  clearPersistedEnrollment();
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
    if (info.active && info.status != "failed" && info.status != "cancelled" &&
        info.status != "expired" && !info.cleanupRequired) {
      failActiveEnrollment("backend_conflict", "Backend rejected the enrollment progress");
    } else {
      handleTerminalEnrollment(info, true);
    }
  }
  return false;
}

// The fingerprint only counts as enrolled once the backend confirmed the slot
// mapping for this session. Until then the local template is provisional.
void attemptEnrollmentCompletion() {
  if (!activeEnrollment.active || activeEnrollment.stage != Stage::BackendFinalization) {
    return;
  }
  if (millis() - activeEnrollment.lastCompleteAttempt < ENROLLMENT_COMPLETE_RETRY_INTERVAL) {
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
    stopEnrollment(true, "finalized");
    return;
  }
  if (httpCode == HTTP_CODE_UNAUTHORIZED || httpCode == HTTP_CODE_FORBIDDEN) {
    return;
  }
  if (httpCode == HTTP_CODE_CONFLICT) {
    if (info.valid && info.active && info.status != "failed" && info.status != "cancelled" &&
        info.status != "expired" && !info.cleanupRequired) {
      failActiveEnrollment("backend_conflict", "Backend rejected the enrollment completion");
    } else if (info.valid) {
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
  bool reported =
      failEnrollment(activeEnrollment.sessionUid, slot, errorCode, message, info, httpCode);
  bool cleared = true;
  if (activeEnrollment.slotStored) {
    cleared = deleteEnrollmentSlot(activeEnrollment.slot);
    if (cleared && reported &&
        (httpCode == HTTP_CODE_OK || (httpCode == HTTP_CODE_CONFLICT && info.cleanupRequired))) {
      acknowledgeCleanup(activeEnrollment.sessionUid, activeEnrollment.slot);
    }
  }
  if (!cleared) {
    // Keep the recovery state and exclusive ownership; the template must be
    // removed before login scanning can safely resume.
    enterEnrollmentCleanup(errorCode + ": " + message);
    return;
  }
  if (!failureStateSaved) {
    Serial.println("Enrollment failure state could not be persisted; continuing with backend state");
  }
  clearPersistedEnrollment();
  stopEnrollment(false, errorCode + ": " + message);
}

// Refuse a finger that already has a template in another slot. Returns false
// only when the sensor could not answer reliably.
bool detectDuplicateFinger(uint16_t ownSlot, bool& duplicate) {
  duplicate = false;
  uint16_t foundSlot = 0;
  if (ownSlot > SLOT_MIN) {
    if (r307sSearchRange(SLOT_MIN, static_cast<uint16_t>(ownSlot - 1), foundSlot)) {
      duplicate = true;
      return true;
    }
    if (r307sLastConfirmation() != R307S_ERR_NOTFOUND) {
      return false;
    }
  }
  if (ownSlot < SLOT_MAX) {
    if (r307sSearchRange(static_cast<uint16_t>(ownSlot + 1),
                         static_cast<uint16_t>(SLOT_MAX - ownSlot), foundSlot)) {
      duplicate = true;
      return true;
    }
    if (r307sLastConfirmation() != R307S_ERR_NOTFOUND) {
      return false;
    }
  }
  return true;
}

void advanceActiveStage(Event event) {
  Stage next = stageNext(activeEnrollment.stage, event);
  if (next == activeEnrollment.stage) {
    return;
  }
  activeEnrollment.stage = next;
  Serial.print("Enrollment stage: ");
  Serial.println(stageName(activeEnrollment.stage));
}

// Report the stage to the backend so the admin page shows real progress.
void logStageProgress() {
  const char* status = stageProgressStatus(activeEnrollment.stage);
  if (status != nullptr && status[0] != '\0') {
    reportActiveProgress(status, 0);
  }
}

bool sensorActionDue(unsigned long interval) {
  if (millis() - activeEnrollment.lastScanAt < interval) {
    return false;
  }
  activeEnrollment.lastScanAt = millis();
  return true;
}

bool handleEnrollmentScanError(const String& message) {
  activeEnrollment.scanErrors++;
  setReaderReadiness(ReaderReadiness::CAPTURE_FAILED);
  if (activeEnrollment.scanErrors >= 3) {
    failActiveEnrollment("sensor_error", message);
    return true;
  }
  return false;
}

void processActiveEnrollment() {
  if (!activeEnrollment.active) {
    if (currentMode != SensorMode::IDLE) {
      currentMode = SensorMode::IDLE;
      sensorResumeAt = millis() + ENROLLMENT_RESUME_COOLDOWN_MS;
      Serial.println("Sensor ownership: IDLE (login scanning resumes after cooldown)");
    }
    return;
  }
  if (sensorActivity != SensorActivity::NONE) {
    return;
  }
  SensorActivityGuard guard(SensorActivity::ENROLLMENT);

  if (activeEnrollment.cleanupPending) {
    if (millis() - activeEnrollment.lastScanAt < ENROLLMENT_CLEANUP_RETRY_INTERVAL) {
      return;
    }
    activeEnrollment.lastScanAt = millis();
    uint16_t cleanupSlot =
        activeEnrollment.slotStored ? activeEnrollment.slot : persistedEnrollmentSlot;
    if (cleanupSlot >= SLOT_MIN && !deleteEnrollmentSlot(cleanupSlot)) {
      Serial.println("Enrollment: cleanup retry failed; sensor ownership retained");
      return;
    }
    if (activeEnrollment.sessionUid.length() > 0 &&
        !acknowledgeCleanup(activeEnrollment.sessionUid, cleanupSlot)) {
      Serial.println("Enrollment: cleanup acknowledgement failed; will retry");
      return;
    }
    clearPersistedEnrollment();
    String reason = activeEnrollment.terminalReason;
    activeEnrollment.cleanupPending = false;
    stopEnrollment(false, reason);
    return;
  }

  if (millis() - activeEnrollment.lastStatusCheck >= ENROLLMENT_RECONCILE_INTERVAL) {
    activeEnrollment.lastStatusCheck = millis();
    reconcileActiveEnrollment();
    if (!activeEnrollment.active) {
      return;
    }
  }

  if (activeEnrollment.stage == Stage::BackendFinalization) {
    attemptEnrollmentCompletion();
    return;
  }

  switch (activeEnrollment.stage) {
    case Stage::ClaimingSession: {
      if (millis() - activeEnrollment.lastClaimAttempt < ENROLLMENT_CLAIM_RETRY_INTERVAL) {
        return;
      }
      activeEnrollment.lastClaimAttempt = millis();
      EnrollmentSessionInfo claimed;
      int httpCode = 0;
      if (!claimEnrollmentCommand(activeEnrollment.sessionUid, claimed, httpCode)) {
        Serial.print("Enrollment: command claim failed HTTP ");
        Serial.println(httpCode);
        return;
      }
      if (httpCode != HTTP_CODE_OK) {
        if (claimed.valid) {
          handleTerminalEnrollment(claimed, true);
        } else {
          Serial.print("Enrollment: command claim rejected HTTP ");
          Serial.println(httpCode);
        }
        return;
      }
      Serial.print("Enrollment: command claim acknowledged (HTTP ");
      Serial.print(httpCode);
      Serial.print(", session=");
      Serial.print(claimed.sessionUid);
      Serial.println(")");

      // The claim response must not silently change the slot allocated when
      // this session was queued.
      if (claimed.slot >= SLOT_MIN && claimed.slot <= SLOT_MAX &&
          static_cast<uint16_t>(claimed.slot) != activeEnrollment.slot) {
        failActiveEnrollment("slot_mismatch",
                             "The claimed session returned a different fingerprint slot");
        return;
      }

      advanceActiveStage(Event::CommandClaimed);
      if (activeEnrollment.slotStored) {
        // Recovery path: the allocated slot already contains this session's
        // template. Claim first, then ask the backend to finalize it.
        activeEnrollment.stage = Stage::BackendFinalization;
        activeEnrollment.lastCompleteAttempt = 0;
        Serial.println(
            "Enrollment: existing allocated template found; confirming registration with the backend");
        reportActiveProgress(stageProgressStatus(activeEnrollment.stage), activeEnrollment.slot);
        return;
      }

      Serial.println("Enrollment: command received from backend; initializing reader");
      setReaderReadiness(ReaderReadiness::INITIALIZING);
      reportActiveProgress("sensor_initializing", 0);
      return;
    }
    case Stage::SensorInitialization: {
      if (!sensorActionDue(ENROLLMENT_PREFLIGHT_INTERVAL)) {
        return;
      }
      if (r307sGetImage()) {
        activeEnrollment.scanErrors = 0;
        setReaderReadiness(ReaderReadiness::FINGER_DETECTED);
        advanceActiveStage(Event::FirstFingerDetected);
        return;
      }
      uint8_t confirmation = r307sLastConfirmation();
      if (confirmation == R307S_ERR_NOFINGER) {
        activeEnrollment.scanErrors = 0;
        setReaderReadiness(ReaderReadiness::WAITING_FOR_FINGER);
        advanceActiveStage(Event::SensorReady);
        Serial.println("Enrollment: reader initialized; place the finger for the first scan");
        logStageProgress();
        return;
      }
      handleEnrollmentScanError("Enrollment reader is not ready");
      return;
    }
    case Stage::WaitingForFirstFinger: {
      if (!sensorActionDue(ENROLLMENT_SCAN_INTERVAL)) {
        return;
      }
      if (!r307sGetImage()) {
        if (r307sLastConfirmation() == R307S_ERR_NOFINGER) {
          activeEnrollment.scanErrors = 0;
          setReaderReadiness(ReaderReadiness::WAITING_FOR_FINGER);
          return;
        }
        handleEnrollmentScanError("Unable to capture the first fingerprint image");
        return;
      }
      activeEnrollment.scanErrors = 0;
      setReaderReadiness(ReaderReadiness::FINGER_DETECTED);
      advanceActiveStage(Event::FirstFingerDetected);
      return;
    }
    case Stage::CapturingFirstImage: {
      advanceActiveStage(Event::FirstImageCaptured);
      return;
    }
    case Stage::ConvertingFirstImage: {
      if (!sensorActionDue(ENROLLMENT_SCAN_INTERVAL)) {
        return;
      }
      if (!r307sImg2tz(1)) {
        setReaderReadiness(ReaderReadiness::CAPTURE_FAILED);
        failActiveEnrollment("first_scan_failed", "Unable to process the first fingerprint image");
        return;
      }
      reportActiveProgress("first_scan_captured", 0);
      Serial.println("Enrollment: first fingerprint captured; remove the finger");
      advanceActiveStage(Event::FirstImageConverted);
      buzzPattern(1);
      logStageProgress();
      return;
    }
    case Stage::WaitingForFingerRemoval: {
      if (!sensorActionDue(ENROLLMENT_SCAN_INTERVAL)) {
        return;
      }
      if (r307sGetImage()) {
        setReaderReadiness(ReaderReadiness::FINGER_DETECTED);
        return;
      }
      if (r307sLastConfirmation() != R307S_ERR_NOFINGER) {
        handleEnrollmentScanError("Finger removal could not be detected");
        return;
      }
      activeEnrollment.scanErrors = 0;
      setReaderReadiness(ReaderReadiness::WAITING_FOR_FINGER);
      advanceActiveStage(Event::FingerRemoved);
      Serial.println("Enrollment: finger removed; place the same finger again");
      buzzPattern(1);
      logStageProgress();
      return;
    }
    case Stage::WaitingForSecondFinger: {
      if (!sensorActionDue(ENROLLMENT_SCAN_INTERVAL)) {
        return;
      }
      if (!r307sGetImage()) {
        if (r307sLastConfirmation() == R307S_ERR_NOFINGER) {
          activeEnrollment.scanErrors = 0;
          setReaderReadiness(ReaderReadiness::WAITING_FOR_FINGER);
          return;
        }
        handleEnrollmentScanError("Unable to capture the second fingerprint image");
        return;
      }
      activeEnrollment.scanErrors = 0;
      setReaderReadiness(ReaderReadiness::FINGER_DETECTED);
      advanceActiveStage(Event::SecondFingerDetected);
      return;
    }
    case Stage::CapturingSecondImage: {
      advanceActiveStage(Event::SecondImageCaptured);
      return;
    }
    case Stage::ConvertingSecondImage: {
      if (!sensorActionDue(ENROLLMENT_SCAN_INTERVAL)) {
        return;
      }
      if (!r307sImg2tz(2)) {
        setReaderReadiness(ReaderReadiness::CAPTURE_FAILED);
        failActiveEnrollment("second_scan_failed", "Unable to process the second fingerprint image");
        return;
      }
      reportActiveProgress("second_scan_captured", 0);
      Serial.println("Enrollment: second fingerprint captured; matching the two samples");
      advanceActiveStage(Event::SecondImageConverted);
      return;
    }
    case Stage::CreatingModel: {
      uint16_t score = 0;
      if (!r307sMatch(score) || score < 50) {
        failActiveEnrollment("finger_mismatch", "The two scans did not match the same finger");
        return;
      }
      Serial.print("Enrollment: samples matched with score ");
      Serial.println(static_cast<unsigned>(score));
      if (!r307sRegModel()) {
        failActiveEnrollment("template_merge_failed", "Unable to create a fingerprint template");
        return;
      }
      advanceActiveStage(Event::ModelCreated);
      return;
    }
    case Stage::StoringTemplate: {
      // The backend allocates the slot for this session. Never choose a local
      // free slot: doing so can create a backend/sensor slot mismatch.
      const uint16_t slot = activeEnrollment.slot;
      if (!r307sValidSlot(slot)) {
        failActiveEnrollment("invalid_slot",
                             "The backend did not allocate a valid fingerprint slot");
        return;
      }

      // Refuse to overwrite an existing template. A failed query is not the
      // same as an empty slot, so fail safely when the sensor cannot confirm.
      bool alreadyExists = false;
      if (!r307sTemplateExists(slot, alreadyExists)) {
        failActiveEnrollment("slot_check_failed", "Unable to verify the allocated fingerprint slot");
        return;
      }
      if (alreadyExists) {
        failActiveEnrollment("slot_conflict",
                             "The backend-allocated fingerprint slot is already occupied");
        return;
      }

      if (!savePersistedEnrollment(activeEnrollment.sessionUid, slot, "complete", "", "")) {
        failActiveEnrollment("recovery_state_failed", "Unable to persist enrollment recovery state");
        return;
      }

      // From here the slot belongs to this active backend session. Mark it for
      // cleanup before issuing Store, because a failed UART response can leave
      // the sensor's flash write outcome uncertain.
      activeEnrollment.slotStored = true;
      Serial.print("Enrollment: storing template in backend-allocated sensor slot ");
      Serial.println(static_cast<unsigned>(slot));
      if (!r307sStoreModel(slot)) {
        failActiveEnrollment("sensor_store_failed", "Unable to store the fingerprint template");
        return;
      }
      Serial.print("Enrollment: template stored in slot ");
      Serial.println(static_cast<unsigned>(slot));

      bool duplicate = false;
      if (!detectDuplicateFinger(slot, duplicate)) {
        failActiveEnrollment("duplicate_check_failed", "Unable to check for duplicate fingerprints");
        return;
      }
      if (duplicate) {
        failActiveEnrollment("duplicate_fingerprint", "This fingerprint is already enrolled");
        return;
      }

      advanceActiveStage(Event::TemplateStored);
      activeEnrollment.lastCompleteAttempt = 0;
      Serial.println("Enrollment: template stored; confirming registration with the backend");
      reportActiveProgress(stageProgressStatus(activeEnrollment.stage), slot);
      attemptEnrollmentCompletion();
      return;
    }
    case Stage::BackendFinalization: {
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
  activeEnrollment.lastScanAt = millis() - ENROLLMENT_PREFLIGHT_INTERVAL;
  activeEnrollment.lastStatusCheck = millis();
  currentMode = SensorMode::ENROLLMENT;
  setLEDStatus(true, false, false);
  setReaderReadiness(ReaderReadiness::ENROLLING);
  advanceActiveStage(Event::Start);

  Serial.println("Sensor ownership: ENROLLMENT (login scanning suspended)");
  Serial.print("Enrollment: command queued for ");
  Serial.print(info.memberName.length() > 0 ? info.memberName : String("member"));
  Serial.print(", session ");
  Serial.println(info.sessionUid);

  // The backend owns slot allocation. Do not start the session unless it
  // supplied a valid R307S slot in the supported range.
  if (info.slot < SLOT_MIN || info.slot > SLOT_MAX) {
    failActiveEnrollment("invalid_slot",
                         "The backend enrollment session has no valid fingerprint slot");
    return;
  }
  activeEnrollment.slot = static_cast<uint16_t>(info.slot);

  // If this is a retry after a power interruption, the backend-allocated slot
  // may already contain the template. Verify that state, but still claim the
  // session before asking the backend to finalize it.
  bool exists = false;
  if (!r307sTemplateExists(activeEnrollment.slot, exists)) {
    failActiveEnrollment("slot_check_failed",
                         "Unable to inspect the backend-allocated fingerprint slot");
    return;
  }
  if (exists) {
    activeEnrollment.slotStored = true;
    if (!savePersistedEnrollment(activeEnrollment.sessionUid, activeEnrollment.slot, "complete", "",
                                 "")) {
      failActiveEnrollment("recovery_state_failed", "Unable to persist enrollment recovery state");
      return;
    }
    Serial.println(
        "Enrollment: existing allocated template found; claiming session before finalization");
  } else {
    Serial.println("Enrollment: acknowledging enrollment command");
  }
}

// Power-loss recovery: reconcile the single provisional template left in NVS
// with the backend before login scanning is allowed to resume.
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
    if (!r307sTemplateExists(persistedEnrollmentSlot, exists)) {
      return false;
    }
    if (!exists) {
      EnrollmentSessionInfo ignored;
      int failCode = 0;
      failEnrollment(persistedEnrollmentUid, 0, "local_template_missing",
                     "Stored fingerprint template is missing from the sensor", ignored, failCode);
      clearPersistedEnrollment();
      return true;
    }

    if (persistedEnrollmentAction == "fail") {
      String errorCode =
          persistedFailureCode.length() > 0 ? persistedFailureCode : String("device_recovery");
      String errorMessage =
          persistedFailureMessage.length() > 0
              ? persistedFailureMessage
              : String("Enrollment was interrupted on the device");
      EnrollmentSessionInfo failed;
      int failCode = 0;
      bool reported = failEnrollment(persistedEnrollmentUid, persistedEnrollmentSlot, errorCode,
                                     errorMessage, failed, failCode);
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
    activeEnrollment.stage = Stage::BackendFinalization;
    activeEnrollment.lastStatusCheck = millis();
    currentMode = SensorMode::ENROLLMENT;
    setLEDStatus(true, false, false);
    setReaderReadiness(ReaderReadiness::ENROLLING);
    Serial.println("Sensor ownership: ENROLLMENT (login scanning suspended)");
    reportActiveProgress(stageProgressStatus(activeEnrollment.stage), activeEnrollment.slot);
    return true;
  }

  if (!info.valid) {
    return false;
  }

  if (!deleteEnrollmentSlot(persistedEnrollmentSlot)) {
    return false;
  }
  clearPersistedEnrollment();
  return true;
}

// Poll the backend for admin-created enrollment work. This runs before login
// scanning so a finger presented for enrollment is never consumed as a login.
void processEnrollmentCommand() {
  if (currentMode != SensorMode::IDLE) {
    return;
  }
  if (!isWifiConnected() || !isDeviceAuthenticated()) {
    return;
  }
  if (millis() - lastEnrollmentPoll < enrollmentPollInterval) {
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
    if (httpCode > 0 && httpCode != HTTP_CODE_OK) {
      Serial.print("Enrollment poll failed HTTP ");
      Serial.println(httpCode);
    }
    return;
  }
  // Poll quickly only while there is work; back off when the queue is empty.
  enrollmentPollInterval =
      action == "none" ? ENROLLMENT_IDLE_POLL_INTERVAL : ENROLLMENT_POLL_INTERVAL;
  if (action != "none") {
    Serial.print("Enrollment poll action: ");
    Serial.print(action);
    Serial.print(" (HTTP ");
    Serial.print(httpCode);
    Serial.print(", status=");
    Serial.print(info.valid ? info.status : "invalid");
    Serial.println(")");
  } else if (millis() - lastEnrollmentIdlePollLog >= 30000UL) {
    lastEnrollmentIdlePollLog = millis();
    Serial.print("Enrollment poll: none (HTTP ");
    Serial.print(httpCode);
    Serial.println(", authenticated=yes)");
  }
  if (action == "cleanup") {
    Serial.println("Enrollment: cleanup command received");
    if (info.slot >= SLOT_MIN && !deleteEnrollmentSlot(static_cast<uint16_t>(info.slot))) {
      Serial.println("Enrollment: cleanup delete failed; will retry");
      return;
    }
    if (!acknowledgeCleanup(info.sessionUid,
                            info.slot >= SLOT_MIN ? static_cast<uint16_t>(info.slot) : 0)) {
      Serial.println("Enrollment: cleanup acknowledgement failed; will retry");
    }
    return;
  }
  if (action == "enroll" && info.active) {
    Serial.print("Enrollment: command for session ");
    Serial.println(info.sessionUid);
    startActiveEnrollment(info);
  }
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

// ===========================================================================
// 16. ARDUINO ENTRY POINTS
// ===========================================================================

void setup() {
  Serial.begin(115200);
  delay(200);
  Serial.println();
  Serial.println("Edwins Lab ESP32 door lock starting");
  Serial.print("Firmware built: ");
  Serial.print(__DATE__);
  Serial.print(" ");
  Serial.println(__TIME__);

  pinMode(RELAY_PIN, OUTPUT);
  pinMode(EXIT_BUTTON, INPUT_PULLUP);
  pinMode(GREEN_LED, OUTPUT);
  pinMode(RED_LED, OUTPUT);
  pinMode(BUZZER_PIN, OUTPUT);

  // Fail-safe default: the de-energised relay is the locked state for this
  // wiring, and the door stays locked until the backend authorises a login.
  digitalWrite(RELAY_PIN, RELAY_LOCKED_LEVEL);
  digitalWrite(GREEN_LED, LOW);
  digitalWrite(RED_LED, LOW);
  digitalWrite(BUZZER_PIN, LOW);

  bool readerReady = r307sBegin();
  readerReadiness = readerReady ? ReaderReadiness::INITIALIZING : ReaderReadiness::NOT_READY;
  Serial.print("Fingerprint reader: ");
  Serial.println(readerReady ? "password verified; capture readiness pending" : "not detected");

  loadPersistedEnrollment();

  wifiBegin();
  Serial.println("Wi-Fi connection started");
}

void loop() {
  // Push-button exit is served first and never depends on Wi-Fi or the backend.
  checkExitButton();
  updateDoor();

  wifiMaintain();

  if (!isWifiConnected()) {
    delay(5);
    return;
  }

  if (!isDeviceAuthenticated()) {
    // An active enrollment must not stall for the full idle auth retry window.
    unsigned long authRetry = activeEnrollment.active ? 5000UL : AUTH_RETRY_INTERVAL;
    if (millis() - lastAuthAttempt >= authRetry) {
      lastAuthAttempt = millis();
      Serial.println("Attempting backend device authentication...");
      if (authenticateDevice()) {
        reportReaderIfDue(true);
        successSignal();
      }
    }
    delay(5);
    return;
  }

  if (currentMode == SensorMode::IDLE) {
    // Claim pending enrollment work first, before any login scan consumes a
    // finger the operator is presenting for enrollment.
    processEnrollmentCommand();
    if (currentMode == SensorMode::IDLE) {
      if (millis() - lastHeartbeat >= HEARTBEAT_INTERVAL) {
        lastHeartbeat = millis();
        handleHeartbeat();
      }
      // Reader status is only reported while idle; a blocking report must not
      // stall the enrollment state machine.
      reportReaderIfDue(false);

      // Safe to poll for login; the backend authorises every successful match.
      pollReaderForLogin();
    }
  } else {
    // Enrollment and cleanup own the sensor exclusively.
    processActiveEnrollment();
  }

  delay(5);
}
