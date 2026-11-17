# JSON Field Mapping Debug

## Backend → Firmware JSON Field Mapping

### Go `claimEnrollmentSession` Response

**Endpoint**: `GET /api/v1/biometric/enrollment-session`
**Handler**: `claimEnrollmentSession()` in `attendance_biometric_tasks.go:731`

**Response format**:
```json
{
  "success": true,
  "action": "enroll",
  "enrollment": {
    "session_uid": "3a2b1c4d5e6f7g8h9i0j",  // 32 hex chars
    "status": "awaiting_sensor",
    "member_name": "John Doe",
    "member_login_id": "jdoe",
    "error_code": null,
    "error_message": null,
    "member_id": 123,
    "attempts": 1,
    "cleanup_required": false,
    "is_active": true,
    "expires_at": "2026-11-18T10:00:00Z",
    "started_at": "2026-11-17T10:00:00Z",
    "completed_at": null,
    "updated_at": "2026-11-17T10:05:00Z",
    "device": { ... }
  },
  "server_time": "2026-11-17T10:30:00Z"
}
```

### Firmware `parseEnrollmentResponse` Parsing

**File**: `device_auth.cpp:308-322`
```cpp
bool parseEnrollmentResponse(const String& response, int httpCode,
                             JsonDocument& doc, EnrollmentSessionInfo& out,
                             String* action = nullptr) {
  if (httpCode <= 0) { return false; }
  if (deserializeJson(doc, response)) { return true; }  // JSON parse failed
  if (action != nullptr) {
    *action = doc["action"].as<String>();  // extracts "enroll"
  }
  parseEnrollmentSession(doc["enrollment"], out);  // parses enrollment object
  return true;
}
```

### Firmware `parseEnrollmentSession` Parsing

**File**: `device_auth.cpp:285-306`
```cpp
bool parseEnrollmentSession(JsonVariantConst variant, EnrollmentSessionInfo& out) {
  out = EnrollmentSessionInfo();
  if (variant.isNull()) { return; }
  JsonObjectConst item = variant.as<JsonObjectConst>();
  out.sessionUid = item["session_uid"].as<String>();       // from "session_uid"
  out.status = item["status"].as<String>();                // from "status"
  out.memberName = item["member_name"].as<String>();       // from "member_name"
  out.memberLoginId = item["member_login_id"].as<String>(); // from "member_login_id"
  out.errorCode = item["error_code"].as<String>();         // from "error_code"
  out.errorMessage = item["error_message"].as<String>();   // from "error_message"
  out.memberId = item["member_id"].as<int64_t>();          // from "member_id"
  out.attempts = item["attempts"].as<int>();              // from "attempts"
  out.cleanupRequired = item["cleanup_required"].as<bool>();// from "cleanup_required"
  out.active = item["is_active"].as<bool>();              // from "is_active"
  JsonVariantConst slot = item["fingerprint_slot"];        // from "fingerprint_slot"
  if (!slot.isNull()) { out.slot = slot.as<int>(); }
  out.valid = out.sessionUid.length() > 0;
}
```

### EnrollmentSessionInfo Structure (firmware)

**File**: `device_auth.cpp` namespace (defined via `EnrollmentSessionInfo` struct)
- `sessionUid` - String, used as `activeEnrollment.sessionUid`
- `status` - String, used to check enrollment progress
- `memberName` - String, displayed in UI
- `memberLoginId` - String, displayed in UI
- `errorCode` - String, error code on failure
- `errorMessage` - String, error message on failure
- `memberId` - int64_t, member ID from backend
- `attempts` - int, number of enrollment attempts
- `cleanupRequired` - bool, whether sensor cleanup is needed
- `active` - bool, whether session is active
- `slot` - int, fingerprint slot number

### Critical Field Mapping Check

| Firmware Field | Go Response Field | Match? |
|---|---|---|
| `sessionUid` | `session_uid` | ✅ Exact match |
| `status` | `status` | ✅ Exact match |
| `memberName` | `member_name` | ✅ Exact match |
| `memberLoginId` | `member_login_id` | ✅ Exact match |
| `errorCode` | `error_code` | ✅ Exact match |
| `errorMessage` | `error_message` | ✅ Exact match |
| `memberId` | `member_id` | ✅ Exact match |
| `attempts` | `attempts` | ✅ Exact match |
| `cleanupRequired` | `cleanup_required` | ✅ Exact match |
| `active` | `is_active` | ✅ Exact match |
| `slot` | `fingerprint_slot` | ✅ Exact match |

**All field names match exactly** between the Go backend response and the C++ firmware parsing. No JSON key mismatches.

### Potential Issues

1. **`deserializeJson` failure**: If the JSON response from the backend is malformed or has unexpected types, `deserializeJson(doc, response)` returns true, and `parseEnrollmentResponse` returns true (indicating failure). The caller `fetchEnrollmentCommand` then returns false.

2. **`doc["action"]` null**: If the `action` field is missing from the response, `doc["action"].as<String>()` returns empty string `""`. Then `processEnrollmentCommand` checks `if (action == "enroll" && info.active)` - since `action` is `""`, this is false, and `startActiveEnrollment` is never called.

3. **`doc["success"]` false**: If the response has `"success": false`, `fetchEnrollmentCommand` returns false after the check `if (httpCode != HTTP_CODE_OK || !doc["success"].as<bool>())`.

4. **`info.sessionUid` empty**: If `item["session_uid"]` is null or missing, `out.sessionUid` becomes empty string. Then `startActiveEnrollment(info)` sets `activeEnrollment.sessionUid = ""`, and the enrollment state machine might behave unexpectedly.

5. **`info.memberId` is 0**: If `item["member_id"]` is null or missing, `out.memberId` becomes 0. This might cause issues later when verifying with the backend.

6. **`info.active` is false**: If `item["is_active"]` is false, then in `processEnrollmentCommand`: `if (action == "enroll" && info.active)` is false, and `startActiveEnrollment` is never called.

7. **`info.slot` is 0 or negative**: If `item["fingerprint_slot"]` is null or out of range (1-127), `out.slot` becomes 0 or unchanged. In `startActiveEnrollment`: `if (info.slot >= 1 && info.slot <= 127)` - if slot is 0 or >127, the preflight check passes and the stage stays at `WaitingFirstScan` instead of jumping to `Finalizing`.

All these field mappings are correct, so the issue is likely elsewhere in the flow - not in JSON parsing.