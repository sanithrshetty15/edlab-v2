#include <unity.h>

#include "../../src/attendance_logic.cpp"

using namespace attendance;

void testStateTransitionsAreExplicit() {
  TEST_ASSERT_TRUE(isTransitionAllowed(TerminalState::Boot, TerminalState::HardwareInit));
  TEST_ASSERT_TRUE(isTransitionAllowed(TerminalState::Idle, TerminalState::FingerDetected));
  TEST_ASSERT_TRUE(isTransitionAllowed(TerminalState::FingerDetected, TerminalState::FingerMatched));
  TEST_ASSERT_TRUE(isTransitionAllowed(TerminalState::EventSubmitting, TerminalState::EventConfirmed));
  TEST_ASSERT_TRUE(isTransitionAllowed(TerminalState::EnrollmentMode, TerminalState::Idle));
  TEST_ASSERT_FALSE(isTransitionAllowed(TerminalState::Boot, TerminalState::Idle));
  TEST_ASSERT_FALSE(isTransitionAllowed(TerminalState::EventConfirmed, TerminalState::FingerDetected));
}

void testHttpClassification() {
  TEST_ASSERT_TRUE(classifyHttpStatus(200) == HttpClass::Success);
  TEST_ASSERT_TRUE(classifyHttpStatus(204) == HttpClass::Success);
  TEST_ASSERT_TRUE(classifyHttpStatus(401) == HttpClass::Reauthenticate);
  TEST_ASSERT_TRUE(classifyHttpStatus(403) == HttpClass::Reauthenticate);
  TEST_ASSERT_TRUE(classifyHttpStatus(429) == HttpClass::Retry);
  TEST_ASSERT_TRUE(classifyHttpStatus(503) == HttpClass::Retry);
  TEST_ASSERT_TRUE(classifyHttpStatus(422) == HttpClass::Reject);
  TEST_ASSERT_TRUE(classifyHttpStatus(409) == HttpClass::Reject);
}

void testExponentialBackoffHasCap() {
  TEST_ASSERT_EQUAL_UINT32(1000, exponentialBackoffMs(0, 1000, 60000));
  TEST_ASSERT_EQUAL_UINT32(2000, exponentialBackoffMs(1, 1000, 60000));
  TEST_ASSERT_EQUAL_UINT32(8000, exponentialBackoffMs(3, 1000, 60000));
  TEST_ASSERT_EQUAL_UINT32(60000, exponentialBackoffMs(20, 1000, 60000));
  TEST_ASSERT_EQUAL_UINT32(60000, exponentialBackoffMs(200, 1000, 60000));
}

void testCooldownHandlesMillisWrap() {
  TEST_ASSERT_TRUE(scanCooldownElapsed(4000, 1000, 3000));
  TEST_ASSERT_FALSE(scanCooldownElapsed(3000, 1000, 3000));
}

void testCooldownWraparound() {
  TEST_ASSERT_TRUE(scanCooldownElapsed(0x00000060UL, 0xFFFFFFF0UL, 100));
  TEST_ASSERT_FALSE(scanCooldownElapsed(0xFFFFFFF5UL, 0xFFFFFFF0UL, 100));
}

void testTimeReachedWraps() {
  TEST_ASSERT_TRUE(timeReached(1000, 1000));
  TEST_ASSERT_TRUE(timeReached(1001, 1000));
  TEST_ASSERT_FALSE(timeReached(999, 1000));
  TEST_ASSERT_TRUE(timeReached(5, 0xFFFFFFF0UL));
  TEST_ASSERT_FALSE(timeReached(0xFFFFFFF0UL, 5));
}

void testSlotValidation() {
  TEST_ASSERT_FALSE(isValidSlot(0));
  TEST_ASSERT_TRUE(isValidSlot(1));
  TEST_ASSERT_TRUE(isValidSlot(127));
  TEST_ASSERT_FALSE(isValidSlot(128));
}

void testIdempotencyKeyValidation() {
  TEST_ASSERT_TRUE(isValidIdempotencyKey("att-0123456789ab-1a2b3c4d-00000001"));
  TEST_ASSERT_FALSE(isValidIdempotencyKey("short"));
  TEST_ASSERT_FALSE(isValidIdempotencyKey("has a space in it"));
  TEST_ASSERT_FALSE(isValidIdempotencyKey(""));
  TEST_ASSERT_FALSE(isValidIdempotencyKey(nullptr));
}

void testPlaceholderDetection() {
  TEST_ASSERT_TRUE(containsPlaceholder("CHANGE_ME_DEVICE_API_KEY"));
  TEST_ASSERT_TRUE(containsPlaceholder("change_me_device_api_key"));
  TEST_ASSERT_TRUE(containsPlaceholder("your-wifi-password"));
  TEST_ASSERT_TRUE(containsPlaceholder("paste-the-provisioned-key"));
  TEST_ASSERT_FALSE(containsPlaceholder("https://edwinslab.onrender.com"));
  TEST_ASSERT_FALSE(containsPlaceholder("Edwins Lab attendance terminal"));
}

void testStateNames() {
  TEST_ASSERT_EQUAL_STRING("IDLE", terminalStateName(TerminalState::Idle));
  TEST_ASSERT_EQUAL_STRING("ENROLLMENT_MODE", terminalStateName(TerminalState::EnrollmentMode));
}

int main(int argc, char** argv) {
  (void)argc;
  (void)argv;
  UNITY_BEGIN();
  RUN_TEST(testStateTransitionsAreExplicit);
  RUN_TEST(testHttpClassification);
  RUN_TEST(testExponentialBackoffHasCap);
  RUN_TEST(testCooldownHandlesMillisWrap);
  RUN_TEST(testCooldownWraparound);
  RUN_TEST(testTimeReachedWraps);
  RUN_TEST(testSlotValidation);
  RUN_TEST(testIdempotencyKeyValidation);
  RUN_TEST(testPlaceholderDetection);
  RUN_TEST(testStateNames);
  return UNITY_END();
}
