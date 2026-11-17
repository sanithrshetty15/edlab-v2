# Edwins Lab ESP32 Door Lock Firmware

Firmware for the Edwins Lab biometric door lock controller: ESP32-WROOM-32UE on
an ALF V1.0 board, one R307S fingerprint reader, a fail-safe electromagnetic
lock, a push-button exit, status LEDs and a buzzer.

The controller talks to the Go backend in `backend/` for device
authentication, fingerprint enrollment, synchronous door authorization, and
access logging.

## Single-file sketch

The entire firmware is one Arduino IDE sketch:

```
firmware/esp32/
├── platformio.ini
├── doorlock/
│   └── doorlock.ino      # entire firmware: config, R307S driver, Wi-Fi,
│                         # device API client, enrollment state machine,
│                         # relay, LEDs, buzzer, exit button
├── legacy/               # archived sketches, not compiled
└── README.md
```

`doorlock.ino` contains the Wi-Fi credentials block, the R307S UART driver, the
backend device client, the enrollment state machine, the door/indicator logic
and the exit-button handling. There are no other firmware source files. The
folder name (`doorlock`) matches the sketch name so the same file opens and
uploads directly in the Arduino IDE.

### Arduino IDE

1. Open `firmware/esp32/doorlock/doorlock.ino`.
2. Install the **ArduinoJson** library (Library Manager), version 7.x.
3. Select board **ESP32 Dev Module** (or your ALF V1.0 board definition), the
   correct port, upload speed `921600`.
4. Set the configuration at the top of the sketch (see below) and press
   Upload. Serial monitor: `115200`.

Everything else (`WiFi`, `HTTPClient`, `Preferences`, `HardwareSerial`) ships
with the ESP32 Arduino core.

### PlatformIO

```bash
cd firmware/esp32
pio run            # build
pio run -t upload  # flash
pio device monitor # serial monitor at 115200
```

`platformio.ini` sets `src_dir = doorlock`, so PlatformIO compiles the same
single sketch. Board: `esp32dev`, framework: Arduino, dependency:
`bblanchon/ArduinoJson`.

## Configuration

The configuration block is at the top of `doorlock.ino` and holds placeholders
only. Never commit real credentials.

```c
#define WIFI_SSID "CHANGE_ME_WIFI_SSID"
#define WIFI_PASSWORD "CHANGE_ME_WIFI_PASSWORD"
#define API_BASE_URL "https://edwinslab.onrender.com"
#define DEVICE_API_KEY "CHANGE_ME_DEVICE_API_KEY"
#define DEVICE_NAME "Edwins Lab ESP32 (ALF V1.0)"
```

To keep secrets out of the sketch file, create `doorlock_secrets.h` next to
`doorlock.ino` (it is git-ignored) and define any of the macros there; the
sketch includes it automatically when present:

```c
#define WIFI_SSID "lab-wifi"
#define WIFI_PASSWORD "..."
#define DEVICE_API_KEY "<api_key from the provisioning response>"
```

## Provisioning the device API key

The backend generates `DEVICE_API_KEY`; it is **not** set manually in the
backend. An administrator provisions the device once and copies the one-time
key into the sketch (or `doorlock_secrets.h`).

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

3. Put the returned key into the sketch configuration (`DEVICE_API_KEY`).

4. Build and flash:

   ```bash
   pio run -t upload
   pio device monitor
   ```

On boot the firmware calls
`POST /api/v1/biometric/authenticate-device` with
`{"api_key": DEVICE_API_KEY, "device_name": DEVICE_NAME}`. The response
contains a one-hour `X-Device-Token`, used for enrollment, status and door
authorization. The firmware re-authenticates automatically after a `401`/`403`.

After successful authentication the firmware reports the connected entry
reader with `POST /api/v1/biometric/readers`, then refreshes that report every
30 seconds (`READER_REPORT_INTERVAL`). This is required before an administrator
can start fingerprint enrollment.

### Key rotation and revocation

