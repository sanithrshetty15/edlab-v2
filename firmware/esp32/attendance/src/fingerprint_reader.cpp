#include "fingerprint_reader.h"

#include "attendance_logic.h"
#include "hardware_config.h"
#include "logger.h"

namespace attendance {
namespace {
constexpr uint8_t kCmdGetImage = 0x01;
constexpr uint8_t kCmdImage2Tz = 0x02;
constexpr uint8_t kCmdMatch = 0x03;
constexpr uint8_t kCmdSearch = 0x04;
constexpr uint8_t kCmdRegModel = 0x05;
constexpr uint8_t kCmdStore = 0x06;
constexpr uint8_t kCmdLoadChar = 0x07;
constexpr uint8_t kCmdDeleteChar = 0x0C;
constexpr uint8_t kCmdVerifyPassword = 0x13;

constexpr uint8_t kOk = 0x00;
constexpr uint8_t kErrNoFinger = 0x02;
constexpr uint8_t kErrImage = 0x06;
constexpr uint8_t kErrMatch = 0x08;
constexpr uint8_t kErrNotFound = 0x09;
constexpr uint8_t kErrEmpty = 0x0A;
constexpr uint8_t kNoConfirmation = 0xFF;

constexpr uint16_t kMaxPacketPayload = 64;

const char* confirmationName(uint8_t value) {
  switch (value) {
    case kOk:
      return "ok";
    case 0x01:
      return "packet_error";
    case kErrNoFinger:
      return "no_finger";
    case 0x03:
      return "enroll_failed";
    case 0x06:
      return "bad_image";
    case 0x07:
      return "feature_failed";
    case kErrMatch:
      return "no_match";
    case kErrNotFound:
      return "not_found";
    case kErrEmpty:
      return "empty";
    case 0x18:
      return "flash_error";
    case kNoConfirmation:
      return "timeout";
    default:
      return "reader_error";
  }
}

uint32_t timeoutFor(uint8_t command) {
  switch (command) {
    case kCmdSearch:
      return 4000;
    case kCmdRegModel:
    case kCmdStore:
    case kCmdDeleteChar:
      return 1500;
    default:
      return 1000;
  }
}
}  // namespace

FingerprintReader::FingerprintReader() : serial_(hardware::kReaderUart) {}

bool FingerprintReader::validSlot(uint16_t slot) { return isValidSlot(static_cast<int>(slot)); }

void FingerprintReader::noteResult(bool ok, bool noFinger) {
  if (ok || noFinger) {
    consecutiveErrors_ = 0;
  } else if (consecutiveErrors_ < 1000) {
    ++consecutiveErrors_;
  }
}

const char* FingerprintReader::lastErrorName() const { return confirmationName(confirmation_); }

bool FingerprintReader::begin() {
  serial_.begin(hardware::kReaderBaud, SERIAL_8N1, hardware::kReaderRxPin, hardware::kReaderTxPin);
  delay(200);  // Reader power-up only; runtime loops never delay for this.
  while (serial_.available()) {
    serial_.read();
  }
  confirmation_ = kNoConfirmation;
  const bool ok = verifyPassword();
  ready_ = ok;
  if (ok) {
    consecutiveErrors_ = 0;
    ATT_LOG_I("reader", "R30x protocol ready on UART%d (RX %d, TX %d, %lu baud)",
              hardware::kReaderUart, hardware::kReaderRxPin, hardware::kReaderTxPin,
              static_cast<unsigned long>(hardware::kReaderBaud));
  } else {
    ATT_LOG_E("reader", "no valid response on UART%d (RX %d, TX %d, %lu baud), status=%s",
              hardware::kReaderUart, hardware::kReaderRxPin, hardware::kReaderTxPin,
              static_cast<unsigned long>(hardware::kReaderBaud), lastErrorName());
  }
  return ok;
}

void FingerprintReader::end() {
  serial_.end();
  ready_ = false;
}

bool FingerprintReader::verifyPassword() {
  const uint32_t password = hardware::kReaderPassword;
  const uint8_t params[4] = {static_cast<uint8_t>((password >> 24) & 0xFF),
                             static_cast<uint8_t>((password >> 16) & 0xFF),
                             static_cast<uint8_t>((password >> 8) & 0xFF),
                             static_cast<uint8_t>(password & 0xFF)};
  size_t length = 0;
  const bool ok = execute(kCmdVerifyPassword, params, sizeof(params), nullptr, 0, length, 1000);
  noteResult(ok, false);
  return ok;
}

bool FingerprintReader::getImage() {
  size_t length = 0;
  const bool ok = execute(kCmdGetImage, nullptr, 0, nullptr, 0, length, 1000);
  noteResult(ok, confirmation_ == kErrNoFinger);
  return ok;
}

bool FingerprintReader::image2Tz(uint8_t bufferId) {
  if (bufferId < 1 || bufferId > 2) {
    return false;
  }
  const uint8_t params[1] = {bufferId};
  size_t length = 0;
  const bool ok = execute(kCmdImage2Tz, params, sizeof(params), nullptr, 0, length, 1200);
  noteResult(ok, false);
  return ok;
}

bool FingerprintReader::search(uint16_t& slot) {
  return searchRange(static_cast<uint16_t>(kFingerprintSlotMin),
                     static_cast<uint16_t>(kFingerprintCapacity), slot);
}

bool FingerprintReader::searchRange(uint16_t startSlot, uint16_t slotCount, uint16_t& slot) {
  slot = 0;
  if (!validSlot(startSlot) || slotCount == 0 ||
      static_cast<uint32_t>(startSlot) + slotCount >
          static_cast<uint32_t>(kFingerprintSlotMax) + 1) {
    return false;
  }
  const uint8_t params[5] = {0x01,
                             static_cast<uint8_t>((startSlot >> 8) & 0xFF),
                             static_cast<uint8_t>(startSlot & 0xFF),
                             static_cast<uint8_t>((slotCount >> 8) & 0xFF),
                             static_cast<uint8_t>(slotCount & 0xFF)};
  uint8_t data[4] = {0};
  size_t length = 0;
  const bool ok = execute(kCmdSearch, params, sizeof(params), data, sizeof(data), length, 4000);
  noteResult(ok, confirmation_ == kErrNotFound);
  if (length >= 2) {
    slot = static_cast<uint16_t>((data[0] << 8) | data[1]);
  }
  return ok && validSlot(slot);
}

bool FingerprintReader::match(uint16_t& score) {
  score = 0;
  uint8_t data[2] = {0};
  size_t length = 0;
  const bool ok = execute(kCmdMatch, nullptr, 0, data, sizeof(data), length, 1000);
  noteResult(ok, confirmation_ == kErrMatch);
  if (length >= 2) {
    score = static_cast<uint16_t>((data[0] << 8) | data[1]);
  }
  return ok;
}

bool FingerprintReader::regModel() {
  size_t length = 0;
  const bool ok = execute(kCmdRegModel, nullptr, 0, nullptr, 0, length, 1500);
  noteResult(ok, false);
  return ok;
}

bool FingerprintReader::storeModel(uint16_t slot) {
  if (!validSlot(slot)) {
    return false;
  }
  const uint8_t params[3] = {0x01, static_cast<uint8_t>((slot >> 8) & 0xFF),
                             static_cast<uint8_t>(slot & 0xFF)};
  size_t length = 0;
  const bool ok = execute(kCmdStore, params, sizeof(params), nullptr, 0, length, 1500);
  noteResult(ok, false);
  return ok;
}

bool FingerprintReader::loadChar(uint8_t bufferId, uint16_t slot) {
  if ((bufferId != 1 && bufferId != 2) || !validSlot(slot)) {
    return false;
  }
  const uint8_t params[3] = {bufferId, static_cast<uint8_t>((slot >> 8) & 0xFF),
                             static_cast<uint8_t>(slot & 0xFF)};
  size_t length = 0;
  const bool ok = execute(kCmdLoadChar, params, sizeof(params), nullptr, 0, length, 1000);
  const bool benignEmpty = confirmation_ == kErrEmpty || confirmation_ == kErrNotFound;
  noteResult(ok, benignEmpty);
  return ok;
}

bool FingerprintReader::deleteModel(uint16_t slot) {
  if (!validSlot(slot)) {
    return false;
  }
  const uint8_t params[2] = {static_cast<uint8_t>((slot >> 8) & 0xFF),
                             static_cast<uint8_t>(slot & 0xFF)};
  size_t length = 0;
  const bool ok = execute(kCmdDeleteChar, params, sizeof(params), nullptr, 0, length, 1500);
  const bool alreadyEmpty = confirmation_ == kErrEmpty;
  noteResult(ok || alreadyEmpty, alreadyEmpty);
  return ok || alreadyEmpty;
}

bool FingerprintReader::templateExists(uint16_t slot, bool& exists) {
  exists = false;
  if (!validSlot(slot)) {
    return false;
  }
  if (loadChar(2, slot)) {
    exists = true;
    return true;
  }
  if (confirmation_ == kErrEmpty || confirmation_ == kErrNotFound || confirmation_ == kOk) {
    return true;
  }
  return false;
}

bool FingerprintReader::execute(uint8_t command, const uint8_t* params, size_t paramLength,
                                uint8_t* data, size_t maxData, size_t& dataLength,
                                uint32_t timeoutMs) {
  dataLength = 0;
  confirmation_ = kNoConfirmation;
  if (paramLength + 12 > kMaxPacketPayload) {
    return false;
  }

  while (serial_.available()) {
    serial_.read();
  }

  uint8_t packet[kMaxPacketPayload];
  size_t index = 0;
  packet[index++] = 0xEF;
  packet[index++] = 0x01;
  packet[index++] = 0xFF;
  packet[index++] = 0xFF;
  packet[index++] = 0xFF;
  packet[index++] = 0xFF;
  packet[index++] = 0x01;
  const uint16_t length = static_cast<uint16_t>(paramLength + 3);
  packet[index++] = static_cast<uint8_t>((length >> 8) & 0xFF);
  packet[index++] = static_cast<uint8_t>(length & 0xFF);
  packet[index++] = command;
  for (size_t i = 0; i < paramLength; ++i) {
    packet[index++] = params[i];
  }
  uint16_t sum = 0;
  for (size_t i = 6; i < index; ++i) {
    sum = static_cast<uint16_t>(sum + packet[i]);
  }
  packet[index++] = static_cast<uint8_t>((sum >> 8) & 0xFF);
  packet[index++] = static_cast<uint8_t>(sum & 0xFF);

  serial_.write(packet, index);
  serial_.flush();
  return readResponse(data, maxData, dataLength, timeoutMs == 0 ? timeoutFor(command) : timeoutMs);
}

bool FingerprintReader::readResponse(uint8_t* data, size_t maxData, size_t& dataLength,
                                     uint32_t timeoutMs) {
  dataLength = 0;
  const uint32_t start = millis();
  bool headerFound = false;

  while (millis() - start < timeoutMs) {
    if (!serial_.available()) {
      delay(1);
      continue;
    }
    if (static_cast<uint8_t>(serial_.read()) != 0xEF) {
      continue;
    }
    const uint32_t innerStart = millis();
    while (millis() - innerStart < 100 && !serial_.available()) {
      delay(1);
    }
    if (!serial_.available()) {
      continue;
    }
    if (static_cast<uint8_t>(serial_.read()) == 0x01) {
      headerFound = true;
      break;
    }
  }
  if (!headerFound) {
    return false;
  }

  uint8_t head[7];
  if (serial_.readBytes(head, sizeof(head)) != sizeof(head)) {
    return false;
  }
  if (head[0] != 0xFF || head[1] != 0xFF || head[2] != 0xFF || head[3] != 0xFF) {
    return false;
  }
  const uint16_t length = static_cast<uint16_t>((head[5] << 8) | head[6]);
  if (length < 3 || length > kMaxPacketPayload) {
    return false;
  }

  uint8_t payload[kMaxPacketPayload];
  if (serial_.readBytes(payload, length) != length) {
    return false;
  }
  const uint16_t expected = static_cast<uint16_t>((payload[length - 2] << 8) | payload[length - 1]);
  uint8_t checkBuffer[kMaxPacketPayload + 2];
  size_t checkLength = 0;
  checkBuffer[checkLength++] = head[4];
  checkBuffer[checkLength++] = head[5];
  checkBuffer[checkLength++] = head[6];
  for (size_t i = 0; i + 2 < length; ++i) {
    checkBuffer[checkLength++] = payload[i];
  }
  uint16_t actual = 0;
  for (size_t i = 0; i < checkLength; ++i) {
    actual = static_cast<uint16_t>(actual + checkBuffer[i]);
  }
  if (actual != expected) {
    return false;
  }

  confirmation_ = payload[0];
  if (confirmation_ != kOk) {
    return false;
  }
  const size_t responseLength = static_cast<size_t>(length) - 3;
  if (data != nullptr && maxData > 0 && responseLength > 0) {
    const size_t copy = responseLength < maxData ? responseLength : maxData;
    for (size_t i = 0; i < copy; ++i) {
      data[i] = payload[1 + i];
    }
    dataLength = copy;
  }
  return true;
}

}  // namespace attendance
