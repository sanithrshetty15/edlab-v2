#pragma once

#include <cstdint>

namespace attendance {

enum class WifiState : uint8_t {
  Idle,
  Connecting,
  Connected,
  Backoff,
};

class WifiManager {
 public:
  void begin();
  void update();
  bool connected() const;
  int rssi() const;
  const char* stateName() const;
  uint8_t attempt() const { return attempt_; }

 private:
  void startAttempt();
  void scheduleRetry();

  WifiState state_ = WifiState::Idle;
  uint8_t attempt_ = 0;
  uint32_t attemptStartedMs_ = 0;
  uint32_t retryAtMs_ = 0;
};

}  // namespace attendance