```bash
# Issue a new key; old key and old tokens stop working immediately.
curl -X POST https://<host>/api/v1/devices/<device_id>/rotate-key \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Revoke the device permanently.
curl -X POST https://<host>/api/v1/devices/<device_id>/revoke \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

After rotation, replace `DEVICE_API_KEY` and reflash. After revocation, the
device receives `401`/`403` until it is replaced by a newly provisioned device.

## Pin assignments (ALF V1.0)

| Signal | GPIO | Notes |
|--------|------|-------|
| `LOGIN_RX` / `LOGIN_TX` | 16 / 17 | R307S reader, UART2, 57600 baud |
| `RELAY_PIN` | 25 | Electromagnetic lock, active LOW unlock |
| `EXIT_BUTTON` | 33 | `INPUT_PULLUP`, pressed = LOW |
| `GREEN_LED` | 23 | Status LED |
| `RED_LED` | 22 | Status LED |
| `BUZZER_PIN` | 21 | Active-HIGH |

There is exactly one reader. It serves both door login and fingerprint
enrollment; a login scan is never performed while the sensor is owned by the
enrollment state machine.

Relay levels are defined in the hardware section of the sketch:

```c
const int RELAY_LOCKED_LEVEL = HIGH;   // energised lock
const int RELAY_UNLOCK_LEVEL = LOW;    // de-energised, fail-safe release
```

Verify these against your relay module; an inverted module needs the levels
swapped.

## Fingerprint storage architecture

- The R307S stores the fingerprint templates in its own flash, one template per
  slot, slots 1-127.
- The backend stores the member details and the slot-to-member mapping. No
  names, USNs, email addresses or other personal data are written to the
  ESP32 or the sensor.
- The ESP32 only learns a slot number from the sensor and sends that slot to
  the backend. The backend decides whether the door opens and writes the
  access log.

## Fingerprint enrollment workflow

An administrator creates the member's details first, then starts a fingerprint
enrollment session from the Door Lock System page. The device polls
`GET /api/v1/biometric/enrollment-session` and claims the session when the
reader is ready.

1. The backend returns an `enroll` action, a `session_uid`, and the allocated
   fingerprint slot.
2. The firmware reports `sensor_initializing`, then `waiting_first_scan`,
   captures the first image and reports `first_scan_captured`.
3. The finger must be removed; the firmware reports `waiting_for_removal` and
   `waiting_second_scan`.
4. The second image is captured and reported as `second_scan_captured`.
5. The driver compares both character buffers, merges them with `regModel`, and
   refuses to continue if the two scans do not match or the two samples are
   below the match threshold.
6. Before storing, the firmware verifies that the backend-allocated slot is
   empty; an occupied slot fails the session instead of overwriting a template.
7. The template is stored, then every other slot is searched for a duplicate
   finger. A duplicate deletes the new template and fails the session.
8. The slot is reported with `verifying` and the firmware calls
   `POST /api/v1/biometric/enrollment-sessions/:uid/complete`.
9. The member becomes enrolled only after the backend finalizes the enrollment.

If the sensor stored a template but the backend did not confirm it, the
firmware keeps the slot number and session UID in NVS, reports the failure, and
deletes the template (or retries cleanup until the sensor confirms). On reboot,
that state is reconciled with the backend before any new session is claimed, so
an untracked template is never left in the sensor. Failed, cancelled or expired
sessions delete the locally stored template. A backend `slot_conflict` response
means the backend recorded someone else's slot; the device never erases a slot
it does not own for that session.

## Backend contract

All routes are under `https://<host>/api/v1`. Device routes authenticate with
`X-Device-Token`; the token is obtained with the provisioned API key.

