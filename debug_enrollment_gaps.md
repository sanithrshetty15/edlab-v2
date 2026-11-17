# Enrollment Flow Debug Gaps

## User Report
"I am not receiving clear confirmation that the enrollment command has reached the ESP32"

## Complete Request Path (as requested)
Admin registration button → React API request → Go handler → PostgreSQL enrollment session → ESP32 command polling → command claim → sensor enrollment → progress updates → sensor template storage → backend completion → admin UI success

## Flow Tracing

### Step 1: Frontend starts enrollment
**File**: `MemberEnrollmentDialog.jsx` line 348-368
```javascript
const handleStart = async () => {
    if (busy || cancelling) return;
    if (!selectedReaderKey) { ... }
    setBusy("start");
    setError(null);
    try {
      const result = await membersManagementApi.startFingerprintEnrollment(member.id, {
        device_registration_id: readersInfo?.device?.id,
        reader_key: selectedReaderKey,
      });
      applyResult(result);
    } catch (startError) {
      setError(startError);
      loadReaders();
    } finally {
      setBusy(null);
    }
  };
```

**API call**: `membersManagementApi.startFingerprintEnrollment(id, payload)` → `POST /api/v1/members-management/:id/fingerprint-enrollment`

**Payload**: `{ device_registration_id: number, reader_key: string }`

**Backend** (`members_management.go:293-509` - `startMemberFingerprintEnrollment`):
- Validates member is `pending_enrollment`
- Validates reader availability via `enrollmentReaderAvailable()`
- Creates/updates `MemberEnrollmentSession`:
  - If session exists and is `pending_details`: transitions to `awaiting_sensor`, increments attempts, updates `device_registration_id`, `reader_key`, `expires_at`, `started_at`
  - If no existing session: creates new session with `session_uid` (hex 16 bytes), `status: awaiting_sensor`, `is_active: true`, `attempts: 1`
- Sets `expires_at = now + 10min` (enrollmentSessionTTL)
- Returns: `{ success: true, message: "...", member, enrollment: { session_uid, member_id, device_registration_id, reader_key, status: "awaiting_sensor", ... } }`

### Step 2: Frontend polls enrollment status
**File**: `MemberEnrollmentDialog.jsx` line 296-341
```javascript
useEffect(() => {
    if (!isActive || success || cancelling) return undefined;
    let stopped = false;
    let timer = null;
    let controller = null;
    const poll = async () => {
      controller = new AbortController();
      try {
        const result = await membersManagementApi.getFingerprintEnrollment(member.id, {
          signal: controller.signal,
        });
        if (stopped) return;
        applyResult(result);
        setError(null);
        if (result?.enrollment?.status !== "finalized") schedule();
      } catch (pollError) {
        if (stopped || pollError?.code === "aborted") return;
        setError(pollError);
        schedule();
      } finally {
        if (pollAbortRef.current === controller) pollAbortRef.current = null;
      }
    };
    poll();
    return () => {
      stopped = true;
      window.clearTimeout(timer);
      controller?.abort();
    };
  }, [applyResult, cancelling, isActive, member?.id, success]);
```

**API call**: `membersManagementApi.getFingerprintEnrollment(id, options)` → `GET /api/v1/members-management/:id/fingerprint-enrollment`

### Step 3: ESP32 fetches enrollment command
**File**: `device_auth.cpp:399-415` - `fetchEnrollmentCommand()`
```cpp
bool fetchEnrollmentCommand(EnrollmentSessionInfo& out, String& action, int& httpCode) {
  out = EnrollmentSessionInfo();
  action = "";
  String response;
  if (!performDeviceRequest("GET", "/api/v1/biometric/enrollment-session", "", response,
                            httpCode, ENROLLMENT_HTTP_TIMEOUT_MS)) {
    return false;
  }
  JsonDocument doc;
  if (!parseEnrollmentResponse(response, httpCode, doc, out, &action)) {
    return false;
  }
  if (httpCode != HTTP_CODE_OK || !doc["success"].as<bool>()) {
    return false;
  }
  return true;
}
```

**Requires**:
1. WiFi connected ✓
2. Device authenticated ✓ (checked in `loop()` before calling `processEnrollmentCommand()`)
3. HTTP GET to `/api/v1/biometric/enrollment-session` succeeds

**Go handler** (`attendance_biometric_tasks.go:731-817` - `claimEnrollmentSession`):
- Gets device via `currentDevice(c)` (requires `RequireDevice()`)
- Looks for cleanup session: `device_registration_id = ? AND cleanup_required = true AND reader_key IS NOT NULL`
- If not found, looks for active enrollment: `device_registration_id = ? AND is_active = true AND reader_key IS NOT NULL`, ordered by `id ASC`
- If found and not expired: returns `action: "enroll"` with session info
- If not found: returns `action: "none"`

