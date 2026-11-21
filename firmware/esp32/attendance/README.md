# Edwins Lab attendance terminal firmware

Production-oriented firmware for the separate ALF V1.0 attendance terminal:
**ESP32-WROOM-32UE**, 2.4 GHz Wi-Fi, onboard buzzer and a UART fingerprint
reader. It never controls the door relay, never calls the door API and shares
no source or build with `firmware/esp32/doorlock`.

## Layout

```
firmware/esp32/attendance/
├── platformio.ini              # esp32dev + native logic test env
├── include/
│   ├── config.h                # application timing, limits, secrets include
│   ├── hardware_config.h       # ALF V1.0 pins, UART, buzzer, optional LEDs
│   ├── gts_root_r4.h           # pinned TLS root CA
│   ├── app.h                   # controller, network task, scan state machine
│   ├── enrollment_manager.h    # non-blocking enrollment/recovery
│   ├── enrollment_store.h      # NVS recovery record
│   ├── fingerprint_reader.h    # R30x driver interface
│   ├── offline_queue.h         # durable NVS event queue
│   ├── api_client.h            # exact backend contracts
│   ├── wifi_manager.h          # reconnecting Wi-Fi
│   ├── feedback.h              # non-blocking buzzer/LED patterns
│   ├── attendance_logic.h      # host-testable pure logic
│   ├── logger.h, time_utils.h, event_id.h
│   └── attendance_secrets.example.h
├── src/                        # implementations + main.cpp
└── test/test_logic/            # native Unity tests
```

## Hardware configuration

The reader UART and buzzer values are the documented ALF V1.0 controller
values. They are defaults, not a claim that this attendance PCB has been
measured. The exact reader model/protocol and buzzer polarity still need
physical confirmation.

| Signal | Default | Status |
| --- | --- | --- |
| MCU | ESP32-WROOM-32UE, `esp32dev`, 4 MB flash | Confirmed by brief |
| Reader UART | UART2, RX GPIO16, TX GPIO17, 57600 8N1 | ALF V1.0 documented; verify |
| Reader protocol | R30x command set, slots 1-127 | Driver implemented; exact reader unverified |
| Buzzer | GPIO21, active HIGH | ALF V1.0 documented; verify polarity |
| Status LEDs | GPIO23 green / GPIO22 red, **disabled by default** | Attendance PCB LED population unverified |
| Relay / door | none | Not present in this firmware |

Override any value in `include/attendance_secrets.h`. If the reader is not
R30x-compatible, the protocol layer in `fingerprint_reader.cpp` must be
replaced; the backend contract does not change. Set an unavailable pin to
`-1`.

## Provisioning

1. As an administrator, create a device with the attendance type:

   ```http
   POST /api/v1/devices
   Content-Type: application/json
   Authorization: Bearer <admin token>

   {"name":"Main entrance attendance","device_type":"attendance_terminal"}
   ```

2. Copy the returned one-time `api_key` into
   `include/attendance_secrets.h` (copied from the example). The file is
   git-ignored. Never put admin credentials on the device.

3. On boot the firmware uses SNTP to obtain a valid clock, then calls
   `POST /api/v1/biometric/authenticate-device` with
   `{"api_key": ..., "device_name": ...}` and uses the returned
   `X-Device-Token` for every other request. It re-authenticates on `401/403`.

4. If the key is rotated or the device revoked, replace the key/re-provision;
   the terminal treats `401/403` as unauthorized and does not fabricate
   attendance.

### TLS

The deployed backend is verified against the pinned **GTS Root R4** CA in
`include/gts_root_r4.h` (valid until 2036-06-22, currently issuing through the
WE1 intermediate for `edwinslab.onrender.com`). Certificate verification is
never disabled. If the backend host or CA changes, update that header and
rebuild; a verification failure is a safe failure. Plain HTTP is refused
unless `ATTENDANCE_ALLOW_INSECURE_HTTP=1` is explicitly compiled for local
development.

## Build, test and flash

```bash
cd firmware/esp32/attendance
pio run                      # firmware build
pio test -e native           # host logic tests
pio run --target upload      # flash over USB
pio device monitor           # 115200 baud
```

The project uses a 3 MB application partition (`huge_app.csv`) because the
verified TLS stack and modular firmware exceed the default 1.3 MB app
partition's comfortable headroom. There is no OTA in this partition layout;
field updates are over USB unless an OTA layout is added later. The NVS
partition is at its normal offset, so reflashing does not erase the queue or
enrollment recovery record.

