#include <Arduino.h>

#include "app.h"

namespace {
attendance::App gApp;
}

void setup() { gApp.setup(); }

void loop() { gApp.loop(); }
