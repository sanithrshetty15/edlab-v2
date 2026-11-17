#include <unity.h>

#include "enrollment_flow.h"

using enrollment_flow::Event;
using enrollment_flow::Stage;

void test_start_enters_claim_and_sensor_initialization() {
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::ClaimingSession),
                    static_cast<int>(enrollment_flow::next(Stage::Idle, Event::Start)));
  Stage stage = enrollment_flow::next(Stage::ClaimingSession, Event::CommandClaimed);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::SensorInitialization), static_cast<int>(stage));
  stage = enrollment_flow::next(stage, Event::SensorReady);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::WaitingForFirstFinger), static_cast<int>(stage));
}

void test_second_scan_cannot_be_accepted_before_removal() {
  Stage stage = enrollment_flow::next(Stage::Idle, Event::Start);
  stage = enrollment_flow::next(stage, Event::CommandClaimed);
  stage = enrollment_flow::next(stage, Event::SensorReady);
  stage = enrollment_flow::next(stage, Event::SecondFingerDetected);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::WaitingForFirstFinger), static_cast<int>(stage));
}

void test_full_success_path() {
  Stage stage = Stage::Idle;
  stage = enrollment_flow::next(stage, Event::Start);
  stage = enrollment_flow::next(stage, Event::CommandClaimed);
  stage = enrollment_flow::next(stage, Event::SensorReady);
  stage = enrollment_flow::next(stage, Event::FirstFingerDetected);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::CapturingFirstImage), static_cast<int>(stage));

  stage = enrollment_flow::next(stage, Event::FirstImageCaptured);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::ConvertingFirstImage), static_cast<int>(stage));
  stage = enrollment_flow::next(stage, Event::FirstImageConverted);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::WaitingForFingerRemoval), static_cast<int>(stage));

  stage = enrollment_flow::next(stage, Event::FingerRemoved);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::WaitingForSecondFinger), static_cast<int>(stage));

  stage = enrollment_flow::next(stage, Event::SecondFingerDetected);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::CapturingSecondImage), static_cast<int>(stage));
  stage = enrollment_flow::next(stage, Event::SecondImageCaptured);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::ConvertingSecondImage), static_cast<int>(stage));
  stage = enrollment_flow::next(stage, Event::SecondImageConverted);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::CreatingModel), static_cast<int>(stage));

  stage = enrollment_flow::next(stage, Event::ModelCreated);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::StoringTemplate), static_cast<int>(stage));
  stage = enrollment_flow::next(stage, Event::TemplateStored);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::BackendFinalization), static_cast<int>(stage));
  stage = enrollment_flow::next(stage, Event::BackendFinalized);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::Completed), static_cast<int>(stage));
  TEST_ASSERT_TRUE(enrollment_flow::isTerminal(stage));
}

void test_cleanup_is_part_of_finalization() {
  Stage stage = Stage::BackendFinalization;
  stage = enrollment_flow::next(stage, Event::CleanupRequired);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::EnrollmentCleanup), static_cast<int>(stage));
  stage = enrollment_flow::next(stage, Event::CleanupCompleted);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::Failed), static_cast<int>(stage));
}

void test_failure_blocks_completion_until_reset() {
  Stage stage = enrollment_flow::next(Stage::Idle, Event::Start);
  stage = enrollment_flow::next(stage, Event::Failed);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::Failed), static_cast<int>(stage));

  stage = enrollment_flow::next(stage, Event::BackendFinalized);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::Failed), static_cast<int>(stage));

  stage = enrollment_flow::next(stage, Event::Reset);
  TEST_ASSERT_EQUAL(static_cast<int>(Stage::Idle), static_cast<int>(stage));
}

void test_every_stage_has_a_name() {
  Stage stages[] = {
      Stage::Idle,
      Stage::ClaimingSession,
      Stage::SensorInitialization,
      Stage::WaitingForFirstFinger,
      Stage::CapturingFirstImage,
      Stage::ConvertingFirstImage,
      Stage::WaitingForFingerRemoval,
      Stage::WaitingForSecondFinger,
      Stage::CapturingSecondImage,
      Stage::ConvertingSecondImage,
      Stage::CreatingModel,
      Stage::StoringTemplate,
      Stage::BackendFinalization,
      Stage::EnrollmentCleanup,
      Stage::Completed,
      Stage::Failed,
  };
  TEST_ASSERT_EQUAL(16, static_cast<int>(sizeof(stages) / sizeof(stages[0])));
  for (size_t i = 0; i < sizeof(stages) / sizeof(stages[0]); i++) {
    TEST_ASSERT_TRUE(enrollment_flow::stageName(stages[i]) != nullptr);
  }
}

void test_progress_statuses() {
  TEST_ASSERT_EQUAL_STRING("sensor_initializing",
                           enrollment_flow::progressStatus(Stage::SensorInitialization));
  TEST_ASSERT_EQUAL_STRING("waiting_first_scan",
                           enrollment_flow::progressStatus(Stage::WaitingForFirstFinger));
  TEST_ASSERT_EQUAL_STRING("first_scan_captured",
                           enrollment_flow::progressStatus(Stage::ConvertingFirstImage));
  TEST_ASSERT_EQUAL_STRING("waiting_for_removal",
                           enrollment_flow::progressStatus(Stage::WaitingForFingerRemoval));
  TEST_ASSERT_EQUAL_STRING("waiting_second_scan",
                           enrollment_flow::progressStatus(Stage::WaitingForSecondFinger));
  TEST_ASSERT_EQUAL_STRING("second_scan_captured",
                           enrollment_flow::progressStatus(Stage::ConvertingSecondImage));
  TEST_ASSERT_EQUAL_STRING("verifying",
                           enrollment_flow::progressStatus(Stage::BackendFinalization));
}

int main(int, char**) {
  UNITY_BEGIN();
  RUN_TEST(test_start_enters_claim_and_sensor_initialization);
  RUN_TEST(test_second_scan_cannot_be_accepted_before_removal);
  RUN_TEST(test_full_success_path);
  RUN_TEST(test_cleanup_is_part_of_finalization);
  RUN_TEST(test_failure_blocks_completion_until_reset);
  RUN_TEST(test_every_stage_has_a_name);
  RUN_TEST(test_progress_statuses);
  return UNITY_END();
}
