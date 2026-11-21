#pragma once

#include <cstdint>

// ---------------------------------------------------------------------------
// ALF V1.0 hardware map
// ---------------------------------------------------------------------------
// The reader UART and feedback pins below are the documented ALF V1.0
// controller values (firmware/esp32/README.md). They are defaults, not a
// substitute for checking the actual attendance PCB. Every value can be
// overridden from include/attendance_secrets.h.
//
// Confirmed by the hardware brief: ESP32-WROOM-32UE, 2.4 GHz Wi-Fi, onboard
// buzzer, UART fingerprint reader, 5 V input / 3.3 V logic.
//
// Still unverified for this PCB: exact reader model/protocol/baud (the driver
// implements the standard R30x protocol), buzzer polarity, and whether status
// LEDs are populated. LEDs are therefore disabled by default.

#ifndef ATTENDANCE_READER_MODEL
#define ATTENDANCE_READER_MODEL "R30x (model unverified)"
#endif

#ifndef ATTENDANCE_READER_UART
#define ATTENDANCE_READER_UART 2
#endif
#ifndef ATTENDANCE_READER_RX
#define ATTENDANCE_READER_RX 16
#endif
#ifndef ATTENDANCE_READER_TX
#define ATTENDANCE_READER_TX 17
#endif
#ifndef ATTENDANCE_READER_BAUD
#define ATTENDANCE_READER_BAUD 57600UL
#endif
#ifndef ATTENDANCE_READER_PASSWORD
#define ATTENDANCE_READER_PASSWORD 0x00000000UL
#endif

#ifndef ATTENDANCE_BUZZER_PIN
#define ATTENDANCE_BUZZER_PIN 21
#endif
#ifndef ATTENDANCE_BUZZER_ACTIVE_HIGH
#define ATTENDANCE_BUZZER_ACTIVE_HIGH 1
#endif

#ifndef ATTENDANCE_HAS_STATUS_LEDS
#define ATTENDANCE_HAS_STATUS_LEDS 0
#endif
#ifndef ATTENDANCE_LED_GREEN_PIN
#define ATTENDANCE_LED_GREEN_PIN 23
#endif
#ifndef ATTENDANCE_LED_RED_PIN
#define ATTENDANCE_LED_RED_PIN 22
#endif
#ifndef ATTENDANCE_LED_ACTIVE_HIGH
#define ATTENDANCE_LED_ACTIVE_HIGH 1
#endif

namespace attendance {
namespace hardware {

constexpr char kReaderModel[] = ATTENDANCE_READER_MODEL;
constexpr int kReaderUart = ATTENDANCE_READER_UART;
constexpr int kReaderRxPin = ATTENDANCE_READER_RX;
constexpr int kReaderTxPin = ATTENDANCE_READER_TX;
constexpr uint32_t kReaderBaud = ATTENDANCE_READER_BAUD;
constexpr uint32_t kReaderPassword = ATTENDANCE_READER_PASSWORD;

constexpr int kBuzzerPin = ATTENDANCE_BUZZER_PIN;
constexpr bool kBuzzerActiveHigh = ATTENDANCE_BUZZER_ACTIVE_HIGH != 0;

constexpr bool kStatusLedsEnabled = ATTENDANCE_HAS_STATUS_LEDS != 0;
constexpr int kGreenLedPin = ATTENDANCE_LED_GREEN_PIN;
constexpr int kRedLedPin = ATTENDANCE_LED_RED_PIN;
constexpr bool kLedActiveHigh = ATTENDANCE_LED_ACTIVE_HIGH != 0;

static_assert(kReaderUart == 1 || kReaderUart == 2,
              "ATTENDANCE_READER_UART must be 1 or 2 on ESP32");
static_assert(kReaderRxPin >= 0 && kReaderTxPin >= 0,
              "Reader RX/TX pins must be valid GPIO numbers");
static_assert(kBuzzerPin >= -1, "Buzzer pin must be -1 or a valid GPIO");
static_assert(!kStatusLedsEnabled || (kGreenLedPin >= 0 && kRedLedPin >= 0),
              "Enabled status LEDs require valid GPIO pins");

}  // namespace hardware
}  // namespace attendance
