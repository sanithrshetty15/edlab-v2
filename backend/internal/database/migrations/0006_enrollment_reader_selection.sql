-- Admin-selectable enrollment readers.
--
-- The ESP32 reports which physical readers are configured and connected.
-- Enrollment sessions record the explicitly selected reader, and new biometric
-- mappings record the reader that owns the template. Legacy mappings keep a
-- null reader_key so devices can preserve their previous behaviour until they
-- are re-enrolled.

ALTER TABLE public.biometric_enrollments
    ADD COLUMN IF NOT EXISTS reader_key character varying(32);

ALTER TABLE public.member_enrollment_sessions
    ADD COLUMN IF NOT EXISTS reader_key character varying(32);

CREATE TABLE IF NOT EXISTS public.device_readers (
    id bigserial PRIMARY KEY,
    device_registration_id bigint NOT NULL,
    reader_key character varying(32) NOT NULL,
    label character varying(64) NOT NULL,
    uart integer NOT NULL DEFAULT 0,
    connected boolean NOT NULL DEFAULT false,
    last_seen_at timestamp(6) without time zone,
    created_at timestamp(6) without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp(6) without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_device_readers_device_key
    ON public.device_readers USING btree (device_registration_id, reader_key);

CREATE INDEX IF NOT EXISTS index_device_readers_on_device_registration_id
    ON public.device_readers USING btree (device_registration_id);

-- Sessions started before reader selection existed cannot be claimed safely:
-- the device would have to guess which sensor contains the template. Fail them
-- without requesting cleanup rather than deleting a potentially wrong slot.
UPDATE public.member_enrollment_sessions
SET status = 'failed',
    is_active = false,
    completed_at = CURRENT_TIMESTAMP,
    error_code = 'reader_selection_required',
    error_message = 'Enrollment was restarted because no reader was selected',
    cleanup_required = false,
    updated_at = CURRENT_TIMESTAMP
WHERE is_active = true
  AND status <> 'pending_details'
  AND reader_key IS NULL;

-- Only one enrollment session may be active on a device at a time, even if the
-- admin opens enrollment for different members in separate browser tabs.
CREATE UNIQUE INDEX IF NOT EXISTS index_member_enrollment_sessions_active_device
    ON public.member_enrollment_sessions USING btree (device_registration_id)
    WHERE is_active = true AND device_registration_id IS NOT NULL;
