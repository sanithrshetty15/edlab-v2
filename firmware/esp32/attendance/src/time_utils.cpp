#include "time_utils.h"

#include <Arduino.h>
#include <time.h>

namespace attendance {
namespace timeutil {
namespace {
bool gStarted = false;
constexpr time_t kMinimumValidEpoch = 1700000000;
}  // namespace

void ensureStarted(bool wifiConnected) {
  if (gStarted || !wifiConnected) {
    return;
  }
  configTime(0, 0, "pool.ntp.org", "time.google.com");
  gStarted = true;
}

bool clockValid() { return time(nullptr) >= kMinimumValidEpoch; }

int64_t nowEpoch() {
  const time_t now = time(nullptr);
  if (now < kMinimumValidEpoch) {
    return 0;
  }
  return static_cast<int64_t>(now);
}

bool formatIso8601(int64_t epochSeconds, char* out, size_t outLength) {
  if (out == nullptr || outLength < 21 || epochSeconds <= 0) {
    return false;
  }
  const time_t value = static_cast<time_t>(epochSeconds);
  struct tm utc;
  if (gmtime_r(&value, &utc) == nullptr) {
    return false;
  }
  return strftime(out, outLength, "%Y-%m-%dT%H:%M:%SZ", &utc) > 0;
}

}  // namespace timeutil
}  // namespace attendance
