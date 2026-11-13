#ifndef WIFI_MANAGER_H
#define WIFI_MANAGER_H

#include <Arduino.h>

// Start the station interface using credentials from config.h.
void wifiBegin();

// Reconnect when disconnected, without blocking the main loop.
void wifiMaintain();

bool isWifiConnected();

// Local IP as a string, or an empty string when disconnected.
String wifiIpAddress();

#endif // WIFI_MANAGER_H
