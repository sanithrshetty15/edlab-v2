#include "enrollment_store.h"

#include "attendance_logic.h"
#include "logger.h"

namespace attendance {
namespace {
constexpr const char* kNamespace = "attendance";
constexpr const char* kUidKey = "en_uid";
constexpr const char* kSlotKey = "en_slot";
constexpr const char* kStoredKey = "en_stored";
}  // namespace

bool EnrollmentStore::begin() {
  ready_ = prefs_.begin(kNamespace, false);
  if (!ready_) {
    ATT_LOG_E("enroll", "NVS unavailable; enrollment recovery is disabled");
  }
  return ready_;
}

bool EnrollmentStore::load(EnrollmentRecovery& out) {
  out = EnrollmentRecovery();
  if (!ready_) {
    return false;
  }
  const String uid = prefs_.getString(kUidKey, "");
  const int slot = prefs_.getInt(kSlotKey, 0);
  const bool stored = prefs_.getBool(kStoredKey, false);
  if (uid.length() == 0) {
    return false;
  }
  if (uid.length() >= static_cast<int>(sizeof(out.uid)) || !isValidSlot(slot)) {
    ATT_LOG_W("enroll", "discarding invalid enrollment recovery record");
    clear();
    return false;
  }
  strncpy(out.uid, uid.c_str(), sizeof(out.uid) - 1);
  out.slot = slot;
  out.templateStored = stored;
  out.present = true;
  return true;
}

bool EnrollmentStore::save(const char* uid, int slot) {
  if (!ready_ || uid == nullptr || !isValidSlot(slot) || strlen(uid) >= 80) {
    return false;
  }
  clear();
  prefs_.putInt(kSlotKey, slot);
  prefs_.putBool(kStoredKey, false);
  return prefs_.putString(kUidKey, uid) > 0;
}

bool EnrollmentStore::markTemplateStored() {
  if (!ready_) {
    return false;
  }
  return prefs_.putBool(kStoredKey, true) > 0;
}

bool EnrollmentStore::clear() {
  if (!ready_) {
    return false;
  }
  prefs_.remove(kUidKey);
  prefs_.remove(kSlotKey);
  prefs_.remove(kStoredKey);
  return true;
}

}  // namespace attendance
