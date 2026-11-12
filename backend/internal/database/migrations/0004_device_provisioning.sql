-- Device provisioning metadata.
-- device_uid is the stable external identifier; token_version lets key rotation
-- invalidate already-issued device tokens before their one-hour expiry.

ALTER TABLE public.device_registrations
    ADD COLUMN IF NOT EXISTS device_uid character varying,
    ADD COLUMN IF NOT EXISTS token_version integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS revoked_at timestamp(6) without time zone;

UPDATE public.device_registrations
SET device_uid = 'legacy-' || id
WHERE device_uid IS NULL;

ALTER TABLE public.device_registrations
    ALTER COLUMN device_uid SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS index_device_registrations_on_device_uid
    ON public.device_registrations USING btree (device_uid);
