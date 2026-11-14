#ifndef ENROLLMENT_FLOW_H
#define ENROLLMENT_FLOW_H

#include <stdint.h>

// Pure, dependency-free enrollment state machine. Keeping the transitions in
// a small header makes it testable on the native PlatformIO environment.
namespace enrollment_flow {

enum class Stage : uint8_t {
  Idle,
  WaitingFirstScan,
  FirstScanCaptured,
  WaitingFingerRemoval,
  WaitingSecondScan,
  SecondScanCaptured,
  Finalizing,
  Complete,
  Failed,
};

enum class Event : uint8_t {
  Start,
  FirstCaptured,
  FirstCaptureReported,
  FingerRemoved,
  SecondCaptured,
  TemplateReady,
  Completed,
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
      return event == Event::Start ? Stage::WaitingFirstScan : Stage::Idle;
    case Stage::WaitingFirstScan:
      return event == Event::FirstCaptured ? Stage::FirstScanCaptured : stage;
    case Stage::FirstScanCaptured:
      return event == Event::FirstCaptureReported ? Stage::WaitingFingerRemoval : stage;
    case Stage::WaitingFingerRemoval:
      return event == Event::FingerRemoved ? Stage::WaitingSecondScan : stage;
    case Stage::WaitingSecondScan:
      return event == Event::SecondCaptured ? Stage::SecondScanCaptured : stage;
    case Stage::SecondScanCaptured:
      return event == Event::TemplateReady ? Stage::Finalizing : stage;
    case Stage::Finalizing:
      return event == Event::Completed ? Stage::Complete : stage;
    case Stage::Complete:
    case Stage::Failed:
      return stage;
  }
  return stage;
}

inline bool isActive(Stage stage) {
  return stage != Stage::Idle && stage != Stage::Complete && stage != Stage::Failed;
}

inline bool isTerminal(Stage stage) {
  return stage == Stage::Complete || stage == Stage::Failed;
}

inline const char* progressStatus(Stage stage) {
  switch (stage) {
    case Stage::WaitingFirstScan:
      return "waiting_first_scan";
    case Stage::FirstScanCaptured:
      return "first_scan_captured";
    case Stage::WaitingFingerRemoval:
      return "waiting_for_removal";
    case Stage::WaitingSecondScan:
      return "waiting_second_scan";
    case Stage::SecondScanCaptured:
      return "second_scan_captured";
    case Stage::Finalizing:
      return "verifying";
    case Stage::Complete:
      return "finalized";
    case Stage::Failed:
      return "failed";
    case Stage::Idle:
      return "";
  }
  return "";
}

} // namespace enrollment_flow

#endif // ENROLLMENT_FLOW_H