**Response format**:
```json
{
  "success": true,
  "action": "enroll",
  "enrollment": {
    "session_uid": "...",
    "status": "awaiting_sensor",
    "member_name": "...",
    "member_login_id": "...",
    "error_code": null,
    "error_message": null,
    "member_id": 123,
    "attempts": 1,
    "cleanup_required": false,
    "is_active": true,
    "expires_at": "...",
    "started_at": "...",
    "completed_at": null,
    "updated_at": "...",
    "device": { ... }
  },
  "server_time": "2026-..."
}
```

**Firmware parsing** (`device_auth.cpp:285-306` - `parseEnrollmentSession`):
```cpp
bool parseEnrollmentSession(JsonVariantConst variant, EnrollmentSessionInfo& out) {
  out = EnrollmentSessionInfo();
  if (variant.isNull()) return;
  JsonObjectConst item = variant.as<JsonObjectConst>();
  out.sessionUid = item["session_uid"].as<String>();
  out.status = item["status"].as<String>();
  out.memberName = item["member_name"].as<String>();
  out.memberLoginId = item["member_login_id"].as<String>();
  out.errorCode = item["error_code"].as<String>();
  out.errorMessage = item["error_message"].as<String>();
  out.memberId = item["member_id"].as<int64_t>();
  out.attempts = item["attempts"].as<int>();
  out.cleanupRequired = item["cleanup_required"].as<bool>();
  out.active = item["is_active"].as<bool>();
  JsonVariantConst slot = item["fingerprint_slot"];
  if (!slot.isNull()) { out.slot = slot.as<int>(); }
  out.valid = out.sessionUid.length() > 0;
}
```

**EnrollmentSessionInfo fields** (from `device_auth.cpp`):
- `sessionUid` - matches `session_uid` from backend ✓
- `status` - matches `status` from backend ✓
- `memberName` - matches `member_name` from backend ✓
- `memberLoginId` - matches `member_login_id` from backend ✓
- `errorCode` - matches `error_code` from backend ✓
- `errorMessage` - matches `error_message` from backend ✓
- `memberId` - matches `member_id` from backend ✓
- `attempts` - matches `attempts` from backend ✓
- `cleanupRequired` - matches `cleanup_required` from backend ✓
- `active` - matches `is_active` from backend ✓
- `slot` - matches `fingerprint_slot` from backend ✓

### Step 4: `startActiveEnrollment(info)` 
**File**: `main.cpp:1104-1137`
```cpp
void startActiveEnrollment(const EnrollmentSessionInfo& info) {
  activeEnrollment = ActiveEnrollment();
  activeEnrollment.active = true;
  activeEnrollment.sessionUid = info.sessionUid;
  activeEnrollment.memberId = info.memberId;
  activeEnrollment.memberName = info.memberName;
  activeEnrollment.lastScanAt = millis() - ENROLLMENT_PREFLIGHT_INTERVAL;
  activeEnrollment.lastStatusCheck = millis();
  activeEnrollment.stage = enrollment_flow::Stage::WaitingFirstScan;
  currentMode = SensorMode::ENROLLMENT;
  setLEDStatus(true, false, false);
  setReaderReadiness(ReaderReadiness::ENROLLING);
  
  Serial.println("Sensor ownership: ENROLLMENT (login scanning suspended)");
  // ...
  
  if (info.slot >= 1 && info.slot <= 127) {
    bool exists = false;
    if (enrollmentReader->templateExists(static_cast<uint16_t>(info.slot), exists) && exists) {
      activeEnrollment.slot = static_cast<uint16_t>(info.slot);
      activeEnrollment.slotStored = true;
      activeEnrollment.stage = enrollment_flow::Stage::Finalizing;
      activeEnrollment.readerPreflightDone = true;
      // ...
    }
  }
}
```

**Sets**:
- `activeEnrollment.active = true`
- `activeEnrollment.sessionUid = info.sessionUid`
- `activeEnrollment.memberId = info.memberId`
- `activeEnrollment.stage = WaitingFirstScan`
- `currentMode = ENROLLMENT`

### Step 5: 19-stage state machine
**File**: `enrollment_flow.h` - Stages: `Idle → WaitingFirstScan → FirstScanCaptured → WaitingFingerRemoval → WaitingSecondScan → SecondScanCaptured → Finalizing → Complete/Failed`

