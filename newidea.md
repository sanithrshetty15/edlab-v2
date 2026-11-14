# EdwinsLab v2 upgrade plan

## Scope

This upgrade keeps the existing EdwinsLab application and changes only the member
registration, door access, and deployment database paths.

- Admin-driven two-step member registration. Public self-registration remains
  disabled (`POST /api/v1/members/register` is an explicit 403 stub).
- Step 1 collects only Full Name, USN or Roll Number, Academic Year, and Gmail.
- Step 2 enrolls the member fingerprint with two scans on the entry reader.
- A confirmation email is sent only after fingerprint enrollment activates the
  member.
- Entry uses one R307S reader and always requires a synchronous backend decision.
- Exit uses the existing GPIO33 push button, unlocks locally, and re-locks after
  `DOOR_OPEN_MS`.
- Attendance remains unchanged and continues to be processed by the existing
  `biometric.process_event` job.
- The upgraded deployment uses a separate PostgreSQL database. The existing
  database is left untouched.

## Registration flow

1. An administrator opens Members and submits the four Step 1 fields.
2. The backend creates a user with `active = false` and a member with status
   `pending_enrollment`.
3. The member is redirected to fingerprint enrollment. The entry reader performs
   two scans and the backend finalizes the enrollment slot.
4. On finalization the backend activates the member and queues the confirmation
   email.
5. The member can use Forgot password after activation.

Validation rules:

- First-year members provide a roll number. Second-year and above members provide
  a USN.
- Email must be a valid Gmail address ending in `@gmail.com`.
- Login ID, USN, roll number, email, and phone must not already exist.
- The account remains inactive until fingerprint enrollment is complete.

## Confirmation email

- Subject: `EdwinsLab – Membership Registration Successful`
- Sender: `Edwins Lab <edwin.lab.aiet@gmail.com>`
- Provider: Resend when `RESEND_API_KEY` is configured, otherwise Gmail SMTP.
- Required Gmail variables: `GMAIL_USERNAME` and `GMAIL_APP_PASSWORD`.
- The email is queued only after enrollment finalization. Delivery failures are
  recorded and can be retried from the admin dashboard.

## Door access

Firmware flow:

1. The entry reader matches a fingerprint slot locally.
2. The firmware calls `POST /api/v1/biometric/door-access` with
   `fingerprint_slot` and `reader_key: "login"`.
3. The backend verifies the device token, slot ownership, member status, user
   status, and reader key.
4. The backend returns `authorized: true` only for an active member with a valid
   enrollment.
5. The firmware unlocks only when `authorized` is true and `duplicate` is false.

Backend response fields:

```json
{
  "success": true,
  "authorized": true,
  "duplicate": false,
  "reason": "authorized",
  "unlock_seconds": 5,
  "member": {
    "id": 1,
    "name": "Member Name",
    "login_id": "4AL25CS001"
  }
}
```

Security rules:

- A local fingerprint match never unlocks the door.
- Backend errors, timeouts, and inactive members deny access.
- Only the entry (`login`) reader is accepted for door access.
- Duplicate scans within three seconds are acknowledged but do not re-unlock.
- The exit button never calls the backend and never creates a logout event.

Attendance preservation:

- An authorized door-access request creates a `fingerprint_scan` device event
  with a unique idempotency key.
- The existing `biometric.process_event` job processes it through the unchanged
  attendance logic.
- No attendance tables, models, or business rules are modified.

## Database separation

No new tables are required for this upgrade. The existing schema already contains
member enrollment sessions, biometric enrollments, registration email records,
device events, and audit logs.

Migration strategy:

1. Create a new PostgreSQL database for the upgraded deployment.
2. Point `DATABASE_URL` at the new database. Never point it at the old database
   during cutover.
3. Start the backend once. Embedded migrations `0001` through `0007` run
   automatically and are recorded in `go_schema_migrations`.
4. Provision the two administrators with `ADMIN_ONE_PASSWORD` and
   `ADMIN_TWO_PASSWORD`, using `ADMIN_BOOTSTRAP_RESET_PASSWORDS=true` only for the
   initial bootstrap.
5. Verify login, member creation, fingerprint enrollment, email delivery, door
   access, and attendance on the new database.
6. Keep the old database untouched and available for rollback.

If existing data must be carried over, take an offline logical backup and restore
it into the new database before starting the new deployment. Do not drop or
truncate the old database, and do not run destructive migrations against it.

## Deployment variables

- `DATABASE_URL`: new PostgreSQL database.
- `GMAIL_USERNAME`: `edwin.lab.aiet@gmail.com`.
- `GMAIL_APP_PASSWORD`: Gmail app password.
- `MAILER_FROM`: `Edwins Lab <edwin.lab.aiet@gmail.com>`.
- `BIOMETRIC_API_SECRET`: shared device authentication secret.
- `ADMIN_ONE_PASSWORD`, `ADMIN_TWO_PASSWORD`: initial administrator passwords.

Do not commit real secrets. `backend/.env` is local-only and must not be included
in the repository.

## Verification

- Backend: `go test ./...`
- Frontend: `npm run lint`, `npm run build`
- Firmware native state-machine tests: `pio test -e native`
- Firmware build: `pio run -e esp32dev`
- Integration tests requiring PostgreSQL: set `TEST_DATABASE_URL` before running
  the enrollment integration tests.

## Open decisions

- Academic year currently allows years 1 through 8. The registration requirement
  only distinguishes first-year roll numbers from second-year-and-above USNs.
  Keep the wider range unless product requirements restrict it to 1 through 4.
