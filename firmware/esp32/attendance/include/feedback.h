#pragma once

#include <cstdint>

namespace attendance {

enum class FeedbackCue : uint8_t {
  Boot,
  Initializing,
  Ready,
  FingerDetected,
  FingerMatched,
  Sending,
  EntryConfirmed,
  ExitConfirmed,
  Duplicate,
  UnknownFingerprint,
  OfflineQueued,
  Enrollment,
  Error,
  WifiLost,
};

class FeedbackController {
 public:
  struct Step {
    bool buzzerOn;
    uint8_t ledMask;
    uint16_t durationMs;
  };

  void begin();
  void play(FeedbackCue cue);
  void update();
  void setEnrollmentIndicator(bool enabled);

 private:
  void applyStep(const Step& step);
  void allOff();
  void setLeds(uint8_t mask);

  bool active_ = false;
  const Step* pattern_ = nullptr;
  uint8_t patternLength_ = 0;
  uint8_t stepIndex_ = 0;
  uint32_t stepStartedMs_ = 0;
  bool enrollmentIndicator_ = false;
  bool indicatorOn_ = false;
  uint32_t indicatorChangedMs_ = 0;
};

}  // namespace attendance
