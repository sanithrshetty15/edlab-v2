#pragma once

#include <Arduino.h>
#include <atomic>
#include <cstdint>

namespace attendance {

class EventIdGenerator {
 public:
  void begin();
  String next();

 private:
  char bootId_[9] = {0};
  std::atomic<uint32_t> counter_{0};
};

}  // namespace attendance
