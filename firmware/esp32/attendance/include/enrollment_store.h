#pragma once

#include <Arduino.h>
#include <Preferences.h>
#include <cstdint>

namespace attendance {

struct EnrollmentRecovery {
  bool present = false;
  char uid[80] = {0};
  int slot = 0;
  bool templateStored = false;
};

// Small NVS record used to reconcile a locally stored template after a reboot.
// The UID is written last, so a power loss during save cannot produce a
// partially written record that points at a different session.
class EnrollmentStore {
 public:
  bool begin();
  bool load(EnrollmentRecovery& out);
  bool save(const char* uid, int slot);
  bool markTemplateStored();
  bool clear();

 private:
  Preferences prefs_;
  bool ready_ = false;
};

}  // namespace attendance
