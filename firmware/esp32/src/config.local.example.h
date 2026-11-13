#ifndef CONFIG_LOCAL_H
#define CONFIG_LOCAL_H

// Copy this file to config.local.h and fill in real values.
// config.local.h is git-ignored and must never be committed.

#define WIFI_SSID "your-wifi-ssid"
#define WIFI_PASSWORD "your-wifi-password"

#define API_BASE_URL "https://edwinslab.onrender.com"

// Device API key returned once by POST /api/v1/devices (admin-only).
// Only its SHA-256 digest is stored by the backend; never commit this file.
#define DEVICE_API_KEY "replace-with-provisioned-device-api-key"
#define DEVICE_NAME "Edwins Lab ESP32 (ALF V1.0)"

// Only set this if event HMAC signing is enabled. It must match the backend
// BIOMETRIC_API_SECRET value. Otherwise leave the default empty string.
// #define DEVICE_HMAC_SECRET "same-value-as-backend-BIOMETRIC_API_SECRET"

// Optional pin overrides if the board revision differs:
// #define LOGIN_RX 16
// #define LOGIN_TX 17
// #define LOGOUT_RX 19
// #define LOGOUT_TX 18
// #define RELAY_PIN 25
// #define EXIT_BUTTON 33
// #define GREEN_LED 23
// #define RED_LED 22
// #define BUZZER_PIN 21

#endif // CONFIG_LOCAL_H
