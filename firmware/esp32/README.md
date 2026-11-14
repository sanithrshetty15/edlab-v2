# Edwins Lab ESP32 Firmware

Firmware for the Edwins Lab biometric attendance and door access controller:
ESP32-WROOM-32UE (ALF V1.0), one R307S entry reader, a fail-safe
electromagnetic lock, a push-button exit, status LEDs and a buzzer.

The controller talks to the Go backend in `backend/` for device
authentication, member enrolment, synchronous door authorization, and
attendance events.

## Build support

PlatformIO is the supported build. The firmware has exactly one entry point:

```
firmware/esp32/
├── platformio.ini
├── src/
│   ├── main.cpp            # ONLY setup() and loop()
│   ├── R307S.h/.cpp        # R307S UART driver (single entry reader)
│   ├── enrollment_flow.h   # Pure two-scan enrollment state machine
│   ├── wifi_manager.h/.cpp # Non-blocking Wi-Fi management
│   ├── device_auth.h/.cpp  # Backend device auth + biometric events
│   ├── config.h            # Single configuration source
│   └── config.local.example.h
├── test/
│   └── test_enrollment_flow/test_main.cpp
├── legacy/                 # Archived sketches, not compiled
│   ├── CompleteFirmware.ino.legacy
│   └── firmware_main.cpp.legacy
└── README.md
```

`CompleteFirmware.ino` and `firmware_main.cpp` were duplicate implementations.
They are archived under `legacy/` so PlatformIO does not compile them.

### PlatformIO

```bash
cd firmware/esp32
pio run            # build
pio run -t upload  # flash
pio device monitor # serial monitor at 115200
pio test -e native # state-machine tests, no hardware required
```

- Board: `esp32dev` (ESP32-WROOM-32UE)
- Framework: Arduino
- Dependencies: `bblanchon/ArduinoJson`, `adafruit/Adafruit Fingerprint Sensor Library`
- The firmware uses its own dependency-free `R307S` driver; the Adafruit
  library is included as the reference driver dependency.

### Arduino IDE

The project is not structured as an Arduino sketch. Use PlatformIO. If you
need the Arduino IDE, open `src/main.cpp` in a sketch folder that also
contains the other `src/` files, but PlatformIO is the tested path.

## Configuration

`src/config.h` is the single source of configuration. Copy the example and
fill in real credentials:

```bash
cp src/config.local.example.h src/config.local.h
```

`src/config.local.h` is git-ignored. Never commit Wi-Fi passwords, the device
API key, or the HMAC secret. `config.h` includes `config.local.h` when present
and otherwise falls back to placeholder values.

```c
#define WIFI_SSID "..."
#define WIFI_PASSWORD "..."
#define API_BASE_URL "https://edwinslab.onrender.com"
#define DEVICE_API_KEY "..."
#define DEVICE_NAME "Edwins Lab ESP32 (ALF V1.0)"
```

`DEVICE_HMAC_SECRET` is optional. Set it only when event HMAC signing is
enabled; it must match the backend `BIOMETRIC_API_SECRET`.

The single entry reader is also the enrollment reader by default. Enrollment
uses the same sensor the member will physically scan at the door:

```c
#define ENROLLMENT_READER ENROLLMENT_READER_LOGIN
```

Set `ENROLLMENT_READER_NONE` only on a controller that must never enroll
members. The legacy `ENROLLMENT_READER_LOGOUT` value is retained for source
compatibility but now also maps to the single entry reader; the ALF V1.0 board
has no second reader in this deployment. The entry reader must be detected at
boot or enrollment stays disabled.

## Provisioning the device API key

The backend generates `DEVICE_API_KEY`; it is **not** set manually in
`config.h`. An administrator provisions the device once and copies the
one-time key into `config.local.h`.

1. Sign in as an administrator and capture the session token:

   ```bash
   curl -X POST https://<host>/api/v1/auth/admin-login \
     -H 'Content-Type: application/json' \
     -d '{"email":"admin@example.com","password":"..."}'
   ```

2. Provision the ESP32 as an admin:

   ```bash
   curl -X POST https://<host>/api/v1/devices \
     -H "Authorization: Bearer $ADMIN_TOKEN" \
     -H 'Content-Type: application/json' \
     -d '{"name":"Edwins Lab ESP32 (ALF V1.0)","device_type":"esp32"}'
   ```

   The `201` response contains `device.id`, a unique `device.device_uid`
   (`esp32-<16 hex>` unless supplied), and a one-time `api_key` (64 lowercase
   hex characters). The response is marked `Cache-Control: no-store`.

