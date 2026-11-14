#ifndef R307S_H
#define R307S_H

#include <Arduino.h>

// R30x / R307S instruction codes (standard fingerprint module protocol).
#define R307S_GENIMG        0x01
#define R307S_IMG2TZ        0x02
#define R307S_MATCH         0x03
#define R307S_SEARCH        0x04
#define R307S_REGMODEL      0x05
#define R307S_STORE         0x06
#define R307S_LOADCHAR      0x07
#define R307S_UPLOADCHAR    0x08
#define R307S_DELCHAR       0x0C
#define R307S_EMPTY         0x0D
#define R307S_VERIFYPWD     0x13

// Confirmation codes returned by the module.
#define R307S_OK            0x00
#define R307S_ERR_RECV      0x01
#define R307S_ERR_NOFINGER  0x02
#define R307S_ERR_ENROLL    0x03
#define R307S_ERR_IMAGE     0x06
#define R307S_ERR_MATCH     0x08
#define R307S_ERR_NOTFOUND  0x09
#define R307S_ERR_EMPTY     0x0A
#define R307S_ERR_FLASH     0x18
#define R307S_NO_CONFIRMATION 0xFF

// Default password of a factory module.
#define R307S_DEFAULT_PASSWORD 0x00000000UL

class R307S {
public:
  // uartNum is the ESP32 UART peripheral (1 or 2). Each instance owns its own
  // HardwareSerial and pins, so two readers can run independently.
  R307S(int uartNum, int rxPin, int txPin);

  // Start the UART at 57600 baud and verify the module password.
  bool begin();

  // Capture an image. The buffer argument is accepted for API compatibility
  // and may be nullptr.
  bool getImage(uint8_t* buffer);

  // Convert the stored image to a character buffer. The first byte of buffer,
  // when provided, selects the buffer slot (1 or 2).
  bool img2tz(uint8_t* buffer);
  bool img2tz(uint8_t bufferId);

  // Combine the two character buffers into a template.
  bool regModel();

  // Persist the current template into a module slot.
  bool storeModel(uint16_t slotId);

  // Search the module for a matching template. Returns true and the slot id.
  bool search(uint16_t& fingerId);
  bool searchRange(uint16_t startSlot, uint16_t slotCount, uint16_t& fingerId);
  bool searchMatch(uint8_t* buffer, int& id);

  // Compare character buffers 1 and 2. Returns true on a match and the score.
  bool match(uint16_t& score);

  // Load a stored template into a character buffer (1 or 2).
  bool loadChar(uint8_t bufferId, uint16_t slotId);

  // Check whether a slot currently contains a template. Returns false only
  // when the sensor could not be queried reliably.
  bool templateExists(uint16_t slotId, bool& exists);

  // Delete the template stored in a slot.
  bool deleteModel(uint16_t slotId);

  // Confirmation code from the most recent command (0xFF when unknown).
  uint8_t lastConfirmation() const;

  // Not all R30x clones support these; kept for API compatibility.
  int getTemplateID();
  bool uploadChar(uint8_t* buffer);
  bool genISO(uint8_t* buffer);
  bool getStatus(uint8_t& status);

private:
  int uartNum_;
  int rxPin_;
  int txPin_;
  HardwareSerial* serial_;
  uint8_t lastConfirmation_;

  bool verifyPassword(uint32_t password);

  uint16_t checksum16(const uint8_t* data, size_t len) const;
  bool execute(uint8_t command, const uint8_t* param, size_t paramLen,
               uint8_t* out, size_t maxOut, size_t& outLen);
  bool readResponse(uint8_t* out, size_t maxOut, size_t& outLen);
};

#endif // R307S_H
