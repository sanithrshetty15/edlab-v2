#include "wifi_manager.h"

#include "config.h"

#include <WiFi.h>

static bool started = false;
static unsigned long lastAttempt = 0;

void wifiBegin() {
  WiFi.mode(WIFI_STA);
  WiFi.setAutoReconnect(true);
  WiFi.begin(WIFI_SSID, WIFI_PASSWORD);
  lastAttempt = millis();
  started = true;
}

void wifiMaintain() {
  if (!started) {
    wifiBegin();
    return;
  }
  if (WiFi.status() == WL_CONNECTED) {
    return;
  }
  if (millis() - lastAttempt >= WIFI_RETRY_INTERVAL) {
    WiFi.disconnect(false, false);
    WiFi.begin(WIFI_SSID, WIFI_PASSWORD);
    lastAttempt = millis();
  }
}

bool isWifiConnected() {
  return WiFi.status() == WL_CONNECTED;
}

String wifiIpAddress() {
  return isWifiConnected() ? WiFi.localIP().toString() : String("");
}
