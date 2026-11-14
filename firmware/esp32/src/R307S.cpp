#include "R307S.h"

R307S::R307S(int uartNum, int rxPin, int txPin)
    : uartNum_(uartNum),
      rxPin_(rxPin),
      txPin_(txPin),
      serial_(nullptr),
      lastConfirmation_(R307S_NO_CONFIRMATION) {}

bool R307S::begin() {
  if (serial_ == nullptr) {
    serial_ = new HardwareSerial(uartNum_);
  }
  serial_->begin(57600, SERIAL_8N1, rxPin_, txPin_);
  delay(200);
  return verifyPassword(R307S_DEFAULT_PASSWORD);
}

bool R307S::verifyPassword(uint32_t password) {
  uint8_t param[4] = {
      static_cast<uint8_t>((password >> 24) & 0xFF),
      static_cast<uint8_t>((password >> 16) & 0xFF),
      static_cast<uint8_t>((password >> 8) & 0xFF),
      static_cast<uint8_t>(password & 0xFF),
  };
  size_t outLen = 0;
  return execute(R307S_VERIFYPWD, param, sizeof(param), nullptr, 0, outLen);
}

uint16_t R307S::checksum16(const uint8_t* data, size_t len) const {
  uint16_t sum = 0;
  for (size_t i = 0; i < len; i++) {
    sum = static_cast<uint16_t>(sum + data[i]);
  }
  return sum;
}

bool R307S::execute(uint8_t command, const uint8_t* param, size_t paramLen,
                    uint8_t* out, size_t maxOut, size_t& outLen) {
  lastConfirmation_ = R307S_NO_CONFIRMATION;
  if (serial_ == nullptr) {
    return false;
  }

  // Command packet:
  // 0xEF 0x01 | 4-byte address | package id | length(2) | command | params | checksum(2)
  const size_t maxPacket = 64;
  if (paramLen + 12 > maxPacket) {
    return false;
  }
  uint8_t packet[maxPacket];
  size_t idx = 0;
  packet[idx++] = 0xEF;
  packet[idx++] = 0x01;
  packet[idx++] = 0xFF;
  packet[idx++] = 0xFF;
  packet[idx++] = 0xFF;
  packet[idx++] = 0xFF;
  packet[idx++] = 0x01;
  uint16_t length = static_cast<uint16_t>(paramLen + 3); // command + checksum
  packet[idx++] = static_cast<uint8_t>((length >> 8) & 0xFF);
  packet[idx++] = static_cast<uint8_t>(length & 0xFF);
  packet[idx++] = command;
  for (size_t i = 0; i < paramLen; i++) {
    packet[idx++] = param[i];
  }
  uint16_t sum = checksum16(packet + 6, idx - 6);
  packet[idx++] = static_cast<uint8_t>((sum >> 8) & 0xFF);
  packet[idx++] = static_cast<uint8_t>(sum & 0xFF);

  serial_->write(packet, idx);
  serial_->flush();

  return readResponse(out, maxOut, outLen);
}

bool R307S::readResponse(uint8_t* out, size_t maxOut, size_t& outLen) {
  outLen = 0;
  lastConfirmation_ = R307S_NO_CONFIRMATION;
  if (serial_ == nullptr) {
    return false;
  }

  // Wait for the 0xEF 0x01 header.
  unsigned long start = millis();
  bool haveHeader = false;
  while (millis() - start < 1000) {
    if (!serial_->available()) {
      continue;
    }
    uint8_t b = static_cast<uint8_t>(serial_->read());
    if (b != 0xEF) {
      continue;
    }
    unsigned long inner = millis();
    while (millis() - inner < 100 && !serial_->available()) {
      // brief wait for the second header byte
    }
    if (!serial_->available()) {
      continue;
    }
    if (static_cast<uint8_t>(serial_->read()) == 0x01) {
      haveHeader = true;
      break;
    }
  }
  if (!haveHeader) {
    return false;
  }

  // Address (4) + package id (1) + length (2).
  uint8_t head[7];
  if (serial_->readBytes(head, sizeof(head)) != sizeof(head)) {
    return false;
  }
  uint8_t packageId = head[4];
  uint16_t length = static_cast<uint16_t>((head[5] << 8) | head[6]);
  if (length < 3 || length > 64) {
    return false;
  }

  uint8_t payload[64];
  if (serial_->readBytes(payload, length) != length) {
    return false;
  }

  uint16_t expected = static_cast<uint16_t>((payload[length - 2] << 8) | payload[length - 1]);
  uint8_t checkBuf[64];
  size_t checkLen = 0;
  checkBuf[checkLen++] = packageId;
  checkBuf[checkLen++] = head[5];
  checkBuf[checkLen++] = head[6];
  for (size_t i = 0; i + 2 < length; i++) {
    checkBuf[checkLen++] = payload[i];
  }
  if (checksum16(checkBuf, checkLen) != expected) {
    return false;
  }

  uint8_t confirmation = payload[0];
  lastConfirmation_ = confirmation;
  size_t dataLen = static_cast<size_t>(length) - 3; // minus confirmation + checksum
  if (out != nullptr && maxOut > 0) {
    size_t copy = dataLen < maxOut ? dataLen : maxOut;
    for (size_t i = 0; i < copy; i++) {
      out[i] = payload[1 + i];
    }
    outLen = copy;
  }
  return confirmation == R307S_OK;
}