3. Put the returned key in `src/config.local.h`:

   ```c
   #define DEVICE_API_KEY "<api_key from the provisioning response>"
   #define DEVICE_NAME "Edwins Lab ESP32 (ALF V1.0)"
   ```

4. Build and flash:

   ```bash
   pio run -t upload
   pio device monitor
   ```

On boot the firmware calls
`POST /api/v1/biometric/authenticate-device` with
`{"api_key": DEVICE_API_KEY, "device_name": DEVICE_NAME}`. The response
contains a one-hour `X-Device-Token`, which is used for enrollment, status,
event submission and sync. The firmware re-authenticates automatically after a
`401`/`403`.

### Key rotation and revocation

```bash
# Issue a new key; old key and old tokens stop working immediately.
curl -X POST https://<host>/api/v1/devices/<device_id>/rotate-key \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Revoke the device permanently.
curl -X POST https://<host>/api/v1/devices/<device_id>/revoke \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

After rotation, replace `DEVICE_API_KEY` in `config.local.h` and reflash.
After revocation, the device receives `401`/`403` until it is replaced by a
newly provisioned device.

Never commit `config.local.h`. `DEVICE_HMAC_SECRET` is a separate shared
secret and is not replaced by per-device key rotation.

## Pin assignments (ALF V1.0)

| Signal | GPIO | Notes |
|--------|------|-------|
| `LOGIN_RX` / `LOGIN_TX` | 16 / 17 | R307S entry reader, UART2 |
| `RELAY_PIN` | 25 | Electromagnetic lock |
| `EXIT_BUTTON` | 33 | `INPUT_PULLUP`, pressed = LOW |
| `GREEN_LED` | 23 | Status LED |
| `RED_LED` | 22 | Status LED |
| `BUZZER_PIN` | 21 | Active-HIGH |

The legacy logout-reader pins are not used by this deployment. Enrollment and
entry share the single entry reader.

Relay levels are configurable in `config.h`:

```c
#define RELAY_LOCKED_LEVEL HIGH   // energised lock
#define RELAY_UNLOCK_LEVEL LOW    // de-energised, fail-safe release
```

Verify these against your relay module; an inverted module needs the levels
swapped. `README` and the archived sketch previously disagreed, so measure the
module before deploying.

## Fingerprint enrollment workflow

An administrator creates the member's details first, then starts a
fingerprint enrollment session from member management. The device polls
`GET /api/v1/biometric/enrollment-session` every two seconds and only claims a
session when an enrollment reader is configured and ready.

1. The backend returns an `enroll` action and a `session_uid`.
2. The firmware reports `waiting_first_scan`, captures the first image, and
   reports `first_scan_captured`.
3. The finger must be removed; the firmware then reports
   `waiting_for_removal` and `waiting_second_scan`.
4. The second image is captured and reported as `second_scan_captured`.
5. The driver compares both character buffers, merges them with `regModel`,
   finds a free slot by probing the sensor, stores the template, and searches
   every other slot for a duplicate.
6. The slot is reported with `verifying` and the firmware calls
   `POST /api/v1/biometric/enrollment-sessions/:uid/complete`.
7. The member becomes active only after the backend finalizes the enrollment.

The firmware never uploads or stores raw fingerprint images or templates. It
keeps only the slot number and session UID in NVS while a stored template is
waiting for backend confirmation. On reboot, that state is reconciled with the
backend before any new session is claimed. Failed, cancelled, or expired
sessions delete the locally stored template; a backend `slot_conflict` response
is the one case where the local template is left untouched because the backend
has told the device not to erase another member's slot.

The firmware handles duplicate fingers, two scans that do not match, sensor
errors, no free slot, NVS failures, backend conflicts, cancellation, expiry,
and network loss during completion. In the last case it retries completion and
keeps the NVS recovery record until the session is resolved.

## Backend contract

All routes are under `https://<host>/api/v1`. Device routes authenticate with
`X-Device-Token`; the token is obtained with the provisioned API key.

