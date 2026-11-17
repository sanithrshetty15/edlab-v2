# Fingerprint Enrollment Fix

Status: code and automated tests complete. Physical ESP32 + R307S validation is still required.

## Reported Symptom

The admin could start an enrollment, but the UI did not confirm that the command had reached the ESP32. The session stayed at `awaiting_sensor`, so a queued command and a command that the controller had already received looked identical.

## Root Causes Found

The exact hardware failure cannot be proven without a serial log from the installed controller, but the following code-level defects explain the symptom and the observed failure modes.

### 1. No persisted command acknowledgement (primary cause of the reported symptom)

The device poll endpoint returned the same active session every time. There was no state transition or timestamp that meant "the ESP32 received this command". The admin UI therefore could not distinguish:

- command written to PostgreSQL;
- command returned by the poll endpoint;
- command received and started by the ESP32.

Fix: `POST /api/v1/biometric/enrollment-sessions/:uid/claim` persists `claimed_at` and `claimed_by_device_id`. The firmware calls it before touching the sensor, and the admin dialog displays the claim time.

### 2. Reader availability ignored readiness

The backend offered a reader based on `connected` and freshness only. A reader that was still initializing, or had just failed a capture, could be selected even though the firmware contract says it must only start when readiness is `ready` or `waiting_for_finger`.

Fix: reader availability now requires a fresh, connected reader with an enrollment-capable readiness value.

### 3. `connected` was serialized as `available`

`serializeEnrollmentReader` returned `connected: available`. A disconnected but fresh reader could therefore look connected, and the UI could not explain why it was unavailable.

Fix: `connected` now comes from the stored reader record.

### 4. Offline devices could be selected

Device selection did not check heartbeat freshness before creating a session.

Fix: start returns HTTP 409 with `code: "device_offline"` when the selected device has no recent heartbeat.

### 5. Stale cleanup sessions could block enrollment forever

The cleanup lookup did not exclude sessions whose cleanup had already completed. It also had no recovery path when the sensor slot deletion or acknowledgement repeatedly failed. A single stale `cleanup_required` row could therefore block every later enrollment on that device.

Fix: cleanup lookup requires `cleanup_completed_at IS NULL`, and cleanup older than 24 hours is abandoned without deleting a possibly reused sensor slot. Active cleanup is still retried and acknowledged normally.

### 6. Firmware accepted invalid backend JSON

`parseEnrollmentResponse` returned `true` after a JSON deserialization failure. Callers could act on an empty or partially filled `EnrollmentSessionInfo`.

Fix: deserialization failure now returns `false`.

### 7. Firmware did not model the required sensor sequence

The previous stage machine did not explicitly separate sensor initialization, image capture, image conversion, model creation and template storage. This made it possible to report progress that did not match what the R307S had actually done, and left cleanup behavior dependent on ad-hoc flags.

Fix: the firmware now has the full 16-stage state machine and calls the sensor in the required order:

```text
getImage -> image2Tz(1) -> finger removal -> getImage -> image2Tz(2)
-> createModel -> storeModel(slot)
```

### 8. Cleanup acknowledgement bugs

The firmware skipped the cleanup acknowledgement when the backend did not allocate a slot, and ignored the acknowledgement result. The backend could therefore keep returning cleanup forever.

Fix: the firmware always acknowledges cleanup, checks the HTTP result, and retries when it fails. It keeps exclusive sensor ownership until deletion and acknowledgement succeed, so login scanning cannot consume the sensor mid-cleanup.

### 9. Command delivery could stall behind token refresh

Device authentication was retried only every 5 minutes while idle. If the token expired just before an enrollment was started, the device could appear not to receive the command for several minutes.

Fix: idle retry is now 60 seconds; an active enrollment retries every 5 seconds.

### 10. Diagnostics were too coarse

Firmware serial output did not identify the active stage, claim failures, or most cleanup retries. The admin UI did not show claim state.

Fix: the firmware logs every stage transition, claim HTTP failures and cleanup retries. The dialog shows command acknowledgement time and polls every second while active.

## Reference Comparison

Six reference projects were extracted under `/tmp/opencode/refs/` and reviewed. They were used only for guidance.

