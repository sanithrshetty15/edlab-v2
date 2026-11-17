#ifndef ENROLLMENT_FLOW_H
#define ENROLLMENT_FLOW_H

#include <stdint.h>

// Pure, dependency-free enrollment state machine. Keeping the transitions in
// a small header makes it testable on the native PlatformIO environment.
namespace enrollment_flow {

// The exact stage sequence required by the enrollment protocol. Each stage has
// one responsibility so the serial log and backend progress always identify
// where enrollment stopped.
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

inline Stage next(Stage stage, Event event) {
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

inline bool isActive(Stage stage) {
  return stage != Stage::Idle && stage != Stage::Completed && stage != Stage::Failed;
}

inline bool isTerminal(Stage stage) {
  return stage == Stage::Completed || stage == Stage::Failed;
}

inline const char* stageName(Stage stage) {
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

inline const char* progressStatus(Stage stage) {
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

} // namespace enrollment_flow

#endif // ENROLLMENT_FLOW_H
