# Edwins Lab Backend — STATUS

_Last updated: 2026-11-11._

## Authentication access changes (2026-11-11)

- Public member self-registration is disabled. `POST /api/v1/members/register`
  now always returns HTTP 403 and does not create users or members.
- Member creation moved behind admin authentication:
  `POST /api/v1/members-management` requires an authenticated admin. Member
  sessions receive HTTP 403 and unauthenticated requests receive HTTP 401.
- The endpoint reuses `MemberRegistrationService` and writes a
  `member_created_by_admin` audit log entry.
- Verified with `test/integration/member_registration_access_test.rb`:
  4 runs, 16 assertions, 0 failures. Live checks against the local API also
  confirmed 403/401/201 behavior and successful member/admin logins.
- Production API and deployment remain unverified because
  `https://edwinslab.onrender.com` still returns HTTP 502.

## Borrowing extension correction (2026-11-11)

- Restored `POST /api/v1/borrowing/extension-request` and
  `BorrowingController#request_extension`.
- Added `db/migrate/20261111200000_create_extension_requests.rb` plus the
  `BorrowRecord#extension_request` and `ExtensionRequest#member` associations.
  The migration has been applied to the local development and test databases;
  production has not been migrated or verified.
- `request_extension` accepts an optional `requested_return_date` (defaults to
  14 days from today) and rejects non-overdue borrow records, missing reasons,
  invalid dates, and dates that are not after the current due date.
- `serialize_extension_request` is restored, and the full borrow-record
  serializer now includes the extension request when present.
- Removed the global `rescue_from StandardError` handler while retaining the
  specific `RecordNotFound` and `RecordInvalid` handlers.
- Verified with `bin/rails routes -g borrowing`:
  `POST /api/v1/borrowing/extension-request` maps to
  `api/v1/borrowing#request_extension`.
- Full Rails test suite: 6 runs, 0 failures, including
  `test/integration/borrowing_extension_test.rb`.
- Changes remain uncommitted; nothing was committed or pushed.

## Background job architecture (current state)

### 1. Solid Queue configuration / operation
**Status (verified by code inspection): configured for production; runtime operation not verified.**

- `Gemfile` includes `solid_queue` (lock: `solid_queue 1.7.0`), plus `solid_cache` and `solid_cable`.
- `config/environments/production.rb`
  - `config.active_job.queue_adapter = :solid_queue`
  - `config.solid_queue.connects_to = { database: { writing: :queue } }`
- `config/database.yml` defines a production `queue` database (`backend_production_queue`, `migrations_paths: db/queue_migrate`); `db/queue_schema.rb` contains Solid Queue tables (`solid_queue_jobs`, execution tables, etc.).
- `config/queue.yml` (default/dev/test/prod): dispatcher `polling_interval: 1`, `batch_size: 500`; workers on `queues: "*"`` threads: 3`, processes: `${JOB_CONCURRENCY:-1}`.
- Execution entrypoints:
  - `bin/jobs` starts `SolidQueue::Cli`.
  - `config/puma.rb` enables the Solid Queue plugin when `SOLID_QUEUE_IN_PUMA` is set.
  - `config/deploy.yml` sets `SOLID_QUEUE_IN_PUMA: true` (run supervisor inside Puma); a separate `job` server (`cmd: bin/jobs`) is commented out.
- **Environment note:** `development.rb` and `test.rb` do **not** set `config.active_job.queue_adapter`. Production is explicitly Solid Queue; dev/test use Rails defaults unless overridden.
- **Not verified:** actual worker/supervisor runtime health, job throughput, or production execution.

### 2. Redis usage
**Status (verified by code inspection): no active runtime Redis dependency found.**

- No `redis` gem in `Gemfile`.
- No active Redis configuration found in app/runtime config. Redis references present are non-runtime placeholders/comments:
  - `.github/workflows/ci.yml`: Redis service / `REDIS_URL` commented out.
  - `config/deploy.yml`: Valkey/Redis accessory commented out.
  - `.env.example` includes an optional placeholder: `# Redis/Cache (if using Redis)` and `REDIS_URL=redis://localhost:6379/0` (template only, not evidence of use).