bool R307S::getImage(uint8_t* buffer) {
  (void)buffer;
  size_t outLen = 0;
  return execute(R307S_GENIMG, nullptr, 0, nullptr, 0, outLen);
}

bool R307S::img2tz(uint8_t* buffer) {
  uint8_t bufferId = (buffer != nullptr && buffer[0] >= 1 && buffer[0] <= 2) ? buffer[0] : 1;
  return img2tz(bufferId);
}

bool R307S::img2tz(uint8_t bufferId) {
  uint8_t param[1] = {bufferId};
  size_t outLen = 0;
  return execute(R307S_IMG2TZ, param, sizeof(param), nullptr, 0, outLen);
}

bool R307S::regModel() {
  size_t outLen = 0;
  return execute(R307S_REGMODEL, nullptr, 0, nullptr, 0, outLen);
}

bool R307S::storeModel(uint16_t slotId) {
  uint8_t param[3] = {
      0x01, // source character buffer
      static_cast<uint8_t>((slotId >> 8) & 0xFF),
      static_cast<uint8_t>(slotId & 0xFF),
  };
  size_t outLen = 0;
  return execute(R307S_STORE, param, sizeof(param), nullptr, 0, outLen);
}

bool R307S::search(uint16_t& fingerId) {
  return searchRange(1, 127, fingerId);
}

bool R307S::searchRange(uint16_t startSlot, uint16_t slotCount, uint16_t& fingerId) {
  fingerId = 0;
  if (startSlot < 1 || startSlot > 127 || slotCount == 0 || slotCount > 127 ||
      static_cast<uint32_t>(startSlot) + slotCount > 128) {
    return false;
  }
  uint8_t param[5] = {
      0x01, // character buffer 1
      static_cast<uint8_t>((startSlot >> 8) & 0xFF),
      static_cast<uint8_t>(startSlot & 0xFF),
      static_cast<uint8_t>((slotCount >> 8) & 0xFF),
      static_cast<uint8_t>(slotCount & 0xFF),
  };
  uint8_t data[4];
  size_t outLen = 0;
  bool ok = execute(R307S_SEARCH, param, sizeof(param), data, sizeof(data), outLen);
  if (outLen >= 2) {
    fingerId = static_cast<uint16_t>((data[0] << 8) | data[1]);
  }
  return ok && fingerId > 0 && fingerId < 0xFFFF;
}

bool R307S::searchMatch(uint8_t* buffer, int& id) {
  (void)buffer;
  uint16_t fingerId = 0;
  if (!search(fingerId)) {
    return false;
  }
  id = fingerId;
  return true;
}

bool R307S::match(uint16_t& score) {
  score = 0;
  uint8_t data[2];
  size_t outLen = 0;
  bool ok = execute(R307S_MATCH, nullptr, 0, data, sizeof(data), outLen);
  if (outLen >= 2) {
    score = static_cast<uint16_t>((data[0] << 8) | data[1]);
  }
  return ok;
}

bool R307S::loadChar(uint8_t bufferId, uint16_t slotId) {
  uint8_t param[3] = {
      bufferId,
      static_cast<uint8_t>((slotId >> 8) & 0xFF),
      static_cast<uint8_t>(slotId & 0xFF),
  };
  size_t outLen = 0;
  return execute(R307S_LOADCHAR, param, sizeof(param), nullptr, 0, outLen);
}

bool R307S::templateExists(uint16_t slotId, bool& exists) {
  exists = false;
  if (slotId < 1 || slotId > 127) {
    return false;
  }
  if (loadChar(2, slotId)) {
    exists = true;
    return true;
  }
  if (lastConfirmation_ == R307S_ERR_EMPTY || lastConfirmation_ == R307S_ERR_NOTFOUND) {
    return true;
  }
  return false;
}

bool R307S::deleteModel(uint16_t slotId) {
  if (slotId < 1 || slotId > 127) {
    return false;
  }
  uint8_t param[2] = {
      static_cast<uint8_t>((slotId >> 8) & 0xFF),
      static_cast<uint8_t>(slotId & 0xFF),
  };
  size_t outLen = 0;
  bool ok = execute(R307S_DELCHAR, param, sizeof(param), nullptr, 0, outLen);
  return ok || lastConfirmation_ == R307S_ERR_EMPTY;
}

uint8_t R307S::lastConfirmation() const {
  return lastConfirmation_;
}

int R307S::getTemplateID() {
  return -1;
}

bool R307S::uploadChar(uint8_t* buffer) {
  uint8_t bufferId = (buffer != nullptr && buffer[0] >= 1 && buffer[0] <= 2) ? buffer[0] : 1;
  size_t outLen = 0;
  return execute(R307S_UPLOADCHAR, &bufferId, 1, nullptr, 0, outLen);
}

bool R307S::genISO(uint8_t* buffer) {
  (void)buffer;
  // ISO template generation is not supported by every R307S clone.
  return false;
}

bool R307S::getStatus(uint8_t& status) {
  (void)status;
  // Read-system-parameter is not implemented by this driver.
  return false;
}
