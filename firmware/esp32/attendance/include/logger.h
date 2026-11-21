#pragma once

#include <cstdint>

#include "config.h"

namespace attendance {

enum LogLevel : uint8_t {
  kLogError = 0,
  kLogWarn = 1,
  kLogInfo = 2,
  kLogDebug = 3,
};

class Logger {
 public:
  static void begin(unsigned long baud);
  static void logf(uint8_t level, const char* tag, const char* format, ...);
};

}  // namespace attendance

#if ATTENDANCE_LOG_LEVEL >= 0
#define ATT_LOG_E(tag, ...) ::attendance::Logger::logf(::attendance::kLogError, tag, __VA_ARGS__)
#else
#define ATT_LOG_E(tag, ...) \
  do {                     \
  } while (0)
#endif

#if ATTENDANCE_LOG_LEVEL >= 1
#define ATT_LOG_W(tag, ...) ::attendance::Logger::logf(::attendance::kLogWarn, tag, __VA_ARGS__)
#else
#define ATT_LOG_W(tag, ...) \
  do {                     \
  } while (0)
#endif

#if ATTENDANCE_LOG_LEVEL >= 2
#define ATT_LOG_I(tag, ...) ::attendance::Logger::logf(::attendance::kLogInfo, tag, __VA_ARGS__)
#else
#define ATT_LOG_I(tag, ...) \
  do {                     \
  } while (0)
#endif

#if ATTENDANCE_LOG_LEVEL >= 3
#define ATT_LOG_D(tag, ...) ::attendance::Logger::logf(::attendance::kLogDebug, tag, __VA_ARGS__)
#else
#define ATT_LOG_D(tag, ...) \
  do {                     \
  } while (0)
#endif
