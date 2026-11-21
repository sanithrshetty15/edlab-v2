// Copy this file to include/attendance_secrets.h and fill in deployment values.
// attendance_secrets.h is git-ignored. Never commit real credentials.
//
// The API key belongs to a device provisioned with:
//   POST /api/v1/devices
//   {"name":"...","device_type":"attendance_terminal"}
//
// The key is exchanged for a short-lived X-Device-Token and is never written to
// logs. After a backend key rotation, replace it and reflash.

#define WIFI_SSID "your-wifi-ssid"
#define WIFI_PASSWORD "your-wifi-password"

#define API_BASE_URL "https://edwinslab.onrender.com"
#define DEVICE_API_KEY "paste-the-provisioned-attendance-device-key"
#define DEVICE_NAME "Edwins Lab attendance terminal"

// Optional hardware overrides. The defaults come from the documented ALF V1.0
// controller map, but the attended PCB still needs physical verification.
// #define ATTENDANCE_READER_MODEL "R307S"
// #define ATTENDANCE_READER_BAUD 57600
// #define ATTENDANCE_READER_RX 16
// #define ATTENDANCE_READER_TX 17
// #define ATTENDANCE_BUZZER_PIN 21
// #define ATTENDANCE_BUZZER_ACTIVE_HIGH 1

// Enable only if the attendance PCB is confirmed to have status LEDs.
// #define ATTENDANCE_HAS_STATUS_LEDS 1
// #define ATTENDANCE_LED_GREEN_PIN 23
// #define ATTENDANCE_LED_RED_PIN 22
// #define ATTENDANCE_LED_ACTIVE_HIGH 1

// 0=ERROR, 1=WARN, 2=INFO, 3=DEBUG. Keep 2 or lower in production.
// #define ATTENDANCE_LOG_LEVEL 2
