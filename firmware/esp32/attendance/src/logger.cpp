#include "logger.h"

#include <Arduino.h>
#include <freertos/FreeRTOS.h>
#include <freertos/semphr.h>
#include <stdarg.h>
#include <stdio.h>
#include <cstring>

namespace attendance {
namespace {
SemaphoreHandle_t gLogMutex = nullptr;
const char* levelName(uint8_t level) {
  switch (level) {
    case kLogError:
      return "E";
    case kLogWarn:
      return "W";
    case kLogInfo:
      return "I";
    default:
      return "D";
  }
}
}  // namespace

void Logger::begin(unsigned long baud) {
  Serial.begin(baud);
  if (gLogMutex == nullptr) {
    gLogMutex = xSemaphoreCreateMutex();
  }
}

void Logger::logf(uint8_t level, const char* tag, const char* format, ...) {
  if (level > ATTENDANCE_LOG_LEVEL) {
    return;
  }
  char message[192];
  va_list args;
  va_start(args, format);
  vsnprintf(message, sizeof(message), format, args);
  va_end(args);

  char line[240];
  snprintf(line, sizeof(line), "[%lu] %s %s: %s\n", static_cast<unsigned long>(millis()),
           levelName(level), tag, message);

  bool locked = false;
  if (gLogMutex != nullptr) {
    locked = xSemaphoreTake(gLogMutex, pdMS_TO_TICKS(20)) == pdTRUE;
  }
  Serial.write(reinterpret_cast<const uint8_t*>(line), strlen(line));
  if (locked) {
    xSemaphoreGive(gLogMutex);
  }
}

}  // namespace attendance
