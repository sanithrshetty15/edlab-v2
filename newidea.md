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
- The public footer now includes official LinkedIn, Instagram, X and email
  links with accessible brand-coloured hover states.

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

## Footer social links and contact email

The public footer uses one central `socialLinks` configuration in
`frontend/src/content/labContent.js`:

| Platform | URL | Hover colour |
|---|---|---|
| LinkedIn | `https://www.linkedin.com/company/edwinslab/` | `#0A66C2` |
| Instagram | `https://www.instagram.com/edwinslab.ig?igsh=MWlnaGZxZ2l0OWo0cg==` | `#E4405F` plus a subtle gradient |
| X (Twitter) | `https://x.com/edwinslabs` | `#FFFFFF` on the dark footer (`#000000` in light mode) |
| Email | `mailto:edwin.lab.aiet@gmail.com` | `#EA4335` |

Footer behaviour:

- Official platform SVG marks are used, not text symbols or emoji.
- Icons match the existing muted footer theme by default.
- Hover transitions run for 240ms and add the brand colour to the icon, border,
  background and a restrained glow, with a small scale lift.
- External links open in a new tab with `target="_blank"` and
  `rel="noopener noreferrer"`.
- The email link uses `mailto:` and opens in the same context.
- Every link has an `aria-label` and a native `title` tooltip.
- The icon row wraps on narrow screens and preserves the existing footer
  navigation, email text link and back-to-top link.
- `prefers-reduced-motion` disables the lift and transitions.

The official lab contact email is `edwin.lab.aiet@gmail.com`. Repository searches
found no remaining legacy library-mailbox references. Personal administrator
emails (`sanithrshetty15@gmail.com`, `diganthhm714@gmail.com`) and the
`edwins@aiet.org.in` administrator identity were intentionally left unchanged.

## Backend Gmail configuration verification

- `backend/internal/mailer/mailer.go` reads `GMAIL_USERNAME` and
  `GMAIL_APP_PASSWORD` with `os.Getenv`; no credentials are hardcoded.
- `backend/internal/config/config.go` reads `GMAIL_USERNAME` when deriving the
  default `MAILER_FROM`.
- Registration confirmation recipients come from the member's registered email
  in `ensureRegistrationEmail` / `queueRegistrationConfirmation`.
- The sender defaults to `Edwins Lab <edwin.lab.aiet@gmail.com>` when
  `MAILER_FROM` is unset.
- `backend/.env.example` documents both Gmail variables and the lab address.
- Email delivery itself was not tested; no message was sent or received during
  this change.

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

- Backend: `go test ./...` passes.
- Frontend: `npm run lint` reports 0 errors and 1 pre-existing React hook
  warning; `npm run build` passes.
- Headless browser check at 1440px and 375px: all four footer links resolve,
  external links use `_blank` + `noopener noreferrer`, hover colours match the
  brand palette, transitions are 240ms, and there is no horizontal overflow or
  console error.
- Firmware native state-machine tests: `pio test -e native` passes.
- Firmware build: `pio run -e esp32dev` passes.
- Integration tests requiring PostgreSQL: set `TEST_DATABASE_URL` before running
  the enrollment integration tests.

Footer/contact update files:

- `frontend/src/sections/Footer.jsx`
- `frontend/src/components/common/SocialIcons.jsx`
- `frontend/src/content/labContent.js`
- `frontend/src/App.css`
- `frontend/README.md`
- `frontend/docs/CONTENT-VERIFICATION.md`
- `newidea.md`

## Mobile login and Android 1.2 release

### Root cause

The installed Android 1.1 APK still used the old admin login contract:

1. The app sent an admin email and password to `POST /api/v1/auth/admin-login`.
2. The upgraded backend no longer returns a session token from that endpoint; it
   returns `{ "purpose": "admin_login" }` and emails a one-time code.
3. The old app passed that response to `completeLogin`, which requires
   `data.token`, producing `"The server did not return a session token."`.

The current frontend source also had an incomplete OTP flow: the admin verify API
and `verifyAndCompleteAdminLogin` were missing, and the member password field and
submit button had accidentally been placed inside the admin form branch.

### Fix

- `frontend/src/api/services.js`
  - `adminLogin` sends only the admin email.
  - Added `verifyAdminOTP` for `POST /api/v1/auth/admin-verify-otp`.
- `frontend/src/auth/AuthContext.jsx`
  - Admin login now stores `pending_admin_otp` state.
  - Added `verifyAndCompleteAdminLogin`, which verifies the OTP, stores the
    returned token through `completeLogin`, and exposes the admin session.
- `frontend/src/pages/LoginPage.jsx`
  - Restored the member password field and shared submit button.
  - Admin login now asks for email only, then shows the OTP verification screen.
  - Admin submit label is `Send verification code`; OTP submit remains `Verify`.

No backend authentication changes were required.

### Verification

- Playwright native-shell flow with mocked API:
  - Valid member login, authenticated `/auth/me`, session persistence after
    reload.
  - Valid admin email plus OTP login, authenticated `/auth/me`, session
    persistence after reload.
  - Invalid member password returns `Invalid login credentials` and clears the
    password.
  - Missing token response shows `The server did not return a session token.`.
  - Invalid admin OTP shows `Invalid or expired verification code`.
  - Expired session returns the login screen with `Session expired`.
- `go test ./...`: 10 passed in 14 packages.
- `npm run build`: passed.
- `npm run lint`: 0 errors, 1 pre-existing React hook warning.
- Local backend: `admin-login` returned `purpose: admin_login`; an invalid
  `admin-verify-otp` returned HTTP 422 with `Invalid or expired verification
  code`.
- Email OTP delivery was not verified because the mailbox was not accessible.

### APK artifact

- Version: `1.2` (`versionCode 3`), signed with the existing release keystore.
- APK: `frontend/public/downloads/EdwinsLab-1.2.apk`
- Manifest: `frontend/public/downloads/edwinslab-apk.json`
- Manifest marks the update as mandatory so Android 1.1 builds with the broken
  admin login are replaced.
- SHA-256: `11a5e5f419fa5aaea18340961a5f0b18727e6e217428b827544f648554f7adfa`
- The APK and manifest are staged locally but not deployed. Deploy the frontend
  so `/downloads/EdwinsLab-1.2.apk` and the manifest are live on Vercel.

Mobile login files:

- `frontend/src/api/services.js`
- `frontend/src/auth/AuthContext.jsx`
- `frontend/src/pages/LoginPage.jsx`
- `frontend/android/app/build.gradle`
- `frontend/public/downloads/EdwinsLab-1.2.apk`
- `frontend/public/downloads/edwinslab-apk.json`
- `newidea.md`

## Open decisions

- Academic year currently allows years 1 through 8. The registration requirement
  only distinguishes first-year roll numbers from second-year-and-above USNs.
  Keep the wider range unless product requirements restrict it to 1 through 4.
