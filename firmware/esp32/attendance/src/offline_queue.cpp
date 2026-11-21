#include "offline_queue.h"

#include <ArduinoJson.h>

#include "attendance_logic.h"
#include "config.h"
#include "logger.h"

namespace attendance {
namespace {
constexpr const char* kNamespace = "attendance";
constexpr const char* kKeyPrefix = "e";
constexpr uint8_t kCapacity = config::kQueueCapacity;

void lock(SemaphoreHandle_t mutex) {
  if (mutex != nullptr) {
    xSemaphoreTake(mutex, portMAX_DELAY);
  }
}

void releaseLock(SemaphoreHandle_t mutex) {
  if (mutex != nullptr) {
    xSemaphoreGive(mutex);
  }
}
}  // namespace

String OfflineQueue::nvsKey(uint8_t physical) const {
  return String(kKeyPrefix) + String(static_cast<unsigned int>(physical));
}

bool OfflineQueue::begin() {
  if (mutex_ == nullptr) {
    mutex_ = xSemaphoreCreateMutex();
  }
  ready_ = prefs_.begin(kNamespace, false);
  if (!ready_) {
    ATT_LOG_E("queue", "NVS unavailable; offline attendance cannot be preserved");
    return false;
  }

  for (uint8_t i = 0; i < kCapacity; ++i) {
    entries_[i] = Meta();
  }
  count_ = 0;
  nextSequence_ = 1;

  for (uint8_t physical = 0; physical < kCapacity; ++physical) {
    if (!prefs_.isKey(nvsKey(physical).c_str())) {
      continue;
    }
    QueueRecord record;
    bool corrupt = false;
    if (!readRecord(physical, record, corrupt) || corrupt) {
      prefs_.remove(nvsKey(physical).c_str());
      ++droppedCorrupt_;
      ATT_LOG_E("queue", "removed corrupt NVS entry %u", static_cast<unsigned>(physical));
      continue;
    }
    bool duplicateSequence = false;
    for (const Meta& existing : entries_) {
      if (existing.used && existing.sequence == record.sequence) {
        duplicateSequence = true;
        break;
      }
    }
    if (duplicateSequence) {
      prefs_.remove(nvsKey(physical).c_str());
      ++droppedCorrupt_;
      ATT_LOG_E("queue", "removed duplicate sequence %lu from slot %u",
                static_cast<unsigned long>(record.sequence), static_cast<unsigned>(physical));
      continue;
    }
    Meta& meta = entries_[physical];
    meta.used = true;
    meta.sequence = record.sequence;
    meta.physical = physical;
    ++count_;
    if (record.sequence >= nextSequence_) {
      nextSequence_ = record.sequence + 1;
    }
  }

  ATT_LOG_I("queue", "ready: %u pending, %lu corrupt dropped", static_cast<unsigned>(count_),
            static_cast<unsigned long>(droppedCorrupt_));
  return true;
}

bool OfflineQueue::readRecord(uint8_t physical, QueueRecord& out, bool& corrupt) {
  corrupt = false;
  const String stored = prefs_.getString(nvsKey(physical).c_str(), "");
  if (stored.length() == 0 || stored.length() > config::kMaxQueuePayloadBytes + 80) {
    corrupt = true;
    return false;
  }

  JsonDocument wrapper;
  const DeserializationError error = deserializeJson(wrapper, stored);
  if (error || !wrapper["seq"].is<uint32_t>() || !wrapper["event"].is<JsonObject>()) {
    corrupt = true;
    return false;
  }
  const uint32_t sequence = wrapper["seq"].as<uint32_t>();
  JsonObjectConst event = wrapper["event"].as<JsonObjectConst>();
  const char* key = event["idempotency_key"] | "";
  const int slot = event["fingerprint_slot"] | 0;
  if (sequence == 0 || !isValidIdempotencyKey(key) || !isValidSlot(slot)) {
    corrupt = true;
    return false;
  }

  String payload;
  serializeJson(event, payload);
  if (payload.length() == 0 || payload.length() > config::kMaxQueuePayloadBytes) {
    corrupt = true;
    return false;
  }
  out.physicalSlot = physical;
  out.sequence = sequence;
  out.payload = payload;
  return true;
}

bool OfflineQueue::push(uint16_t fingerprintSlot, const char* idempotencyKey,
                        const char* capturedAt) {
  if (!ready_ || !isValidSlot(fingerprintSlot) || !isValidIdempotencyKey(idempotencyKey)) {
    return false;
  }

  lock(mutex_);
  if (count_ >= kCapacity) {
    releaseLock(mutex_);
    ATT_LOG_E("queue", "full (%u); event not queued", static_cast<unsigned>(kCapacity));
    return false;
  }
  int physical = -1;
  for (uint8_t i = 0; i < kCapacity; ++i) {
    if (!entries_[i].used) {
      physical = i;
      break;
    }
  }
  if (physical < 0) {
    releaseLock(mutex_);
    return false;
  }

  JsonDocument wrapper;
  wrapper["seq"] = nextSequence_;
  JsonObject event = wrapper["event"].to<JsonObject>();
  event["idempotency_key"] = idempotencyKey;
  event["fingerprint_slot"] = fingerprintSlot;
  if (capturedAt != nullptr && capturedAt[0] != '\0') {
    event["captured_at"] = capturedAt;
  }
  String stored;
  serializeJson(wrapper, stored);
  if (stored.length() > config::kMaxQueuePayloadBytes + 80) {
    releaseLock(mutex_);
    ATT_LOG_E("queue", "event document too large; not queued");
    return false;
  }

  const String key = nvsKey(static_cast<uint8_t>(physical));
  if (prefs_.putString(key.c_str(), stored) == 0) {
    releaseLock(mutex_);
    ATT_LOG_E("queue", "NVS write failed; event not queued");
    return false;
  }

  Meta& meta = entries_[physical];
  meta.used = true;
  meta.sequence = nextSequence_;
  meta.physical = static_cast<uint8_t>(physical);
  ++nextSequence_;
  ++count_;
  const size_t depth = count_;
  releaseLock(mutex_);
  ATT_LOG_I("queue", "queued event, depth %u", static_cast<unsigned>(depth));
  return true;
}

size_t OfflineQueue::count() const {
  lock(mutex_);
  const size_t value = count_;
  releaseLock(mutex_);
  return value;
}

size_t OfflineQueue::snapshot(QueueRecord* out, size_t max) {
  if (out == nullptr || max == 0) {
    return 0;
  }
  lock(mutex_);

  // Insertion sort the used descriptors by sequence. At most 32 entries.
  uint8_t order[kCapacity];
  size_t used = 0;
  for (uint8_t i = 0; i < kCapacity; ++i) {
    if (!entries_[i].used) {
      continue;
    }
    size_t position = used;
    while (position > 0 && entries_[order[position - 1]].sequence > entries_[i].sequence) {
      order[position] = order[position - 1];
      --position;
    }
    order[position] = i;
    ++used;
  }

  size_t written = 0;
  for (size_t i = 0; i < used && written < max; ++i) {
    QueueRecord record;
    bool corrupt = false;
    if (!readRecord(entries_[order[i]].physical, record, corrupt) || corrupt) {
      const uint8_t physical = entries_[order[i]].physical;
      prefs_.remove(nvsKey(physical).c_str());
      entries_[physical] = Meta();
      --count_;
      ++droppedCorrupt_;
      ATT_LOG_E("queue", "dropped corrupt pending entry %u",
                static_cast<unsigned>(physical));
      continue;
    }
    out[written++] = record;
  }
  releaseLock(mutex_);
  return written;
}

bool OfflineQueue::removeBySequence(uint32_t sequence) {
  lock(mutex_);
  for (uint8_t i = 0; i < kCapacity; ++i) {
    if (!entries_[i].used || entries_[i].sequence != sequence) {
      continue;
    }
    const uint8_t physical = entries_[i].physical;
    prefs_.remove(nvsKey(physical).c_str());
    entries_[i] = Meta();
    --count_;
    releaseLock(mutex_);
    return true;
  }
  releaseLock(mutex_);
  return false;
}

}  // namespace attendance
