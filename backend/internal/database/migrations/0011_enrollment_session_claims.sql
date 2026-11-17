-- Track when a biometric device first receives an enrollment command.
--
-- The admin UI uses claimed_at to prove the command reached the ESP32 instead
-- of only knowing that it was queued. Claiming is idempotent for the device
-- that owns the session and must never hand the same command to another
-- device.

ALTER TABLE public.member_enrollment_sessions
    ADD COLUMN IF NOT EXISTS claimed_at timestamp(6) without time zone;

ALTER TABLE public.member_enrollment_sessions
    ADD COLUMN IF NOT EXISTS claimed_by_device_id bigint;

CREATE INDEX IF NOT EXISTS index_member_enrollment_sessions_device_claimed
    ON public.member_enrollment_sessions USING btree (device_registration_id, claimed_at)
    WHERE is_active = true;
