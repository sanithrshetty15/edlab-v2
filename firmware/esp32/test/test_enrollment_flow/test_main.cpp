#include <unity.h>

#include "enrollment_flow.h"

using enrollment_flow::Event;
using enrollment_flow::Stage;

void test_start_enters_waiting_first_scan() {
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::WaitingFirstScan),
                    static_cast<int>(enrollment_flow::next(Stage::Idle, Event::Start)));
}

void test_second_scan_cannot_be_accepted_before_removal() {
  Stage stage = enrollment_flow::next(Stage::Idle, Event::Start);
  stage = enrollment_flow::next(stage, Event::SecondCaptured);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::WaitingFirstScan), static_cast<int>(stage));
}

void test_full_success_path() {
  Stage stage = Stage::Idle;
  stage = enrollment_flow::next(stage, Event::Start);
  stage = enrollment_flow::next(stage, Event::FirstCaptured);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::FirstScanCaptured), static_cast<int>(stage));

  stage = enrollment_flow::next(stage, Event::FirstCaptureReported);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::WaitingFingerRemoval), static_cast<int>(stage));

  stage = enrollment_flow::next(stage, Event::FingerRemoved);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::WaitingSecondScan), static_cast<int>(stage));

  stage = enrollment_flow::next(stage, Event::SecondCaptured);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::SecondScanCaptured), static_cast<int>(stage));

  stage = enrollment_flow::next(stage, Event::TemplateReady);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::Finalizing), static_cast<int>(stage));

  stage = enrollment_flow::next(stage, Event::Completed);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::Complete), static_cast<int>(stage));
  TEST_ASSERT_TRUE(enrollment_flow::isTerminal(stage));
}

void test_failure_blocks_completion_until_reset() {
  Stage stage = enrollment_flow::next(Stage::Idle, Event::Start);
  stage = enrollment_flow::next(stage, Event::Failed);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::Failed), static_cast<int>(stage));

  stage = enrollment_flow::next(stage, Event::Completed);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::Failed), static_cast<int>(stage));

  stage = enrollment_flow::next(stage, Event::Reset);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::Idle), static_cast<int>(stage));
}

void test_progress_statuses() {
  TEST_ASSERT_EQUAL_STRING("waiting_first_scan",
                           enrollment_flow::progressStatus(Stage::WaitingFirstScan));
  TEST_ASSERT_EQUAL_STRING("first_scan_captured",
                           enrollment_flow::progressStatus(Stage::FirstScanCaptured));
  TEST_ASSERT_EQUAL_STRING("waiting_for_removal",
                           enrollment_flow::progressStatus(Stage::WaitingFingerRemoval));
  TEST_ASSERT_EQUAL_STRING("waiting_second_scan",
                           enrollment_flow::progressStatus(Stage::WaitingSecondScan));
  TEST_ASSERT_EQUAL_STRING("second_scan_captured",
                           enrollment_flow::progressStatus(Stage::SecondScanCaptured));
  TEST_ASSERT_EQUAL_STRING("verifying",
                           enrollment_flow::progressStatus(Stage::Finalizing));
}

int main(int, char**) {
  UNITY_BEGIN();
  RUN_TEST(test_start_enters_waiting_first_scan);
  RUN_TEST(test_second_scan_cannot_be_accepted_before_removal);
  RUN_TEST(test_full_success_path);
  RUN_TEST(test_failure_blocks_completion_until_reset);
  RUN_TEST(test_progress_statuses);
  return UNITY_END();
}
