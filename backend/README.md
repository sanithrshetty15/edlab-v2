# Edwins Lab Backend

Go API for the Edwins Lab platform. The backend was migrated from Ruby on Rails; **Go is the only backend runtime**. The Rails application, gems, tests, credentials, and deployment files have been removed.

## Stack

- **Go 1.26+**
- **Gin** — HTTP router and middleware
- **GORM** — PostgreSQL ORM
- **PostgreSQL** — primary datastore
- Embedded SQL migrations (no external migration tool)
- bcrypt password hashing, server-side sessions, OTP, JWT admin/member auth
- Resend or Gmail SMTP for email, Twilio Verify for SMS

## Layout

```
backend/
├── cmd/api/main.go              # Entrypoint: config, DB, migrations, bootstrap, HTTP server
├── internal/
│   ├── api/                     # Gin server, route registration, handlers
│   ├── bootstrap/               # Admin provisioning and default seed data
│   ├── config/                  # Environment loading and validation
│   ├── database/                # GORM connection + embedded SQL migrations
│   │   └── migrations/*.sql
│   ├── jobs/                    # Background job manager
│   ├── mailer/                  # Resend / Gmail SMTP
│   ├── middleware/              # Auth, CORS, audit, logging, rate limiting
│   ├── models/                  # GORM models
│   ├── phone/                   # Phone normalization/validation
│   ├── ratelimit/               # Token-bucket limiter
│   ├── security/                # Password hashing, sessions, OTP, JWT
│   └── twilio/                  # Twilio Verify client
├── Dockerfile
├── .dockerignore
├── .env.example
├── go.mod
└── go.sum
```

## Requirements

- Go **1.26.10** or newer (see `go.mod`)
- PostgreSQL **14+**
- `pgcrypto` extension (created automatically on startup)

## Local development

```bash
cd backend

# 1. Create the environment file and fill in real values.
cp .env.example .env

# 2. Create the database (or point DATABASE_URL at an existing one).
createdb edwinslab_development

# 3. Download dependencies and run.
go mod download
go run ./cmd/api
```

The server listens on `PORT` (default `8080`). On startup it:

1. Loads `.env` (and `backend/.env`) if present.
2. Connects to PostgreSQL and verifies the connection.
3. Creates `pgcrypto` if missing.
4. Runs embedded SQL migrations, recorded in `go_schema_migrations`.
5. Provisions the two required administrators when their passwords are set.
6. Seeds default group settings and lab-rules versions.
7. Starts background workers and the HTTP server.

Because migrations are embedded with `//go:embed`, the compiled binary is self-contained; no migration files need to be copied into the runtime image.

### Admin bootstrap

`ADMIN_ONE_PASSWORD` and `ADMIN_TWO_PASSWORD` must each be at least 12 characters. If an admin does not exist and the password is unset, startup logs a warning and skips that admin. To provision or reset passwords on an existing database, run once with:

```bash
ADMIN_BOOTSTRAP_RESET_PASSWORDS=true go run ./cmd/api
```

## Environment variables

All variables are read by `internal/config/config.go`. Values in `.env.example` are placeholders — never commit real secrets.

### Core

| Variable | Default | Notes |
|---|---|---|
| `APP_ENV` | `development` | `production` enables strict validation |
| `PORT` | `8080` | HTTP listen port |
| `APP_TIMEZONE` | `Asia/Kolkata` | IANA timezone |
| `JWT_SECRET` | random per boot | **Required, ≥32 chars in production** |
| `SESSION_SECRET` | random per boot | **Required, ≥32 chars in production** |
| `FRONTEND_URL` | `http://localhost:5173` | Added to CORS allow-list |
| `CORS_ALLOWED_ORIGINS` | localhost defaults | Comma-separated origins |

### Database

| Variable | Default | Notes |
|---|---|---|
| `DATABASE_URL` | local dev URL | **Required in production** |

Example: `postgres://user:password@host:5432/edwinslab?sslmode=require`

### Administrators

| Variable | Notes |
|---|---|
| `ADMIN_ONE_PASSWORD` | ≥12 characters; used only when admin #1 is missing |
| `ADMIN_TWO_PASSWORD` | ≥12 characters; used only when admin #2 is missing |
| `ADMIN_BOOTSTRAP_RESET_PASSWORDS` | `true` to reset existing admin passwords once |

### Email

| Variable | Notes |
|---|---|
| `RESEND_API_KEY` | Resend provider |
| `GMAIL_USERNAME` / `GMAIL_APP_PASSWORD` | Gmail SMTP sender; use `edwin.lab.aiet@gmail.com` and a Gmail App Password |
| `MAILER_FROM` | From header; defaults from `GMAIL_USERNAME` or `Edwins Lab <edwin.lab.aiet@gmail.com>` |

### OTP and sessions

