#pragma once

#include <Arduino.h>
#include <Preferences.h>
#include <freertos/FreeRTOS.h>
#include <freertos/semphr.h>
#include <cstddef>
#include <cstdint>

#include "config.h"

namespace attendance {

struct QueueRecord {
  uint32_t sequence = 0;
  uint8_t physicalSlot = 0;
  String payload;
};

// Fixed-capacity NVS ring. There is no separate head/tail metadata that can be
// lost: begin() rebuilds the index by scanning every physical slot, so a
// confirmed write survives a power loss even if the device died immediately
// afterwards.
class OfflineQueue {
 public:
  bool begin();
  bool push(uint16_t fingerprintSlot, const char* idempotencyKey, const char* capturedAt);
  size_t count() const;
  size_t capacity() const { return config::kQueueCapacity; }

  // Copies up to max records in insertion order. Corrupt records encountered
  // here are removed and counted, never silently retained.
  size_t snapshot(QueueRecord* out, size_t max);
  bool removeBySequence(uint32_t sequence);

  uint32_t droppedCorrupt() const { return droppedCorrupt_; }
  uint32_t permanentFailures() const { return permanentFailures_; }
  void notePermanentFailure() { ++permanentFailures_; }

 private:
  struct Meta {
    bool used = false;
    uint32_t sequence = 0;
    uint8_t physical = 0;
  };

  bool readRecord(uint8_t physical, QueueRecord& out, bool& corrupt);
  void removePhysical(uint8_t physical);
  String nvsKey(uint8_t physical) const;

  Preferences prefs_;
  bool ready_ = false;
  Meta entries_[config::kQueueCapacity];
  size_t count_ = 0;
  uint32_t nextSequence_ = 1;
  uint32_t droppedCorrupt_ = 0;
  uint32_t permanentFailures_ = 0;
  SemaphoreHandle_t mutex_ = nullptr;
};

}  // namespace attendance