| Endpoint | Method | Purpose | Auth |
|---|---|---|---|
| `/api/v1/biometric/authenticate-device` | POST | Exchange `{api_key, device_name}` for a token | none |
| `/api/v1/biometric/enrollment-session` | GET | Claim cleanup or active enrollment work | `X-Device-Token` |
| `/api/v1/biometric/enrollment-sessions/:uid` | GET | Check status, cancellation, or expiry | `X-Device-Token` |
| `/api/v1/biometric/enrollment-sessions/:uid/progress` | POST | Report two-scan progress | `X-Device-Token` |
| `/api/v1/biometric/enrollment-sessions/:uid/complete` | POST | Finalize the slot and activate the member | `X-Device-Token` |
| `/api/v1/biometric/enrollment-sessions/:uid/fail` | POST | Report a sensor or match failure | `X-Device-Token` |
| `/api/v1/biometric/enrollment-sessions/:uid/cleanup` | POST | Acknowledge local slot cleanup | `X-Device-Token` |
| `/api/v1/biometric/enrollments` | GET | Cache slot-to-member mappings | `X-Device-Token` |
| `/api/v1/biometric/submit-event` | POST | Submit `fingerprint_scan` events | `X-Device-Token` or HMAC signature |
| `/api/v1/biometric/status` | GET | Device status / pending event count | `X-Device-Token` |
| `/api/v1/biometric/sync-events` | POST | Fetch events for the device | `X-Device-Token` |

The legacy `POST /api/v1/biometric/enroll-fingerprint` route remains in the
backend for compatibility, but the current firmware uses the session workflow
above.

There is no `/api/v1/device/authenticate`, `/api/v1/device/heartbeat`,
`/api/v1/biometric/event`, `/api/v1/biometric/enroll` or
`/api/v1/biometric/sync` route in the Go backend. The firmware uses the actual
routes above.

Authentication response:

```json
{
  "success": true,
  "device": {
    "id": 1,
    "device_uid": "esp32-0123456789abcdef",
    "name": "Edwins Lab ESP32",
    "device_type": "esp32",
    "status": "active",
    "token": "<payload>.<signature>",
    "expires_in": 3600
  }
}
```

Allowed event types on `/submit-event`: `fingerprint_scan`,
`finger_mismatch`, `enrollment_success`, `enrollment_failure`.

## Safety behaviour

- **A local fingerprint match never unlocks the door on its own.** The firmware
  calls `POST /api/v1/biometric/door-access` and unlocks only when the backend
  synchronously returns `authorized: true` for an active member and the slot
  belongs to the entry reader. Backend errors and timeouts keep the door locked.
- **Duplicate scans are acknowledged but do not re-unlock.** The backend
  suppresses repeated grants for the same member within three seconds.
- **Exit is local and immediate.** The GPIO33 button de-energises the relay
  without Wi-Fi or backend access and the normal `DOOR_OPEN_MS` timer re-locks
  the door automatically.
- **Exit never creates a member logout event.** The button does not call the
  backend and does not submit a biometric event.
- Attendance is preserved through the backend: an authorized door-access request
  creates a `fingerprint_scan` device event and the existing
  `biometric.process_event` job records the login/logout.

## Verification status

- `pio run`: **passes** (ESP32 image built).
- `pio test -e native`: **passes** (5 two-scan state-machine tests).
- Backend contract inspected from `backend/internal/api/member_enrollment.go`,
  `backend/internal/api/biometric.go` and
  `backend/internal/api/attendance_biometric_tasks.go`.
- **No physical hardware testing has been performed.** R307S readers, the
  relay, the exit button, LEDs and buzzer are untested on real hardware.
- Backend integration was compiled against the real routes but not exercised
  against a running backend from the device.

## Troubleshooting

1. **Wi-Fi never connects** — check `config.local.h`; the firmware retries
   every `WIFI_RETRY_INTERVAL` without blocking the loop.
2. **`Device auth HTTP 401`** — `DEVICE_API_KEY` does not match a provisioned,
   active device in the backend.
3. **Enrollment never starts** — confirm `ENROLLMENT_READER` is
   `ENROLLMENT_READER_LOGIN` (the default), the entry reader was detected at
   boot, and the member is still `pending_enrollment`.
4. **Entry denied, fingerprint not enrolled** — the finger matches on the
   sensor but the slot is not registered to an active member on this device, so
   the backend keeps the door locked by design.
5. **No fingerprint detected** — verify the entry reader UART pins and that the
   reader is powered; sensors run at 57600 baud.
6. **Door never unlocks** — confirm the relay active levels and that the device
   is authenticated and enrolled.
7. **Enrollment stuck after a reboot** — the firmware reconciles the NVS
   session UID and slot before claiming new work. Check the serial log for
   `Pending enrollment recovery` and sensor cleanup errors.
8. **Flash usage** — the build uses about 90% of the 1.3 MB app partition;
   keep an eye on it when adding libraries.
