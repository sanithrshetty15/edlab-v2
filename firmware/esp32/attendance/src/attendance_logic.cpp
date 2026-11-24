#include "attendance_logic.h"

#include <cctype>

namespace attendance {
namespace {

struct Transition {
  TerminalState from;
  TerminalState to;
};

constexpr Transition kTransitions[] = {
    {TerminalState::Boot, TerminalState::HardwareInit},
    {TerminalState::HardwareInit, TerminalState::WifiConnecting},
    {TerminalState::HardwareInit, TerminalState::ErrorRecovery},
    {TerminalState::WifiConnecting, TerminalState::BackendConnecting},
    {TerminalState::WifiConnecting, TerminalState::Idle},
    {TerminalState::WifiConnecting, TerminalState::ErrorRecovery},
    {TerminalState::BackendConnecting, TerminalState::Syncing},
    {TerminalState::BackendConnecting, TerminalState::Idle},
    {TerminalState::BackendConnecting, TerminalState::ErrorRecovery},
    {TerminalState::Syncing, TerminalState::Idle},
    {TerminalState::Syncing, TerminalState::ErrorRecovery},
    {TerminalState::Idle, TerminalState::FingerDetected},
    {TerminalState::Idle, TerminalState::EnrollmentMode},
    {TerminalState::Idle, TerminalState::ErrorRecovery},
    {TerminalState::FingerDetected, TerminalState::FingerMatched},
    {TerminalState::FingerDetected, TerminalState::Idle},
    {TerminalState::FingerMatched, TerminalState::EventPending},
    {TerminalState::FingerMatched, TerminalState::Idle},
    {TerminalState::EventPending, TerminalState::EventSubmitting},
    {TerminalState::EventPending, TerminalState::Idle},
    {TerminalState::EventSubmitting, TerminalState::EventConfirmed},
    {TerminalState::EventSubmitting, TerminalState::EventPending},
    {TerminalState::EventSubmitting, TerminalState::Idle},
    {TerminalState::EventConfirmed, TerminalState::Idle},
    {TerminalState::EnrollmentMode, TerminalState::Idle},
    {TerminalState::EnrollmentMode, TerminalState::ErrorRecovery},
    {TerminalState::ErrorRecovery, TerminalState::HardwareInit},
    {TerminalState::ErrorRecovery, TerminalState::WifiConnecting},
    {TerminalState::ErrorRecovery, TerminalState::Idle},
};

bool containsIgnoreCase(const char* haystack, const char* needle) {
  if (*needle == '\0') {
    return true;
  }
  for (const char* start = haystack; *start != '\0'; ++start) {
    const char* a = start;
    const char* b = needle;
    while (*a != '\0' && *b != '\0' &&
           std::tolower(static_cast<unsigned char>(*a)) ==
               std::tolower(static_cast<unsigned char>(*b))) {
      ++a;
      ++b;
    }
    if (*b == '\0') {
      return true;
    }
  }
  return false;
}

}  // namespace

const char* terminalStateName(TerminalState state) {
  switch (state) {
    case TerminalState::Boot:
      return "BOOT";
    case TerminalState::HardwareInit:
      return "HARDWARE_INIT";
    case TerminalState::WifiConnecting:
      return "WIFI_CONNECTING";
    case TerminalState::BackendConnecting:
      return "BACKEND_CONNECTING";
    case TerminalState::Syncing:
      return "SYNCING";
    case TerminalState::Idle:
      return "IDLE";
    case TerminalState::FingerDetected:
      return "FINGERPRINT_DETECTED";
    case TerminalState::FingerMatched:
      return "FINGERPRINT_MATCHED";
    case TerminalState::EventPending:
      return "EVENT_PENDING";
    case TerminalState::EventSubmitting:
      return "EVENT_SUBMITTING";
    case TerminalState::EventConfirmed:
      return "EVENT_CONFIRMED";
    case TerminalState::EnrollmentMode:
      return "ENROLLMENT_MODE";
    case TerminalState::ErrorRecovery:
      return "ERROR_RECOVERY";
  }
  return "UNKNOWN";
}

bool isTransitionAllowed(TerminalState from, TerminalState to) {
  if (from == to) {
    return true;
  }
  for (const Transition& transition : kTransitions) {
    if (transition.from == from && transition.to == to) {
      return true;
    }
  }
  return false;
}

HttpClass classifyHttpStatus(int status) {
  if (status >= 200 && status < 300) {
    return HttpClass::Success;
  }
  if (status == 401 || status == 403) {
    return HttpClass::Reauthenticate;
  }
  if (status == 408 || status == 425 || status == 429 || status >= 500) {
    return HttpClass::Retry;
  }
  if (status >= 400 && status < 500) {
    return HttpClass::Reject;
  }
  return HttpClass::Retry;
}

uint32_t exponentialBackoffMs(uint8_t attempt, uint32_t baseMs, uint32_t maxMs) {
  if (baseMs == 0) {
    return 0;
  }
  if (attempt > 20) {
    return maxMs;
  }
  uint64_t delay = static_cast<uint64_t>(baseMs) << attempt;
  if (delay > maxMs) {
    delay = maxMs;
  }
  return static_cast<uint32_t>(delay);
}

bool scanCooldownElapsed(uint32_t nowMs, uint32_t lastScanMs, uint32_t cooldownMs) {
  return static_cast<uint32_t>(nowMs - lastScanMs) >= cooldownMs;
}

bool timeReached(uint32_t nowMs, uint32_t targetMs) {
  return static_cast<int32_t>(nowMs - targetMs) >= 0;
}

bool isValidSlot(int slot) {
  return slot >= kFingerprintSlotMin && slot <= kFingerprintSlotMax;
}

bool isValidIdempotencyKey(const char* value) {
  if (value == nullptr) {
    return false;
  }
  const size_t length = std::strlen(value);
  if (length < 8 || length > 128) {
    return false;
  }
  for (size_t i = 0; i < length; ++i) {
    const unsigned char c = static_cast<unsigned char>(value[i]);
    const bool allowed = std::isalnum(c) || c == '.' || c == '_' || c == ':' || c == '-';
    if (!allowed) {
      return false;
    }
  }
  return true;
}

bool containsPlaceholder(const char* value) {
  if (value == nullptr) {
    return true;
  }
  return containsIgnoreCase(value, "change_me") || containsIgnoreCase(value, "your-") ||
         containsIgnoreCase(value, "paste-") || containsIgnoreCase(value, "example.com");
}

}  // namespace attendance