| Endpoint | Method | Purpose | Auth |
|---|---|---|---|
| `/api/v1/biometric/authenticate-device` | POST | Exchange `{api_key, device_name}` for a token | none |
| `/api/v1/biometric/readers` | POST | Report the connected entry reader | `X-Device-Token` |
| `/api/v1/biometric/enrollment-session` | GET | Claim cleanup or active enrollment work | `X-Device-Token` |
| `/api/v1/biometric/enrollment-sessions/:uid` | GET | Check status, cancellation, or expiry | `X-Device-Token` |
| `/api/v1/biometric/enrollment-sessions/:uid/claim` | POST | Claim the session before touching the sensor | `X-Device-Token` |
| `/api/v1/biometric/enrollment-sessions/:uid/progress` | POST | Report two-scan progress | `X-Device-Token` |
| `/api/v1/biometric/enrollment-sessions/:uid/complete` | POST | Finalize the slot and activate the member | `X-Device-Token` |
| `/api/v1/biometric/enrollment-sessions/:uid/fail` | POST | Report a sensor or match failure | `X-Device-Token` |
| `/api/v1/biometric/enrollment-sessions/:uid/cleanup` | POST | Acknowledge local slot cleanup | `X-Device-Token` |
| `/api/v1/biometric/door-access` | POST | Map a slot to a member and authorize access | `X-Device-Token` |
| `/api/v1/biometric/status` | GET | Device status / pending event count | `X-Device-Token` |

The backend also exposes `/api/v1/biometric/enrollments`, `/submit-event`,
`/enroll-fingerprint` and `/sync-events` for other clients; this sketch does not
call them. Door access creates the audit entry and the attendance event on the
backend, so the firmware does not submit attendance events itself.

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

Door-access response (the door opens only for `"authorized": true`):

```json
{
  "success": true,
  "authorized": true,
  "duplicate": false,
  "reason": "authorized",
  "unlock_seconds": 5,
  "member": { "id": 7, "name": "...", "login_id": "..." }
}
```

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
- **The exit button is not an emergency button and never creates a member
  logout event.** It does not call the backend.
- **The exit button is serviced on every loop iteration.** `checkExitButton()`
  and `updateDoor()` run before any network or sensor work, and no `delay()` in
  the sketch waits out the unlock timer. Backend calls and sensor searches are
  bounded (8s HTTP timeout, 4s full-library search) and the button is polled
  again immediately afterwards.
- Attendance is preserved through the backend: an authorized door-access
  request creates a `fingerprint_scan` device event and the existing
  `biometric.process_event` job records the login.

## Verification status

- `pio run`: **passes** (ESP32 image built, the sketch is compiled through the
  Arduino `.ino` preprocessor).
- Flash: about 93% of the 1.3 MB app partition; RAM about 15%.
- Backend contract inspected from `backend/internal/api/member_enrollment.go`,
  `backend/internal/api/door_access.go` and `backend/internal/api/biometric.go`.
- The former native `pio test -e native` state-machine tests were removed
  together with the modular `enrollment_flow.h` / `reader_readiness.h` files
  they included; the state machine now lives inside the sketch and is exercised
  through the build and on-device serial diagnostics.
- **No physical hardware testing has been performed.** The R307S, relay, exit
  button, LEDs and buzzer are untested on real hardware.

## Troubleshooting

1. **Wi-Fi never connects** — check the configuration block; the firmware
   retries every `WIFI_RETRY_INTERVAL` without blocking the loop.
2. **`Device auth HTTP 401`** — `DEVICE_API_KEY` does not match a provisioned,
   active device in the backend.
3. **Enrollment never starts** — check the serial log for
   `Enrollment poll: none`, confirm the reader was detected at boot
   (`Fingerprint reader: password verified`), and that the backend reports the
   reader as connected.
4. **Entry denied, fingerprint not enrolled** — the finger matches on the
   sensor but the slot is not registered to an active member on this device, so
   the backend keeps the door locked by design.
5. **No fingerprint detected** — verify the UART pins (RX 16, TX 17), the
   57600 baud rate and reader power. `R307S GenImg:` diagnostics in the serial
   log show whether the module answers at all.
6. **Door never unlocks** — confirm the relay active levels and that the device
   is authenticated and the slot is enrolled.
7. **Enrollment stuck after a reboot** — the firmware reconciles the NVS
   session UID and slot before claiming new work. Look for
   `Pending enrollment recovery` and cleanup errors in the serial log.
