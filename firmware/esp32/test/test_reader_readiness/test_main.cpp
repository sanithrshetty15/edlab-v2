#include <unity.h>

#include "reader_readiness.h"

void test_status_strings() {
  TEST_ASSERT_EQUAL_STRING("not_ready",
                           readerReadinessStatus(ReaderReadiness::NOT_READY));
  TEST_ASSERT_EQUAL_STRING("initializing",
                           readerReadinessStatus(ReaderReadiness::INITIALIZING));
  TEST_ASSERT_EQUAL_STRING("ready", readerReadinessStatus(ReaderReadiness::READY));
  TEST_ASSERT_EQUAL_STRING("waiting_for_finger",
                           readerReadinessStatus(ReaderReadiness::WAITING_FOR_FINGER));
  TEST_ASSERT_EQUAL_STRING("finger_detected",
                           readerReadinessStatus(ReaderReadiness::FINGER_DETECTED));
  TEST_ASSERT_EQUAL_STRING("capture_failed",
                           readerReadinessStatus(ReaderReadiness::CAPTURE_FAILED));
}

void test_connected_states() {
  TEST_ASSERT_FALSE(readerReadinessIsConnected(ReaderReadiness::NOT_READY));
  TEST_ASSERT_TRUE(readerReadinessIsConnected(ReaderReadiness::INITIALIZING));
  TEST_ASSERT_TRUE(readerReadinessIsConnected(ReaderReadiness::READY));
  TEST_ASSERT_TRUE(readerReadinessIsConnected(ReaderReadiness::WAITING_FOR_FINGER));
  TEST_ASSERT_TRUE(readerReadinessIsConnected(ReaderReadiness::FINGER_DETECTED));
  TEST_ASSERT_FALSE(readerReadinessIsConnected(ReaderReadiness::CAPTURE_FAILED));
}

void test_only_verified_states_can_start_enrollment() {
  TEST_ASSERT_FALSE(readerReadinessCanStartEnrollment(ReaderReadiness::NOT_READY));
  TEST_ASSERT_FALSE(readerReadinessCanStartEnrollment(ReaderReadiness::INITIALIZING));
  TEST_ASSERT_TRUE(readerReadinessCanStartEnrollment(ReaderReadiness::READY));
  TEST_ASSERT_TRUE(
      readerReadinessCanStartEnrollment(ReaderReadiness::WAITING_FOR_FINGER));
  TEST_ASSERT_FALSE(readerReadinessCanStartEnrollment(ReaderReadiness::FINGER_DETECTED));
  TEST_ASSERT_FALSE(readerReadinessCanStartEnrollment(ReaderReadiness::CAPTURE_FAILED));
}

void test_finger_detection_is_explicit() {
  TEST_ASSERT_TRUE(readerReadinessHasFinger(ReaderReadiness::FINGER_DETECTED));
  TEST_ASSERT_FALSE(readerReadinessHasFinger(ReaderReadiness::WAITING_FOR_FINGER));
  TEST_ASSERT_FALSE(readerReadinessHasFinger(ReaderReadiness::READY));
}

int main(int, char**) {
  UNITY_BEGIN();
  RUN_TEST(test_status_strings);
  RUN_TEST(test_connected_states);
  RUN_TEST(test_only_verified_states_can_start_enrollment);
  RUN_TEST(test_finger_detection_is_explicit);
  return UNITY_END();
}
