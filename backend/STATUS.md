# Edwins Lab — Backend Status

## Build status

Validated from `backend/` on 2026-11-12:

| Command | Result |
|---|---|
| `go mod tidy` | ✅ exit 0 |
| `go build ./...` | ✅ exit 0 |
| `go vet ./...` | ✅ exit 0 |
| `go test ./...` | ✅ exit 0 (`internal/api` tests pass) |
| `TEST_DATABASE_URL=... go test ./internal/api -run TestDeviceProvisioningLifecycleIntegration` | ✅ exit 0 (live PostgreSQL 18) |

The backend is a Go application (Gin + GORM + PostgreSQL). No Ruby or Rails files remain.

## Implementation

### Entrypoint and packages

- `cmd/api/main.go` — loads config, connects to PostgreSQL, runs migrations, provisions admins, seeds defaults, starts background workers and the HTTP server with graceful shutdown.
- `internal/api` — Gin router, handlers, route registration.
- `internal/config` — `.env` loading and production validation.
- `internal/database` — GORM connection and embedded SQL migrations.
- `internal/bootstrap` — idempotent admin provisioning and default seed data.
- `internal/security` — bcrypt hashing, session tokens, OTP, JWT.
- `internal/middleware` — authentication, CORS, audit, logging.
- `internal/jobs` — background job manager.
- `internal/mailer` — Resend / Gmail SMTP.
- `internal/twilio` — Twilio Verify.
- `internal/models` — GORM models.
- `internal/phone`, `internal/ratelimit`, `internal/httpx` — support packages.

### Database

- PostgreSQL is the only database.
- Migrations live in `internal/database/migrations/*.sql` and are embedded with `//go:embed`.
- Applied versions are tracked in `public.go_schema_migrations`.
- `pgcrypto` is created on startup.
- No GORM `AutoMigrate` is used; schema changes must be added as numbered SQL migrations.
- `0004_device_provisioning.sql` adds `device_uid` (unique), `token_version` and `revoked_at`.

### Authentication and APIs

- Two administrators are provisioned from `ADMIN_ONE_PASSWORD` and `ADMIN_TWO_PASSWORD` (each ≥12 characters); `ADMIN_BOOTSTRAP_RESET_PASSWORDS=true` resets existing passwords once.
- Member and admin authentication, sessions, OTP, password reset, member/admin management, attendance, biometric device auth/enrollment/events, borrowing, returns, inventory, messaging, announcements, notifications, tasks, events, FAQs, contact, analytics, audit logs, and uploads are implemented in `internal/api`.
- Device authentication uses `POST /api/v1/biometric/authenticate-device` with an API key whose SHA-256 digest is stored in `device_registrations.api_key_digest`.
- Admin-only device provisioning lives at `/api/v1/devices` (create/list/show/rotate-key/revoke). Raw keys are returned once; rotation and revocation increment `token_version`, which invalidates existing device tokens.

### Configuration

Environment variables are documented in `.env.example` and `README.md`. Required in production:

- `DATABASE_URL`
- `JWT_SECRET` (≥32 characters)
- `SESSION_SECRET` (≥32 characters)
- `ADMIN_ONE_PASSWORD`, `ADMIN_TWO_PASSWORD` (for first boot)

Optional: email (Resend or Gmail SMTP), Twilio Verify, `BIOMETRIC_API_SECRET`, upload limits, rate limits, session/OTP TTLs.

## Deployment

- `render.yaml` at the repository root builds `./backend/Dockerfile` with `./backend` as context and health-checks `/health`.
- The Dockerfile compiles `./cmd/api` statically, runs as non-root UID/GID `1001`, and creates `/app/data/uploads`.
- Migrations run automatically on container start.

## Pending manual actions

1. **Production `DATABASE_URL`** — set in the Render dashboard; required before the service can start.
2. **Admin passwords** — set `ADMIN_ONE_PASSWORD` and `ADMIN_TWO_PASSWORD` (≥12 characters).
3. **Email provider** — set `RESEND_API_KEY` or `GMAIL_USERNAME` / `GMAIL_APP_PASSWORD`.
4. **Render service** — verify the service points at `backend/Dockerfile`, uses the `backend/` context, and has all secrets set.
5. **Biometric hardware** — R307S/ESP32 enrollment and unlock still require physical device testing.
6. **Android release** — configure a signing keystore outside the repository and build a signed APK from `frontend/android`.
7. **Upload persistence** — the container filesystem is ephemeral; attach a persistent disk or object storage if uploads must survive redeploys.

## Administrator identifiers

- Admin 1: phone `+916362177190`, email `sanithrshetty15@gmail.com` (primary)
- Admin 2: phone `+919380455051`, email `diganthhm714@gmail.com`

Passwords come only from `ADMIN_ONE_PASSWORD` and `ADMIN_TWO_PASSWORD`; they are never stored in this document.

## Android release notes

- Application ID: `com.edwinslab.app`
- Reuse the same signing key for every update.
- Increment `versionCode` and set an appropriate `versionName` for each release.
- Required environment variables for signing: `ANDROID_KEYSTORE_PATH`, `ANDROID_KEYSTORE_PASSWORD`, `ANDROID_KEY_ALIAS`, `ANDROID_KEY_PASSWORD`.
- Build: `cd frontend/android && ./gradlew assembleRelease`.
- Output: `frontend/android/app/build/outputs/apk/release/app-release.apk`.