- Production uses the Postgres-backed Solid stack instead of Redis:
  - Jobs: `:solid_queue` + queue DB.
  - Cache: `config.cache_store = :solid_cache_store` (`production.rb`), `config/cache.yml` uses `database: cache`, `db/cache_schema.rb` defines `solid_cache_entries`.
  - Action Cable: `config/cable.yml` production uses `solid_cable` with `connects_to: { database: { writing: :cable } }` (dev `async`, test `test`).

### 3. Job execution: `EventProcessingJob` and notification jobs
**Status (verified by code inspection):**

#### `EventProcessingJob`
- `app/jobs/event_processing_job.rb` (`ApplicationJob`, `queue_as :default`) processes biometric `DeviceEvent` records (fingerprint scan/login-logout attendance + audit logging, mismatch/enrollment handling).
- Enqueued asynchronously from `app/controllers/api/v1/biometric_controller.rb:256`: `EventProcessingJob.perform_later(device_event.id)`.
- Error handling: rescues exceptions, logs, and marks the `DeviceEvent` `processing_status: "failed"` / `processing_error`; it does not re-raise for adapter retry by default.
- Production execution path: Active Job → **Solid Queue** (`default` queue) → Solid Queue worker/supervisor (`bin/jobs` or Puma plugin when `SOLID_QUEUE_IN_PUMA=true`).

#### Notification jobs
- No `*NotificationJob` Active Job classes found in `app/jobs/`.
- Notifications are sent via **Action Mailer `.deliver_later`** (Active Job-backed) from `app/services/notification_service.rb`:
  - `AdminMailer.notification(...).deliver_later`
  - `UserMailer.announcement_notification(...).deliver_later`
- These use the configured Active Job adapter (Solid Queue in production), not custom notification job classes.

### 4. Scheduled jobs, retries, and failed-job handling
**Status (verified by code inspection): partially configured.**

- **Scheduled/recurring:** `config/recurring.yml` (production) defines `clear_solid_queue_finished_jobs` running `SolidQueue::Job.clear_finished_in_batches(sleep_between_batches: 0.3)` on `every hour at minute 12`. No other app-specific recurring jobs found.
- **Retries/discards (app code):** `app/jobs/application_job.rb` contains commented `retry_on ActiveRecord::Deadlocked` and `discard_on ActiveJob::DeserializationError`; `EventProcessingJob` has no `retry_on`/`discard_on`/`limits_concurrency`. `EventProcessingJob` catches/report failures to `DeviceEvent` instead of re-raising.
- **Failed-job storage:** `db/queue_schema.rb` includes `solid_queue_failed_executions` and related execution tables (adapter-level failure tracking available). Effective retry/failure behavior for specific jobs is **not runtime-tested**.
- **Ops/monitoring:** no failed-job dashboard/alerting config identified in this inspection; separate job deployment is available but commented in `config/deploy.yml`.

### 5. Is Redis necessary?
**Status (verified by code inspection): no, not necessary for the current architecture.**

- Current jobs/cache/cable are configured to use the Postgres-backed Solid stack (Solid Queue/Cache/Cable). No Redis gem or active Redis runtime configuration was found.
- Redis/Valkey would only be necessary if the project intentionally switches to Redis-backed adapters/components (e.g., `redis_cache_store`, Redis Action Cable, Sidekiq, or another feature requiring `REDIS_URL`).

## Recommendation: `REDIS_URL` in `.env`

**Do not add `REDIS_URL` to `.env` for the current setup** (inspection-based recommendation).

- Not required by the current configured architecture.
- `.env.example` already contains an optional placeholder; adding an unused `REDIS_URL` to a real `.env` can be misleading.
- Add/configure `REDIS_URL` only if a Redis-dependent component is intentionally introduced; use the real environment-specific URL and keep credentials out of committed files.

## Verification summary

| Area | Verified by code inspection | Verified by automated tests | Verified at runtime/production |
|---|---:|---:|---:|
| Solid Queue/Cache/Cable gems & config files | Yes | No | No |
| Production Active Job adapter = `:solid_queue` | Yes | No | No |
| `EventProcessingJob` enqueue path & implementation | Yes | No | No |
| Notification delivery via `deliver_later` (mailer/Active Job) | Yes | No | No |
| Recurring cleanup job config (`recurring.yml`) | Yes | No | No |
| No active runtime Redis usage found | Yes | No | No |
| Worker/supervisor actually running, jobs/failures handled in prod | No | No | No |
