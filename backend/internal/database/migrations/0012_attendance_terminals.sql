-- Dedicated fingerprint attendance terminals.
--
-- Terminal slot mappings live in biometric_enrollments, exactly like the
-- existing door readers. These tables only track the enrollment command
-- lifecycle and terminal health. No fingerprint templates are stored here.

CREATE TABLE IF NOT EXISTS public.attendance_enrollment_sessions (
    id bigserial PRIMARY KEY,
    session_uid character varying(64) NOT NULL,
    member_id bigint NOT NULL,
    device_registration_id bigint NOT NULL,
    requested_by_id bigint NOT NULL DEFAULT 0,
    operation character varying(24) NOT NULL DEFAULT 'enroll',
    status character varying(32) NOT NULL DEFAULT 'pending',
    reader_key character varying(32),
    fingerprint_slot integer,
    device_user_id character varying,
    attempts integer NOT NULL DEFAULT 0,
    error_code character varying,
    error_message text,
    cleanup_required boolean NOT NULL DEFAULT false,
    cleanup_completed_at timestamp(6) without time zone,
    is_active boolean NOT NULL DEFAULT true,
    claimed_at timestamp(6) without time zone,
    claimed_by_device_id bigint,
    expires_at timestamp(6) without time zone,
    started_at timestamp(6) without time zone,
    completed_at timestamp(6) without time zone,
    created_at timestamp(6) without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp(6) without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS index_attendance_enrollment_sessions_on_session_uid
    ON public.attendance_enrollment_sessions USING btree (session_uid);

CREATE INDEX IF NOT EXISTS index_attendance_enrollment_sessions_on_member_id
    ON public.attendance_enrollment_sessions USING btree (member_id);

CREATE INDEX IF NOT EXISTS index_attendance_enrollment_sessions_on_device_status
    ON public.attendance_enrollment_sessions USING btree (device_registration_id, status);

CREATE UNIQUE INDEX IF NOT EXISTS index_attendance_enrollment_sessions_active_device
    ON public.attendance_enrollment_sessions USING btree (device_registration_id)
    WHERE is_active = true;

CREATE UNIQUE INDEX IF NOT EXISTS index_attendance_enrollment_sessions_pending_cleanup
    ON public.attendance_enrollment_sessions USING btree (device_registration_id)
    WHERE cleanup_required = true AND cleanup_completed_at IS NULL;

CREATE TABLE IF NOT EXISTS public.attendance_terminal_status (
    id bigserial PRIMARY KEY,
    device_registration_id bigint NOT NULL,
    firmware_version character varying(64),
    reader_model character varying(64),
    reader_firmware character varying(64),
    reader_connected boolean NOT NULL DEFAULT false,
    reader_readiness character varying(32) NOT NULL DEFAULT 'not_ready',
    queue_depth integer NOT NULL DEFAULT 0,
    network_status character varying(32),
    last_sync_at timestamp(6) without time zone,
    last_capture_at timestamp(6) without time zone,
    reported_at timestamp(6) without time zone,
    created_at timestamp(6) without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp(6) without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS index_attendance_terminal_status_on_device
    ON public.attendance_terminal_status USING btree (device_registration_id);

CREATE INDEX IF NOT EXISTS index_device_events_on_event_type_id
    ON public.device_events USING btree (event_type, id);