**File**: `main.cpp:885-1102` - `processActiveEnrollment()` handles each stage:
- `WaitingFirstScan`: capture first image
- `FirstScanCaptured`: report progress, wait for finger removal
- `WaitingFingerRemoval`: detect finger removal
- `WaitingSecondScan`: capture second image
- `SecondScanCaptured`: match samples, create model, find free slot, store template
- `Finalizing`: attempt backend completion

### Step 6: Backend completion
**File**: `members_management.go:1062-1254` - `completeEnrollmentSession`:
- Validates session is active, not expired, slot matches, reader key set
- Creates `BiometricEnrollment` record
- Updates member status to `active`
- Updates session status to `finalized`
- Sends confirmation email

## Critical Gap Analysis

### Gap 1: Device Authentication
**Symptom**: `fetchEnrollmentCommand()` returns false, ESP32 never gets enrollment command
**Check**: Is `isDeviceAuthenticated()` returning true in `loop()`?
- `isDeviceAuthenticated()`: `g_deviceToken.length() > 0 && g_deviceId > 0`
- `authenticateDevice()`: `POST /api/v1/biometric/authenticate-device` with `api_key` and `device_name`
- Config has `DEVICE_API_KEY` and `DEVICE_NAME` set in `config.local.h`
- `lastAuthAttempt` retry: `AUTH_RETRY_INTERVAL = 300000ms` (5 min)

**Potential issue**: If `authenticateDevice()` hasn't been called yet, or the token expired, the ESP32 can't fetch enrollment commands.

### Gap 2: Session Not Owned by Device
**Symptom**: `claimEnrollmentSession()` returns `action: "none"`, ESP32 keeps polling but never starts enrollment
**Check**: Does the `MemberEnrollmentSession` have the correct `device_registration_id` and `reader_key`?
- `startMemberFingerprintEnrollment` sets `device_registration_id` from the request payload
- Sets `reader_key` from the request (must be "login" per validation)
- `claimEnrollmentSession` looks for: `device_registration_id = ? AND is_active = true AND reader_key IS NOT NULL`
- If the session was created with a different `device_registration_id`, or `reader_key` is null, the session won't be found

**Potential issue**: The frontend passes `device_registration_id: readersInfo?.device?.id` - need to verify this matches the ESP32's actual device registration ID.

### Gap 3: `processEnrollmentCommand()` Timing
**Symptom**: Enrollment command fetched but `startActiveEnrollment()` not called, or enrollment doesn't progress
**Check**: 
- `processEnrollmentCommand()` line 1284-1296:
  ```cpp
  enrollmentPollInterval = action == "none" ? ENROLLMENT_IDLE_POLL_INTERVAL : ENROLLMENT_POLL_INTERVAL;
  if (action == "cleanup") { ... }
  if (action == "enroll" && info.active) {
    startActiveEnrollment(info);
  }
  ```
- Requires: `action == "enroll" && info.active`
- `info.active` comes from `item["is_active"].as<bool()>` in `parseEnrollmentSession`
- If `is_active` is false or `sessionUid` is empty, `startActiveEnrollment` won't be called

### Gap 4: Enrollment Poll Interval
**Symptom**: Polling too fast/slow, commands missed or stale
**Check**:
- `enrollmentPollInterval` starts at `ENROLLMENT_POLL_INTERVAL = 1000ms`
- When `action == "none"`: back off to `ENROLLMENT_IDLE_POLL_INTERVAL = 2000ms`
- When `action == "enroll"`: stay at `ENROLLMENT_POLL_INTERVAL = 1000ms`
- But `processEnrollmentCommand()` is gated by `millis() - lastEnrollmentPoll >= enrollmentPollInterval`
- And `sensorActivity != SensorActivity::NONE` prevents polling during active enrollment

## Next Steps - Verification Plan

Based on the user's instruction: "Do not assume that the firmware is responsible until you have verified the frontend request and backend session creation"

1. **Verify frontend request**: Does `startFingerprintEnrollment` send the correct payload with valid `device_registration_id` and `reader_key`?
2. **Verify backend session creation**: Does `startMemberFingerprintEnrollment` create a valid `MemberEnrollmentSession` with proper `device_registration_id`, `reader_key`, `session_uid`, and `status: awaiting_sensor`?
3. **Verify ESP32 can authenticate**: Does `authenticateDevice()` succeed? Is `g_deviceToken` and `g_deviceId` set?
4. **Verify ESP32 can fetch command**: Does `fetchEnrollmentCommand()` succeed? What does the HTTP response look like?
5. **Verify claimEnrollmentSession returns enroll action**: Does the Go handler find the session and return `action: "enroll"`?

Let me start by verifying the frontend and backend flow, then move to the ESP32.