Useful patterns adopted:

- server-created enrollment session with an expiry window;
- device poll loop with `enroll` / `cleanup` / `none` actions;
- explicit reader readiness reporting;
- explicit claim/acknowledgement before hardware work;
- cleanup acknowledgement and retry.

Patterns deliberately not copied:

- reference serial protocols and sensor command wrappers;
- reference route names or authentication schemes;
- reference database schema changes;
- direct upload of fingerprint templates to the backend;
- any behavior that unlocks the door from a local fingerprint match alone.

The implementation keeps the existing EdwinsLab architecture: backend owns authorization and slot allocation, while the ESP32 owns the sensor and never sends raw templates.

## Corrected Flow

### Happy path

1. Admin selects an available reader and starts enrollment.
2. Backend validates: member is pending, device is online, reader is connected/fresh/ready, reader key is valid.
3. Backend creates or restarts a session as `awaiting_sensor`, clears any previous claim, and sets a 10 minute expiry.
4. ESP32 polls `GET /api/v1/biometric/enrollment-session` and receives action `enroll`.
5. Firmware enters `ClaimingSession` and calls `POST /api/v1/biometric/enrollment-sessions/:uid/claim`.
6. Backend persists `claimed_at` and `claimed_by_device_id`.
7. Firmware enters `SensorInitialization`, reports progress `sensor_initializing`, and probes the R307S with `getImage`.
8. No finger: stage `WaitingForFirstFinger`, progress `waiting_first_scan`.
9. First finger: stage `CapturingFirstImage`, then `ConvertingFirstImage`; firmware calls `image2Tz(1)` and reports `first_scan_captured`.
10. Stage `WaitingForFingerRemoval`; progress `waiting_for_removal`.
11. Finger removed: stage `WaitingForSecondFinger`; progress `waiting_second_scan`.
12. Second finger: `CapturingSecondImage`, then `ConvertingSecondImage`; firmware calls `image2Tz(2)` and reports `second_scan_captured`.
13. Stage `CreatingModel`: firmware calls `match()` and requires a score of at least 50, then `createModel()`.
14. Stage `StoringTemplate`: firmware finds a free slot, persists recovery state, calls `storeModel(slot)`, then runs the duplicate check.
15. Firmware reports `verifying` and enters `BackendFinalization`.
16. Firmware calls `POST /complete` with the slot.
17. Backend verifies the session, allocates the slot mapping, activates the member, and marks the session `finalized`.
18. Firmware clears recovery state and reports success. Admin dialog shows the completed member.

### Failure and cleanup path

1. A sensor, match, storage, duplicate or backend error calls the failure endpoint with an error code.
2. If a template may exist, the backend marks the session `cleanup_required`.
3. Firmware deletes the local sensor slot and acknowledges cleanup.
4. On a backend slot conflict, firmware deletes the template it just stored before stopping; otherwise that template could later authorize as the member that already owns the slot.
5. If deletion or acknowledgement fails, firmware keeps exclusive sensor ownership and retries; it does not resume login scanning.
6. The backend will not issue a new enrollment to that device until cleanup is acknowledged.
7. After 24 hours of an unacknowledged cleanup, the backend abandons the row to avoid a permanent wedge. The sensor slot may then need manual cleanup; the backend does not reuse a slot blindly.

### Reboot recovery

- Firmware persists the session UID and slot before storing the template.
- On boot, recovery checks the backend session and whether the local template exists.
- If the template exists and the session is still active, firmware resumes `BackendFinalization`.
- If the backend says finalized, firmware keeps the local template and clears recovery state.
- If the backend says cleanup, firmware deletes the local template and acknowledges cleanup.

## Firmware State Machine

