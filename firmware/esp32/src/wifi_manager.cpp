#include "wifi_manager.h"
#include "config.h"

#include <WiFi.h>

static bool started = false;
static unsigned long lastAttempt = 0;
static bool connectionReported = false;

void wifiBegin() {
  WiFi.mode(WIFI_STA);
  WiFi.setAutoReconnect(true);

  Serial.print("Connecting to Wi-Fi: ");
  Serial.println(WIFI_SSID);

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
    if (!connectionReported) {
      Serial.println("Wi-Fi connected!");
      Serial.print("IP address: ");
      Serial.println(WiFi.localIP());
      connectionReported = true;
    }
    return;
  }

  connectionReported = false;

  if (millis() - lastAttempt >= WIFI_RETRY_INTERVAL) {
    Serial.print("Wi-Fi connection failed. Status: ");
    Serial.println(WiFi.status());
    Serial.println("Retrying Wi-Fi...");

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
