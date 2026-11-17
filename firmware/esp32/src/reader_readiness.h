#ifndef READER_READINESS_H
#define READER_READINESS_H

#include <stdint.h>

// A reader is only ready when the sensor itself confirms that it can capture
// images. A successful password handshake alone is not enough.
enum class ReaderReadiness : uint8_t {
  NOT_READY,
  INITIALIZING,
  READY,
  WAITING_FOR_FINGER,
  FINGER_DETECTED,
  CAPTURE_FAILED,
  ENROLLING,
};

inline const char* readerReadinessStatus(ReaderReadiness readiness) {
  switch (readiness) {
    case ReaderReadiness::NOT_READY:
      return "not_ready";
    case ReaderReadiness::INITIALIZING:
      return "initializing";
    case ReaderReadiness::READY:
      return "ready";
    case ReaderReadiness::WAITING_FOR_FINGER:
      return "waiting_for_finger";
    case ReaderReadiness::FINGER_DETECTED:
      return "finger_detected";
    case ReaderReadiness::CAPTURE_FAILED:
      return "capture_failed";
    case ReaderReadiness::ENROLLING:
      return "enrolling";
  }
  return "not_ready";
}

inline bool readerReadinessIsConnected(ReaderReadiness readiness) {
  switch (readiness) {
    case ReaderReadiness::INITIALIZING:
    case ReaderReadiness::READY:
    case ReaderReadiness::WAITING_FOR_FINGER:
    case ReaderReadiness::FINGER_DETECTED:
    case ReaderReadiness::ENROLLING:
      return true;
    case ReaderReadiness::NOT_READY:
    case ReaderReadiness::CAPTURE_FAILED:
      return false;
  }
  return false;
}

inline bool readerReadinessCanStartEnrollment(ReaderReadiness readiness) {
  return readiness == ReaderReadiness::READY ||
         readiness == ReaderReadiness::WAITING_FOR_FINGER;
}

inline bool readerReadinessHasFinger(ReaderReadiness readiness) {
  return readiness == ReaderReadiness::FINGER_DETECTED;
}

#endif // READER_READINESS_H