| Stage | Responsibility | Progress status |
| --- | --- | --- |
| `Idle` | No enrollment | none |
| `ClaimingSession` | Acknowledge command to backend | none |
| `SensorInitialization` | Probe R307S with `getImage` | `sensor_initializing` |
| `WaitingForFirstFinger` | Wait for first finger | `waiting_first_scan` |
| `CapturingFirstImage` | Transition after image capture | `waiting_first_scan` |
| `ConvertingFirstImage` | `image2Tz(1)` | `first_scan_captured` |
| `WaitingForFingerRemoval` | Wait for finger removal | `waiting_for_removal` |
| `WaitingForSecondFinger` | Wait for second finger | `waiting_second_scan` |
| `CapturingSecondImage` | Transition after image capture | `waiting_second_scan` |
| `ConvertingSecondImage` | `image2Tz(2)` | `second_scan_captured` |
| `CreatingModel` | `match` then `createModel` | `verifying` |
| `StoringTemplate` | Free slot, persist, `storeModel`, duplicate check | `verifying` |
| `BackendFinalization` | POST completion and retry | `verifying` |
| `EnrollmentCleanup` | Delete local slot, acknowledge, retry | none |
| `Completed` | Terminal success | `finalized` |
| `Failed` | Terminal failure | `failed` |

## API Contract Changes

### New claim endpoint

```http
POST /api/v1/biometric/enrollment-sessions/:uid/claim
X-Device-Token: <device token>
```

Response:

```json
{
  "success": true,
  "message": "Fingerprint enrollment command acknowledged",
  "enrollment": {
    "session_uid": "...",
    "status": "awaiting_sensor",
    "claimed_at": "2026-11-17T11:41:11.520533Z",
    "claimed_by_device_id": 16
  }
}
```

The endpoint is idempotent for the device that owns the session. Another device, an inactive session, an expired session, or a cleanup-required session is rejected with HTTP 409.

### Start behavior

- Offline selected device: HTTP 409, `code: "device_offline"`.
- Unavailable reader: HTTP 409, `code: "reader_unavailable"`.
- Reader availability now includes readiness.

### Session serialization

`claimed_at` and `claimed_by_device_id` are returned by the admin enrollment status endpoint. Existing fields are unchanged.

## Files Changed

Backend:

- `backend/internal/database/migrations/0011_enrollment_session_claims.sql` (new)
- `backend/internal/models/models.go`
- `backend/internal/api/member_enrollment.go`
- `backend/internal/api/attendance_biometric_tasks.go`
- `backend/internal/api/device_readers.go`
- `backend/internal/api/devices.go`
- `backend/internal/api/member_enrollment_integration_test.go`

Firmware:

- `firmware/esp32/src/enrollment_flow.h`
- `firmware/esp32/src/main.cpp`
- `firmware/esp32/src/device_auth.cpp`
- `firmware/esp32/src/device_auth.h`
- `firmware/esp32/src/config.h`
- `firmware/esp32/test/test_enrollment_flow/test_main.cpp`

Frontend:

- `frontend/src/pages/app/admin/MemberEnrollmentDialog.jsx`
- `frontend/src/pages/app/admin/DevicesPage.jsx`

## Validation Performed

All commands were run from the repository root unless noted.

Backend:

```bash
cd backend
gofmt -w ./internal/api/member_enrollment.go \
  ./internal/api/device_readers.go \
  ./internal/api/member_enrollment_integration_test.go \
  ./internal/models/models.go
go build ./...
go vet ./...
TEST_DATABASE_URL='postgres://sanith@localhost/edwinslab_enrollment_test?sslmode=disable' \
  go test ./internal/api -count=3
TEST_DATABASE_URL='postgres://sanith@localhost/edwinslab_enrollment_test?sslmode=disable' \
  go test ./... -count=1
```

Result: pass.

Firmware:

```bash
cd firmware/esp32
pio test -e native
pio run -e esp32dev
```

Result: 11 native tests passed; ESP32 build succeeded. Flash usage is 1,227,040 bytes of 1,310,720 bytes (93.6%), leaving about 84 KB.

Frontend:

```bash
cd frontend
npm run build
```

Result: pass.

## Deployment and Upload

### Backend

The new migration is embedded in the backend binary and runs automatically on startup. After deploying, verify that `public.go_schema_migrations` contains version `0011` before testing enrollment.

### Frontend

Deploy the normal frontend build. The dialog now polls every second while a session is active and displays `claimed_at`.

### ESP32

Do not erase the ESP32 flash and do not reset the R307S template database.

