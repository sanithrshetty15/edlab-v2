#pragma once

#include <cstddef>
#include <cstdint>

// Local deployment values. This file is git-ignored and must never be
// committed. Copy attendance_secrets.example.h to include/attendance_secrets.h.
#if __has_include("attendance_secrets.h")
#include "attendance_secrets.h"
#endif

#ifndef WIFI_SSID
#define WIFI_SSID "CHANGE_ME_WIFI_SSID"
#endif
#ifndef WIFI_PASSWORD
#define WIFI_PASSWORD "CHANGE_ME_WIFI_PASSWORD"
#endif
#ifndef API_BASE_URL
#define API_BASE_URL "https://edwinslab.onrender.com"
#endif
#ifndef DEVICE_API_KEY
#define DEVICE_API_KEY "CHANGE_ME_DEVICE_API_KEY"
#endif
#ifndef DEVICE_NAME
#define DEVICE_NAME "Edwins Lab attendance terminal"
#endif
#ifndef ATTENDANCE_LOG_LEVEL
#define ATTENDANCE_LOG_LEVEL 2  // 0=ERROR, 1=WARN, 2=INFO, 3=DEBUG
#endif
#ifndef ATTENDANCE_ALLOW_INSECURE_HTTP
#define ATTENDANCE_ALLOW_INSECURE_HTTP 0
#endif

namespace attendance {
namespace config {

constexpr char kFirmwareVersion[] = "attendance-2.0.0";
constexpr char kWifiSsid[] = WIFI_SSID;
constexpr char kWifiPassword[] = WIFI_PASSWORD;
constexpr char kApiBaseUrl[] = API_BASE_URL;
constexpr char kDeviceApiKey[] = DEVICE_API_KEY;
constexpr char kDeviceName[] = DEVICE_NAME;

// Timing. All values are milliseconds unless named otherwise.
constexpr uint32_t kScanPollIntervalMs = 80;
constexpr uint32_t kScanCooldownMs = 3000;  // re-arm only after removal + this
constexpr uint32_t kFingerRemovalTimeoutMs = 8000;
constexpr uint32_t kReaderReinitIntervalMs = 5000;
constexpr uint32_t kReaderMaxConsecutiveErrors = 3;

constexpr uint32_t kHeartbeatIntervalMs = 30000;
constexpr uint32_t kQueueFlushIntervalMs = 15000;
constexpr uint32_t kQueueFlushMaxBackoffMs = 300000;
constexpr uint32_t kAuthRetryBaseMs = 30000;
constexpr uint32_t kAuthRetryMaxMs = 300000;
constexpr uint32_t kWifiRetryBaseMs = 1000;
constexpr uint32_t kWifiRetryMaxMs = 60000;
constexpr uint32_t kWifiConnectTimeoutMs = 20000;

constexpr int kHttpTimeoutMs = 6000;
constexpr int kHttpConnectTimeoutMs = 4000;
constexpr int kEnrollmentHttpTimeoutMs = 5000;
constexpr size_t kMaxHttpBodyBytes = 4096;

constexpr uint32_t kEnrollmentFingerTimeoutMs = 30000;
constexpr uint32_t kEnrollmentRemovalTimeoutMs = 15000;
constexpr uint8_t kEnrollmentCompleteAttempts = 3;
constexpr uint8_t kEnrollmentMatchSlotsPerUpdate = 8;

// Offline queue. The backend batch limit is 50; keep the batch comfortably
// below it so one JSON document stays inside the ESP32 heap budget.
constexpr uint8_t kQueueCapacity = 32;
constexpr uint8_t kQueueBatchLimit = 20;
constexpr size_t kMaxQueuePayloadBytes = 220;

constexpr uint32_t kNetworkTaskStackBytes = 16384;
constexpr uint32_t kNetworkTaskPriority = 1;
constexpr uint8_t kNetworkTaskCore = 0;

constexpr bool kInsecureHttpAllowed = ATTENDANCE_ALLOW_INSECURE_HTTP != 0;

static_assert(kQueueBatchLimit <= 50, "Backend batch limit is 50 scans");
static_assert(kQueueCapacity >= kQueueBatchLimit, "Queue must hold one full batch");

}  // namespace config
}  // namespace attendance