`OTP_EXPIRY_SECONDS` (600), `OTP_MAX_ATTEMPTS` (5), `OTP_REQUEST_WINDOW_SECONDS` (900), `OTP_MAX_REQUESTS` (5), `OTP_TEST_RECIPIENT`, `SESSION_TTL_SECONDS` (86400), `ADMIN_SESSION_TTL_SECONDS` (86400).

### SMS

`TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_VERIFY_SERVICE_SID`.

### Uploads

| Variable | Default | Notes |
|---|---|---|
| `UPLOAD_DIR` | `data/uploads` | Created relative to the working directory |
| `MAX_UPLOAD_BYTES` | `20971520` (20 MiB) | Per-file limit |

### Biometric devices

`BIOMETRIC_API_SECRET` is optional. When set, it signs device tokens and event HMACs; when empty the backend falls back to `SESSION_SECRET`. Set it in production if you want device credentials to survive session-secret rotation.

### Device provisioning

Device API keys are generated by the backend and shown **once**. Only the SHA-256 hex digest is stored in `device_registrations.api_key_digest`; the raw key never enters PostgreSQL.

All provisioning routes require an **admin session** (`Authorization: Bearer <token>` from `POST /api/v1/auth/admin-login`):

| Endpoint | Purpose |
|---|---|
| `POST /api/v1/devices` | Provision a device and return its API key once |
| `GET /api/v1/devices` | List devices (no secrets) |
| `GET /api/v1/devices/:id` | Show one device (no secrets) |
| `POST /api/v1/devices/:id/rotate-key` | Replace the key and invalidate old tokens |
| `POST /api/v1/devices/:id/revoke` | Revoke the device immediately |

Provision request:

```bash
curl -X POST https://<host>/api/v1/devices \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"ALF V1.0 ESP32","device_type":"esp32"}'
```

`device_uid` is optional. When omitted the server generates `esp32-<16 hex>`. If supplied it must be unique (4–64 characters: letters, digits, `.`, `_`, `:`, `-`).

`201 Created` response:

```json
{
  "success": true,
  "message": "Device provisioned successfully. Store the API key now; it will not be shown again.",
  "device": {
    "id": 7,
    "device_uid": "esp32-0123456789abcdef",
    "name": "ALF V1.0 ESP32",
    "device_type": "esp32",
    "status": "active",
    "last_seen_at": null,
    "created_at": "2026-11-12T19:15:34Z",
    "updated_at": "2026-11-12T19:15:34Z"
  },
  "api_key": "<64 lowercase hex characters>"
}
```

The response is sent with `Cache-Control: no-store`. Copy `api_key` into the firmware's `config.local.h` immediately; it cannot be retrieved later. If it is lost, rotate the key.

Rotation:

```bash
curl -X POST https://<host>/api/v1/devices/7/rotate-key \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

Returns a new `api_key` once. The old key stops working immediately, and previously issued `X-Device-Token` tokens are invalidated by incrementing `token_version`.

Revocation:

```bash
curl -X POST https://<host>/api/v1/devices/7/revoke \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

Revocation is immediate: device authentication returns `403`, and existing tokens return `401`. Revoked devices cannot rotate keys; provision a replacement.

Security notes:

- Raw keys are never stored in PostgreSQL, logs, or list/detail responses.
- Only provisioning and rotation responses return `api_key`, once each, with `no-store`.
- `BIOMETRIC_API_SECRET` remains independent of per-device keys; the optional HMAC-only event path is not invalidated by rotating a device key.

### Member fingerprint enrollment

Creating a member through `POST /api/v1/members-management` now requires a
name, academic year, email, and either a roll number (first year) or USN
(second year and above). It creates an inactive `pending_enrollment` member and
a `pending_details` session; it does not activate the member. The login ID is
derived from the roll number or USN, and a temporary password is generated when
one is not supplied. An administrator then selects an available reader and
starts fingerprint enrollment:

| Endpoint | Method | Purpose |
|---|---|---|
| `/api/v1/members-management/:id/fingerprint-enrollment/readers` | GET | List available enrollment readers |
| `/api/v1/members-management/:id/fingerprint-enrollment` | POST | Start or resume enrollment with `device_registration_id` and `reader_key` |
| `/api/v1/members-management/:id/fingerprint-enrollment` | GET | Read the latest session |
| `/api/v1/members-management/:id/fingerprint-enrollment` | DELETE | Cancel and delete the pending member |
| `/api/v1/members-management/:id/registration-email` | GET | Read confirmation-email delivery status |
| `/api/v1/members-management/:id/registration-email/retry` | POST | Retry a failed confirmation email |

Device session routes use `X-Device-Token` and are polled by the ESP32:

| Endpoint | Method | Purpose |
|---|---|---|
| `/api/v1/biometric/enrollment-session` | GET | Claim cleanup or active work |
| `/api/v1/biometric/enrollment-sessions/:uid` | GET | Read status/cancellation/expiry |
| `/api/v1/biometric/enrollment-sessions/:uid/progress` | POST | Report two-scan progress |
| `/api/v1/biometric/enrollment-sessions/:uid/complete` | POST | Finalize slot and activate member |
| `/api/v1/biometric/enrollment-sessions/:uid/fail` | POST | Report failure |
| `/api/v1/biometric/enrollment-sessions/:uid/cleanup` | POST | Acknowledge local cleanup |

