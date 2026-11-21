#include "wifi_manager.h"

#include <Arduino.h>
#include <WiFi.h>

#include "attendance_logic.h"
#include "config.h"
#include "logger.h"

namespace attendance {

void WifiManager::begin() {
  WiFi.mode(WIFI_STA);
  WiFi.persistent(false);
  WiFi.setAutoReconnect(false);
  WiFi.setSleep(false);
  attempt_ = 0;
  state_ = WifiState::Idle;
  startAttempt();
}

void WifiManager::startAttempt() {
  state_ = WifiState::Connecting;
  attemptStartedMs_ = millis();
  ATT_LOG_I("wifi", "connecting to SSID %s (attempt %u)", config::kWifiSsid,
            static_cast<unsigned>(attempt_ + 1));
  WiFi.disconnect(false, false);
  WiFi.begin(config::kWifiSsid, config::kWifiPassword);
}

void WifiManager::scheduleRetry() {
  const uint32_t delayMs = exponentialBackoffMs(attempt_, config::kWifiRetryBaseMs,
                                                config::kWifiRetryMaxMs);
  retryAtMs_ = millis() + delayMs;
  state_ = WifiState::Backoff;
  if (attempt_ < 250) {
    ++attempt_;
  }
  ATT_LOG_W("wifi", "disconnected; retrying in %lu ms", static_cast<unsigned long>(delayMs));
}

void WifiManager::update() {
  const uint32_t now = millis();
  switch (state_) {
    case WifiState::Idle:
      startAttempt();
      break;
    case WifiState::Connecting:
      if (WiFi.status() == WL_CONNECTED) {
        state_ = WifiState::Connected;
        attempt_ = 0;
        ATT_LOG_I("wifi", "connected, IP %s, RSSI %d dBm", WiFi.localIP().toString().c_str(),
                  WiFi.RSSI());
      } else if (now - attemptStartedMs_ >= config::kWifiConnectTimeoutMs) {
        WiFi.disconnect(false, false);
        scheduleRetry();
      }
      break;
    case WifiState::Connected:
      if (WiFi.status() != WL_CONNECTED) {
        attempt_ = 0;
        scheduleRetry();
      }
      break;
    case WifiState::Backoff:
      if (timeReached(now, retryAtMs_)) {
        startAttempt();
      }
      break;
  }
}

bool WifiManager::connected() const { return WiFi.status() == WL_CONNECTED; }

int WifiManager::rssi() const { return connected() ? WiFi.RSSI() : 0; }

const char* WifiManager::stateName() const {
  switch (state_) {
    case WifiState::Idle:
      return "idle";
    case WifiState::Connecting:
      return "connecting";
    case WifiState::Connected:
      return "online";
    case WifiState::Backoff:
      return "backoff";
  }
  return "unknown";
}

}  // namespace attendance
