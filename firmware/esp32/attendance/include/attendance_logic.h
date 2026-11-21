#pragma once

#include <cstddef>
#include <cstdint>
#include <cstring>

// Pure, Arduino-independent logic shared by the firmware and the native tests.
namespace attendance {

enum class TerminalState : uint8_t {
  Boot = 0,
  HardwareInit,
  WifiConnecting,
  BackendConnecting,
  Syncing,
  Idle,
  FingerDetected,
  FingerMatched,
  EventPending,
  EventSubmitting,
  EventConfirmed,
  EnrollmentMode,
  ErrorRecovery,
};

const char* terminalStateName(TerminalState state);

bool isTransitionAllowed(TerminalState from, TerminalState to);

enum class HttpClass : uint8_t {
  Success,
  Retry,
  Reauthenticate,
  Reject,
};

HttpClass classifyHttpStatus(int status);

// Capped exponential backoff with no jitter, shared by Wi-Fi, auth and queue
// retry loops. Attempt 0 returns the base delay.
uint32_t exponentialBackoffMs(uint8_t attempt, uint32_t baseMs, uint32_t maxMs);

// Returns true once the cooldown after the previous scan has elapsed. Handles
// millis() wraparound with unsigned arithmetic.
bool scanCooldownElapsed(uint32_t nowMs, uint32_t lastScanMs, uint32_t cooldownMs);

// Wrap-safe deadline check for millis()-based deadlines.
bool timeReached(uint32_t nowMs, uint32_t targetMs);

bool isValidSlot(int slot);
bool isValidIdempotencyKey(const char* value);

// True for the placeholder values shipped in the example configuration.
bool containsPlaceholder(const char* value);

}  // namespace attendance