Session statuses are `pending_details`, `awaiting_sensor`,
`waiting_first_scan`, `first_scan_captured`, `waiting_for_removal`,
`waiting_second_scan`, `second_scan_captured`, `verifying`, `finalized`,
`failed`, `cancelled`, and `expired`. Sessions expire after 10 minutes.
Completion is transactional and idempotent: the backend locks the session and
member, verifies the pending state, slot range, slot ownership, and expiry,
then creates the biometric enrollment and activates the member. Templates are
never uploaded; only `device_user_id` and `fingerprint_slot` are stored.

Cancellation deletes the pending member/user but keeps the session so the
device can clean up any stored template. When a slot is already owned by
another member, finalization marks the session failed with
`cleanup_required=false` so the device does not erase another member's
template.

After finalization commits, the backend queues a branded confirmation email to
the member. Delivery status is tracked as `pending`, `queued`, `sending`,
`sent`, or `failed`. A delivery failure does not roll back activation; the
member remains active and an administrator can retry the email from the
members screen.

### Rate limiting

`RATE_LIMIT_REQUESTS` (100) and `RATE_LIMIT_PERIOD_SECONDS` (60).

## API

- `GET /health`, `GET /up`, `GET /api/v1/health` — liveness/readiness
- `POST /api/v1/auth/...` — member/admin login, logout, session refresh, password reset
- `/api/v1/members`, `/api/v1/members-management`, `/api/v1/admins` — member and admin management
- `/api/v1/attendance` — attendance list, today, summary, analytics, manual entry, logout
- `/api/v1/biometric` — device authentication, two-scan fingerprint enrollment sessions, status, event submission/sync
- `/api/v1/devices` — admin device provisioning, listing, key rotation, revocation
- `/api/v1/borrowing`, `/api/v1/returns`, `/api/v1/inventory` — inventory workflow
- `/api/v1/messages`, `/api/v1/announcements`, `/api/v1/notifications` — communication
- `/api/v1/tasks`, `/api/v1/events`, `/api/v1/faqs`, `/api/v1/contact`, `/api/v1/analytics`, `/api/v1/audit-logs`
- `/api/v1/inventory/import` — admin CSV/XLSX inventory upload (limited by `MAX_UPLOAD_BYTES`)

All `/api/v1` routes are registered in `internal/api/*.go`; authentication and role checks are applied per route.

## Validation

```bash
cd backend
go mod tidy
go build ./...
go vet ./...
go test ./...
```

The device lifecycle test uses a real PostgreSQL database and is skipped unless `TEST_DATABASE_URL` is set:

```bash
TEST_DATABASE_URL='postgres://user:password@localhost:5432/edwinslab_test?sslmode=disable' \
  go test ./internal/api -run TestDeviceProvisioningLifecycleIntegration -v
```

The two-scan enrollment tests are also integration-only:

```bash
TEST_DATABASE_URL='postgres://user:password@localhost:5432/edwinslab_test?sslmode=disable' \
  go test ./internal/api -run TestMemberFingerprintEnrollment -v
```

## Docker

Build from the `backend/` directory (matches Render):

```bash
docker build -t edwinslab-api ./backend
docker run --rm --env-file backend/.env -p 8080:8080 edwinslab-api
```

The image:

- Compiles `./cmd/api` with `CGO_ENABLED=0`.
- Runs as non-root UID/GID `1001`.
- Creates `/app/data/uploads` for uploads.
- Excludes `.env`, git metadata, binaries, and local data via `.dockerignore`.

## Render deployment

`render.yaml` at the repository root defines the `edwinslab-api` web service:

- Runtime: Docker
- Dockerfile: `./backend/Dockerfile`
- Context: `./backend`
- Health check: `/health`

Set these in the Render dashboard:

- `DATABASE_URL` — PostgreSQL connection string
- `ADMIN_ONE_PASSWORD`, `ADMIN_TWO_PASSWORD` — bootstrap passwords (≥12 chars)
- `FRONTEND_URL`, `CORS_ALLOWED_ORIGINS` — deployed frontend origin(s)
- `RESEND_API_KEY` or Gmail credentials
- Optional: `BIOMETRIC_API_SECRET`, Twilio variables

`JWT_SECRET` and `SESSION_SECRET` are generated by Render. Do not reuse secrets from the previous deployment.

Migrations run automatically on container start. Ensure the database is reachable and `pgcrypto` can be created by the configured role.

## Notes

- No Rails files, gems, Rake tasks, credentials, or Rails tests remain.
- SQL migrations live only in `internal/database/migrations/` and are embedded at build time.
- Uploaded files are local to the container by default; use a persistent disk or object storage if uploads must survive redeploys.
