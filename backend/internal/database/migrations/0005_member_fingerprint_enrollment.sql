-- Two-step member registration: member details remain pending until the R307S
-- enrollment session has captured the same finger twice and the backend has
-- finalized the member/device/slot mapping.
--
-- Only the slot mapping is stored here. Fingerprint templates never leave the
-- R307S sensor and are never sent to or stored by the backend.

ALTER TABLE public.biometric_enrollments
    ADD COLUMN IF NOT EXISTS fingerprint_slot integer;

CREATE UNIQUE INDEX IF NOT EXISTS index_biometric_enrollments_on_device_slot
    ON public.biometric_enrollments USING btree (device_registration_id, fingerprint_slot)
    WHERE fingerprint_slot IS NOT NULL;

CREATE TABLE IF NOT EXISTS public.member_enrollment_sessions (
    id bigserial PRIMARY KEY,
    session_uid character varying(64) NOT NULL,
    member_id bigint NOT NULL,
    requested_by_id bigint NOT NULL DEFAULT 0,
    device_registration_id bigint,
    status character varying NOT NULL DEFAULT 'pending_details',
    fingerprint_slot integer,
    device_user_id character varying,
    attempts integer NOT NULL DEFAULT 0,
    error_code character varying,
    error_message text,
    cleanup_required boolean NOT NULL DEFAULT false,
    is_active boolean NOT NULL DEFAULT true,
    expires_at timestamp(6) without time zone,
    started_at timestamp(6) without time zone,
    completed_at timestamp(6) without time zone,
    cleanup_completed_at timestamp(6) without time zone,
    created_at timestamp(6) without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp(6) without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS index_member_enrollment_sessions_on_session_uid
    ON public.member_enrollment_sessions USING btree (session_uid);

CREATE UNIQUE INDEX IF NOT EXISTS index_member_enrollment_sessions_active_member
    ON public.member_enrollment_sessions USING btree (member_id)
    WHERE is_active = true;

CREATE INDEX IF NOT EXISTS index_member_enrollment_sessions_on_member_id
    ON public.member_enrollment_sessions USING btree (member_id);

CREATE INDEX IF NOT EXISTS index_member_enrollment_sessions_on_status
    ON public.member_enrollment_sessions USING btree (status);

CREATE INDEX IF NOT EXISTS index_member_enrollment_sessions_device_active
    ON public.member_enrollment_sessions USING btree (device_registration_id, is_active);

CREATE INDEX IF NOT EXISTS index_member_enrollment_sessions_device_cleanup
    ON public.member_enrollment_sessions USING btree (device_registration_id, cleanup_required)
    WHERE cleanup_required = true;
