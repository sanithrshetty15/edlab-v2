#include "event_id.h"

#include <Esp.h>
#include <esp_system.h>
#include <cstdio>

#include "logger.h"

namespace attendance {

void EventIdGenerator::begin() {
  const uint64_t mac = ESP.getEfuseMac();
  snprintf(bootId_, sizeof(bootId_), "%08lx", static_cast<unsigned long>(esp_random()));
  ATT_LOG_I("event", "event id namespace ready for MAC %04lx%08lx",
            static_cast<unsigned long>((mac >> 32) & 0xFFFF),
            static_cast<unsigned long>(mac & 0xFFFFFFFF));
}

String EventIdGenerator::next() {
  const uint32_t count = ++counter_;
  const uint64_t mac = ESP.getEfuseMac();
  char key[64];
  snprintf(key, sizeof(key), "att-%04lx%08lx-%s-%08lx",
           static_cast<unsigned long>((mac >> 32) & 0xFFFF),
           static_cast<unsigned long>(mac & 0xFFFFFFFF), bootId_,
           static_cast<unsigned long>(count));
  return String(key);
}

}  // namespace attendance