For a fresh ALF V1.0 board: connect USB, select **ESP32 Dev Module** (or the
ALF board definition), upload at 921600, then open the monitor at 115200. If
upload fails, hold **BOOT**, tap **RST**, release **BOOT**, and retry. Do not
erase NVS unless intentionally clearing queued attendance.

## Backend contract used

| Purpose | Request |
| --- | --- |
| Authenticate | `POST /api/v1/biometric/authenticate-device` |
| Submit one scan | `POST /api/v1/attendance-terminal/attendance` |
| Batch backlog | `POST /api/v1/attendance-terminal/attendance/batch` |
| Heartbeat | `POST /api/v1/attendance-terminal/heartbeat` |
| Poll command | `GET /api/v1/attendance-terminal/enrollment-session` |
| Session status | `GET /api/v1/attendance-terminal/enrollment-sessions/:uid` |
| Claim | `POST .../:uid/claim` |
| Progress | `POST .../:uid/progress` |
| Complete | `POST .../:uid/complete` |
| Fail | `POST .../:uid/fail` |
| Cleanup ack | `POST .../:uid/cleanup` |

The firmware sends only `idempotency_key`, `fingerprint_slot`, `result` and
`captured_at` for scans. It never sends an image, template or member personal
data. The backend owns slot-to-member mapping, Entry/Exit toggling and the
authoritative timestamp.

Enrollment progress uses only statuses this attendance backend accepts:
`waiting_for_finger`, `finger_detected` and `captured` (plus the backend's own
terminal states). The earlier sketch's `first_scan_captured` and related
statuses were rejected with `422` and have been removed.

## Offline queue

- Up to 32 events in NVS (`config::kQueueCapacity`), sent in batches of up to
  20 (backend limit 50).
- Every event is written before network submission and keeps its generated
  idempotency key across retries and reboots.
- Boot rebuilds the queue index by scanning all 32 NVS slots. There is no
  separate head/tail metadata that a power loss can desynchronize.
- Corrupt entries are logged and removed; duplicate sequences are rejected.
- Queue full: the scan is **not** recorded, an error pattern plays, and no
  unconfirmed event is overwritten.
- Confirmed and duplicate acknowledgements remove the event. Retryable
  failures remain queued with capped exponential backoff. Permanent backend
  rejections are logged and removed so they cannot block the queue.
- Runtime states map to: `PENDING` (queued), `SENDING` (in the current batch),
  `CONFIRMED` (removed after acknowledgement), `RETRY_REQUIRED` (kept after a
  transient failure), `FAILED` (permanent rejection, logged and removed).
- A failed scan never displays the confirmed feedback. The offline pattern is
  distinct from success.

## Enrollment and recovery

Enrollment is admin-driven and non-blocking:

1. Poll and claim the backend command.
2. Verify the backend-allocated slot is empty **before** reporting progress.
   An occupied slot fails without asking the backend to clean someone else's
   template.
3. Persist `{session_uid, slot, template_stored=false}` in NVS.
4. Capture two scans with removal detection, report accepted progress states,
   run `regModel`, search the existing library for a duplicate finger, store
   the template, and mark `template_stored=true`.
5. Complete the backend mapping. A transient completion failure is retried;
   a final failure reports the session and deletes the local template through
   the backend cleanup lifecycle.

On reboot, a pending record is reconciled with
`GET /enrollment-sessions/:uid`: completed sessions keep the template,
cleanup commands delete it and acknowledge, and a session that never stored a
template is failed and cleaned. Recovery is attempted only while the network
is authenticated; offline attendance scanning is not blocked by an
unreachable recovery check.

## State machine and concurrency

Explicit diagnostic states: `BOOT`, `HARDWARE_INIT`, `WIFI_CONNECTING`,
`BACKEND_CONNECTING`, `SYNCING`, `IDLE`, `FINGERPRINT_DETECTED`,
`FINGERPRINT_MATCHED`, `EVENT_PENDING`, `EVENT_SUBMITTING`,
`EVENT_CONFIRMED`, `ENROLLMENT_MODE`, `ERROR_RECOVERY`. Invalid transitions
are logged.

A dedicated 16 KB FreeRTOS network task handles Wi-Fi, SNTP, authentication,
heartbeat, batch upload and enrollment polling. The Arduino loop handles the
fingerprint reader, feedback and event state machine. The API client and NVS
queue are mutex-protected, so queue upload and heartbeat continue while the
reader is idle. HTTP operations are bounded (6 s normal, 5 s enrollment) and
never use unbounded loops.

## Feedback and scan reliability

