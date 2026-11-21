#include "feedback.h"

#include <Arduino.h>

#include "hardware_config.h"
#include "logger.h"

namespace attendance {
namespace {
constexpr uint8_t kLedGreen = 0x01;
constexpr uint8_t kLedRed = 0x02;
constexpr uint8_t kLedBlue = 0x04;

constexpr FeedbackController::Step kBoot[] = {{true, kLedGreen, 120}};
constexpr FeedbackController::Step kInitializing[] = {
    {false, kLedBlue, 100}, {false, 0, 100}, {false, kLedBlue, 100}};
constexpr FeedbackController::Step kReady[] = {{true, kLedGreen, 70}};
constexpr FeedbackController::Step kDetected[] = {{true, kLedBlue, 40}};
constexpr FeedbackController::Step kMatched[] = {{true, kLedGreen, 60}};
constexpr FeedbackController::Step kSending[] = {{false, kLedBlue, 120}};
constexpr FeedbackController::Step kEntry[] = {
    {true, kLedGreen, 80}, {false, 0, 70}, {true, kLedGreen, 80}};
constexpr FeedbackController::Step kExit[] = {{true, kLedGreen, 180}};
constexpr FeedbackController::Step kDuplicate[] = {
    {true, kLedGreen | kLedRed, 60}, {false, 0, 60}, {true, kLedGreen | kLedRed, 60}};
constexpr FeedbackController::Step kUnknown[] = {{true, kLedRed, 350}};
constexpr FeedbackController::Step kOffline[] = {
    {true, kLedRed, 60}, {false, 0, 70}, {true, kLedRed, 60},
    {false, 0, 70},      {true, kLedRed, 60}};
constexpr FeedbackController::Step kEnrollment[] = {
    {true, kLedBlue, 90}, {false, 0, 90}, {true, kLedBlue, 90}};
constexpr FeedbackController::Step kError[] = {
    {true, kLedRed, 70}, {false, 0, 80}, {true, kLedRed, 70},
    {false, 0, 80},      {true, kLedRed, 70}};
constexpr FeedbackController::Step kWifiLost[] = {
    {true, kLedRed, 120}, {false, 0, 100}, {true, kLedRed, 120}};

struct Pattern {
  const FeedbackController::Step* steps;
  uint8_t length;
};

Pattern patternFor(FeedbackCue cue) {
  switch (cue) {
    case FeedbackCue::Boot:
      return {kBoot, sizeof(kBoot) / sizeof(kBoot[0])};
    case FeedbackCue::Initializing:
      return {kInitializing, sizeof(kInitializing) / sizeof(kInitializing[0])};
    case FeedbackCue::Ready:
      return {kReady, sizeof(kReady) / sizeof(kReady[0])};
    case FeedbackCue::FingerDetected:
      return {kDetected, sizeof(kDetected) / sizeof(kDetected[0])};
    case FeedbackCue::FingerMatched:
      return {kMatched, sizeof(kMatched) / sizeof(kMatched[0])};
    case FeedbackCue::Sending:
      return {kSending, sizeof(kSending) / sizeof(kSending[0])};
    case FeedbackCue::EntryConfirmed:
      return {kEntry, sizeof(kEntry) / sizeof(kEntry[0])};
    case FeedbackCue::ExitConfirmed:
      return {kExit, sizeof(kExit) / sizeof(kExit[0])};
    case FeedbackCue::Duplicate:
      return {kDuplicate, sizeof(kDuplicate) / sizeof(kDuplicate[0])};
    case FeedbackCue::UnknownFingerprint:
      return {kUnknown, sizeof(kUnknown) / sizeof(kUnknown[0])};
    case FeedbackCue::OfflineQueued:
      return {kOffline, sizeof(kOffline) / sizeof(kOffline[0])};
    case FeedbackCue::Enrollment:
      return {kEnrollment, sizeof(kEnrollment) / sizeof(kEnrollment[0])};
    case FeedbackCue::Error:
      return {kError, sizeof(kError) / sizeof(kError[0])};
    case FeedbackCue::WifiLost:
      return {kWifiLost, sizeof(kWifiLost) / sizeof(kWifiLost[0])};
  }
  return {nullptr, 0};
}
}  // namespace

void FeedbackController::begin() {
  if (hardware::kBuzzerPin >= 0) {
    pinMode(hardware::kBuzzerPin, OUTPUT);
    digitalWrite(hardware::kBuzzerPin, hardware::kBuzzerActiveHigh ? LOW : HIGH);
  }
  if (hardware::kStatusLedsEnabled) {
    pinMode(hardware::kGreenLedPin, OUTPUT);
    pinMode(hardware::kRedLedPin, OUTPUT);
  }
  allOff();
}

void FeedbackController::setLeds(uint8_t mask) {
  if (!hardware::kStatusLedsEnabled) {
    return;
  }
  const bool green = (mask & kLedGreen) != 0;
  const bool red = (mask & kLedRed) != 0;
  const bool blue = (mask & kLedBlue) != 0;
  const int onLevel = hardware::kLedActiveHigh ? HIGH : LOW;
  const int offLevel = hardware::kLedActiveHigh ? LOW : HIGH;
  digitalWrite(hardware::kGreenLedPin, green ? onLevel : offLevel);
  digitalWrite(hardware::kRedLedPin, red ? onLevel : offLevel);
  if (blue) {
    ATT_LOG_D("feedback", "blue requested but the attendance PCB has no verified blue LED");
  }
}

void FeedbackController::applyStep(const Step& step) {
  if (hardware::kBuzzerPin >= 0) {
    const bool high = step.buzzerOn == hardware::kBuzzerActiveHigh;
    digitalWrite(hardware::kBuzzerPin, high ? HIGH : LOW);
  }
  setLeds(step.ledMask);
}

void FeedbackController::allOff() {
  if (hardware::kBuzzerPin >= 0) {
    digitalWrite(hardware::kBuzzerPin, hardware::kBuzzerActiveHigh ? LOW : HIGH);
  }
  if (hardware::kStatusLedsEnabled) {
    const int offLevel = hardware::kLedActiveHigh ? LOW : HIGH;
    digitalWrite(hardware::kGreenLedPin, offLevel);
    digitalWrite(hardware::kRedLedPin, offLevel);
  }
}

void FeedbackController::play(FeedbackCue cue) {
  const Pattern pattern = patternFor(cue);
  if (pattern.steps == nullptr || pattern.length == 0) {
    return;
  }
  pattern_ = pattern.steps;
  patternLength_ = pattern.length;
  stepIndex_ = 0;
  stepStartedMs_ = millis();
  active_ = true;
  applyStep(pattern_[0]);
}

void FeedbackController::setEnrollmentIndicator(bool enabled) {
  enrollmentIndicator_ = enabled;
  if (!enabled) {
    indicatorOn_ = false;
  } else {
    indicatorChangedMs_ = millis();
    indicatorOn_ = true;
  }
}

void FeedbackController::update() {
  const uint32_t now = millis();
  if (active_) {
    const Step& current = pattern_[stepIndex_];
    if (now - stepStartedMs_ >= current.durationMs) {
      ++stepIndex_;
      stepStartedMs_ = now;
      if (stepIndex_ >= patternLength_) {
        active_ = false;
        allOff();
      } else {
        applyStep(pattern_[stepIndex_]);
      }
    }
    return;
  }

  if (!enrollmentIndicator_ || !hardware::kStatusLedsEnabled) {
    return;
  }
  if (now - indicatorChangedMs_ >= (indicatorOn_ ? 120 : 880)) {
    indicatorOn_ = !indicatorOn_;
    indicatorChangedMs_ = now;
    setLeds(indicatorOn_ ? kLedBlue : 0);
  }
}

}  // namespace attendance
