#pragma once

#include <Arduino.h>
#include <HardwareSerial.h>
#include <cstddef>
#include <cstdint>

namespace attendance {

class FingerprintReader {
 public:
  FingerprintReader();

  bool begin();
  void end();
  bool ready() const { return ready_; }

  bool verifyPassword();
  bool getImage();
  bool image2Tz(uint8_t bufferId);
  bool search(uint16_t& slot);
  bool searchRange(uint16_t startSlot, uint16_t slotCount, uint16_t& slot);
  bool match(uint16_t& score);
  bool regModel();
  bool storeModel(uint16_t slot);
  bool loadChar(uint8_t bufferId, uint16_t slot);
  bool deleteModel(uint16_t slot);
  bool templateExists(uint16_t slot, bool& exists);

  uint8_t lastConfirmation() const { return confirmation_; }
  uint32_t consecutiveErrors() const { return consecutiveErrors_; }
  const char* lastErrorName() const;

  static bool validSlot(uint16_t slot);

 private:
  bool execute(uint8_t command, const uint8_t* params, size_t paramLength, uint8_t* data,
               size_t maxData, size_t& dataLength, uint32_t timeoutMs);
  bool readResponse(uint8_t* data, size_t maxData, size_t& dataLength, uint32_t timeoutMs);
  void noteResult(bool ok, bool noFinger);

  HardwareSerial serial_;
  bool ready_ = false;
  uint8_t confirmation_ = 0xFF;
  uint32_t consecutiveErrors_ = 0;
};

}  // namespace attendance