Feedback is non-blocking (millis-based patterns, no `delay` in the buzzer
path): boot, ready, finger detected, confirmed entry (two short), confirmed
exit (one long), duplicate, unknown, offline queued, enrollment, Wi-Fi lost
and fault. Patterns use only the verified buzzer by default; optional LEDs
are behind `ATTENDANCE_HAS_STATUS_LEDS`.

After a scan the terminal waits for finger removal, then applies a 3 s cooldown
before re-arming. The backend's 10 s duplicate window remains authoritative.
Reader communication errors are counted; three consecutive failures trigger a
UART reinitialization. The reader is retried every 5 s after a failed start.

## Security

- No fingerprint image, template or feature data leaves the reader.
- No member names, USN or contact data are stored on the ESP32.
- Device key and Wi-Fi password live only in the git-ignored local header.
- No secret or credential is logged. Logs contain device IDs, session UIDs,
  slots, state names and HTTP status codes only.
- TLS certificate verification is always enabled.
- Enrollment commands are accepted only from the authenticated backend and
  only use the backend-allocated slot.
- No web server, AP mode, relay or door-unlock path exists.

## Verification status

- `pio run`: **passes** for `esp32dev` / Arduino ESP32 3.3.12.
- `pio test -e native`: **10/10 pass** (state transitions, HTTP retry classes,
  backoff cap, cooldown/deadline including millis wrap, slot and idempotency
  validation, placeholder detection).
- No compiler warnings were emitted by the final build.
- RAM: 50,400 bytes (15.4%). Flash: 1,268,776 bytes in the 3 MB app partition
  (40.3%).
- **Not hardware-tested.** The reader model/baud, buzzer polarity, pin map,
  actual enrollment behavior and TLS handshake have not been exercised on the
  physical ALF V1.0 attendance PCB.

## Known limitations

- The reader model is not confirmed. The driver implements the standard R30x
  UART protocol; a different sensor requires a new driver layer.
- The buzzer is assumed to be an active buzzer driven by GPIO level. A passive
  buzzer needs `tone()` and a frequency configuration. Polarity is unverified.
- Status LEDs are disabled by default because their population on this PCB is
  not confirmed. Enable them only after checking the board.
- TLS is pinned to GTS Root R4. If the backend changes CA, rebuild with the new
  root rather than disabling verification.
- There is no OTA partition. Firmware updates are over USB. The 3 MB app layout
  intentionally trades OTA for headroom.
- Flash encryption and secure boot are not enabled by this project. They are
  one-way eFuse provisioning steps and must be planned before production units
  are flashed.
- The queue holds 32 events. When full, new scans are rejected with an explicit
  error and are never allowed to overwrite unconfirmed attendance.
- The backend heartbeat schema has no uptime, RSSI or queue-capacity fields, so
  those are available in serial diagnostics rather than the dashboard.
- The backend rejects the door-enrollment progress labels
  (`first_scan_captured`, etc.) on this attendance endpoint. The dashboard
  therefore shows the accepted stages `waiting_for_finger`, `finger_detected`
  and `captured`.
- No physical ALF V1.0 attendance board, reader or buzzer has been tested.

## Hardware testing checklist

1. Confirm 3.3 V logic and reader supply; verify RX/TX orientation and common
   ground. Set `ATTENDANCE_READER_MODEL`/baud if the reader is not R30x at
   57600.
2. Power on: startup signal, reader password verification, Wi-Fi connection,
   SNTP, device authentication, first heartbeat.
3. Unknown finger: distinct error pattern, no attendance record.
4. Enrolled finger: entry confirmation, then exit confirmation; verify backend
   dashboard member/year/terminal/time.
5. Duplicate scan within cooldown: duplicate pattern, no extra record.
6. Finger left on sensor: no repeat scan until removal.
7. Disconnect Wi-Fi/backend, scan: offline warning; verify no success display
   and a pending queue entry. Restore connectivity and confirm batch replay
   with exactly one backend record.
8. Power-cycle with pending events: queue depth survives and replays.
9. Admin starts enrollment: two scans, removal prompt, duplicate-finger
   rejection, backend mapping appears.
10. Remove mapping: template disappears; a slot cannot be silently reused for
    another member without a new enrollment.
11. Cancel an enrollment mid-flow: no orphan template remains after cleanup.
12. Power-cycle during enrollment, after store and after completion: recovery
    leaves no unmapped template.
13. Long run (24 h+): no reboot loop, stable heap, heartbeat each 30 s,
    scanning responsive during queue upload.
