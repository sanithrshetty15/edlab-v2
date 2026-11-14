#ifndef CONFIG_H
#define CONFIG_H

// ===========================================================================
// Single source of configuration for the Edwins Lab ESP32 firmware.
//
// Real credentials must live in src/config.local.h, which is git-ignored.
// Copy src/config.local.example.h to src/config.local.h and fill it in.
// Never commit Wi-Fi passwords, device API keys or HMAC secrets.
// ===========================================================================

#if __has_include("config.local.h")
#include "config.local.h"
#endif

// ===== WI-FI =====
#ifndef WIFI_SSID
#define WIFI_SSID "CHANGE_ME_WIFI_SSID"
#endif
#ifndef WIFI_PASSWORD
#define WIFI_PASSWORD "CHANGE_ME_WIFI_PASSWORD"
#endif

// ===== BACKEND API =====
#ifndef API_BASE_URL
#define API_BASE_URL "https://edwinslab.onrender.com"
#endif

// Device API key provisioned by the backend (hashed server-side).
// The backend exchanges this key for a short-lived X-Device-Token.
#ifndef DEVICE_API_KEY
#define DEVICE_API_KEY "CHANGE_ME_DEVICE_API_KEY"
#endif

// Friendly name reported during device authentication.
#ifndef DEVICE_NAME
#define DEVICE_NAME "Edwins Lab ESP32"
#endif

// Shared secret used for optional event HMAC signatures. Must match the
// backend BIOMETRIC_API_SECRET when HMAC signing is used.
#ifndef DEVICE_HMAC_SECRET
#define DEVICE_HMAC_SECRET ""
#endif

// ===== HARDWARE PINOUT (ALF V1.0 board, see README pin table) =====
// Fingerprint reader 1 (login) on UART2
#ifndef LOGIN_UART
#define LOGIN_UART 2
#endif
#ifndef LOGIN_RX
#define LOGIN_RX 16
#endif
#ifndef LOGIN_TX
#define LOGIN_TX 17
#endif

// Fingerprint reader 2 (logout) on UART1
#ifndef LOGOUT_UART
#define LOGOUT_UART 1
#endif
#ifndef LOGOUT_RX
#define LOGOUT_RX 19
#endif
#ifndef LOGOUT_TX
#define LOGOUT_TX 18
#endif

// ===== FINGERPRINT ENROLLMENT READER =====
// Enrollment is disabled unless explicitly pointed at one of the two readers.
// The default is deliberately fail-closed: an unconfigured controller never
// claims enrollment commands and never falls back to a login/logout reader.
#define ENROLLMENT_READER_NONE 0
#define ENROLLMENT_READER_LOGIN 1
#define ENROLLMENT_READER_LOGOUT 2
#ifndef ENROLLMENT_READER
#define ENROLLMENT_READER ENROLLMENT_READER_NONE
#endif
#if ENROLLMENT_READER != ENROLLMENT_READER_NONE && \
    ENROLLMENT_READER != ENROLLMENT_READER_LOGIN && \
    ENROLLMENT_READER != ENROLLMENT_READER_LOGOUT
#error "ENROLLMENT_READER must be NONE, LOGIN or LOGOUT"
#endif

// How often the controller asks the backend for enrollment work and checks
// whether an active session was cancelled or expired.
#ifndef ENROLLMENT_POLL_INTERVAL
#define ENROLLMENT_POLL_INTERVAL 2000UL
#endif
#ifndef ENROLLMENT_RECONCILE_INTERVAL
#define ENROLLMENT_RECONCILE_INTERVAL 5000UL
#endif
#ifndef ENROLLMENT_SCAN_INTERVAL
#define ENROLLMENT_SCAN_INTERVAL 300UL
#endif

// Electromagnetic lock relay.
// Fail-safe wiring: the lock is energised while locked and releases when the
// relay is de-energised, so an emergency exit works without Wi-Fi or power.
#ifndef RELAY_PIN
#define RELAY_PIN 25
#endif
#ifndef RELAY_LOCKED_LEVEL
#define RELAY_LOCKED_LEVEL HIGH
#endif
#ifndef RELAY_UNLOCK_LEVEL
#define RELAY_UNLOCK_LEVEL LOW
#endif

// Emergency exit button, active LOW (INPUT_PULLUP).
#ifndef EXIT_BUTTON
#define EXIT_BUTTON 33
#endif

// Status indicators.
#ifndef GREEN_LED
#define GREEN_LED 23
#endif
#ifndef RED_LED
#define RED_LED 22
#endif
#ifndef BUZZER_PIN
#define BUZZER_PIN 21
#endif

// ===== FIRMWARE TIMING =====
#ifndef DOOR_OPEN_MS
#define DOOR_OPEN_MS 5000UL
#endif
#ifndef HEARTBEAT_INTERVAL
#define HEARTBEAT_INTERVAL 30000UL
#endif
#ifndef WIFI_RETRY_INTERVAL
#define WIFI_RETRY_INTERVAL 15000UL
#endif
#ifndef AUTH_RETRY_INTERVAL
#define AUTH_RETRY_INTERVAL 10000UL
#endif
#ifndef SCAN_DEBOUNCE_MS
#define SCAN_DEBOUNCE_MS 750UL
#endif
#ifndef EXIT_DEBOUNCE_MS
#define EXIT_DEBOUNCE_MS 500UL
#endif

// Longest number of local enrollment mappings cached from the backend.
#ifndef MAX_ENROLLMENTS
#define MAX_ENROLLMENTS 64
#endif

#endif // CONFIG_H