```bash
cd firmware/esp32
pio run -e esp32dev
pio run -e esp32dev -t upload
pio device monitor -b 115200
```

If PlatformIO cannot select a port:

```bash
pio run -e esp32dev -t upload --upload-port /dev/ttyUSB0
pio device monitor --port /dev/ttyUSB0 -b 115200
```

Use the actual port shown by `pio device list`.

## Manual Hardware Test

Preconditions:

- backend deployed with migration `0011`;
- admin frontend deployed;
- ESP32 flashed with this firmware;
- device authenticated and reporting the login reader;
- a member in `pending_enrollment`, with no existing biometric mapping;
- sensor contains only the templates that should remain.

### 1. Boot sanity

Open the serial monitor and confirm:

- `Edwins Lab ESP32 controller starting`;
- `Entry reader: password verified; capture readiness pending`;
- `Enrollment reader: configured; capture readiness pending`;
- `Wi-Fi connected!`;
- `Attempting backend device authentication...`;
- `Reader readiness: waiting_for_finger` or `ready`.

If readiness stays `not_ready` or `capture_failed`, fix the reader before starting enrollment.

### 2. Delivery confirmation

1. Open the pending member's enrollment dialog.
2. Confirm the login reader is listed as available.
3. Start enrollment.
4. Within roughly 1-2 seconds the dialog should show `Command received` with a relative time.
5. The serial monitor should show in order:

```text
Enrollment poll action: enroll
Enrollment: command for session <uid>
Enrollment stage: claiming_session
Sensor ownership: ENROLLMENT (login scanning suspended)
Enrollment: command queued for <member>, session <uid>
Enrollment: acknowledging enrollment command
Enrollment stage: sensor_initialization
Enrollment: command received from backend; initializing reader
Enrollment stage: waiting_for_first_finger
Enrollment: reader initialized; place the finger for the first scan
```

If `Command received` never appears but the poll reports `enroll`, check device authentication and the claim endpoint logs.

### 3. Two-scan capture

1. Place the member's finger on the reader.
2. Confirm `first_scan_captured` appears and the dialog asks for finger removal.
3. Remove the finger.
4. Confirm the dialog asks for the same finger again.
5. Place the same finger again.
6. Confirm serial output:

```text
Enrollment stage: converting_first_image
Enrollment: first fingerprint captured; remove the finger
Enrollment stage: waiting_for_finger_removal
Enrollment: finger removed; place the same finger again
Enrollment stage: converting_second_image
Enrollment: second fingerprint captured; matching the two samples
Enrollment stage: creating_model
Enrollment stage: storing_template
Enrollment: storing template in sensor slot <n>
Enrollment: template stored in slot <n>
Enrollment: template stored; confirming registration with the backend
Enrollment: registration successful
```

### 4. Acceptance checks

- Admin dialog shows `Member activated`.
- Member status becomes active.
- `member_enrollment_sessions` row is `finalized`, has the correct slot, and `cleanup_required = false`.
- The sensor reports the template in the allocated slot.
- A second member cannot use the same fingerprint.
- Login with the new fingerprint succeeds only after backend authorization.
- An unauthorized or unknown slot still does not unlock the door.

### 5. Failure and cleanup checks

- Disconnect the R307S during enrollment. The firmware must report a sensor error, not continue as if capture succeeded. The member stays pending.
- Cancel an enrollment after a slot is allocated. The session must become `cleanup_required`; the firmware must delete the slot and acknowledge cleanup before login scanning resumes.
- Stop the backend mid-enrollment. The firmware must retry completion after the backend returns; it must not declare success locally.
- Restart the ESP32 after template storage but before completion. On boot it must resume backend finalization, not perform a third scan.

## Operational Notes

- `claimed_at` proves command delivery only. It does not prove the sensor will succeed.
- Reader reports are periodic (30 seconds). Wait for a fresh report after reconnecting a reader.
- Do not lower the enrollment session TTL below the time needed for two scans plus network retries.
- The 24 hour stale-cleanup abandonment is a safety valve, not normal operation. Investigate any occurrence; a sensor slot may still exist and require manual deletion.
- Firmware flash headroom is tight. Any further sensor/UI strings should be checked against the 4 MB partition limit.
