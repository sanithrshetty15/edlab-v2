-- Add the readiness column to device_readers when it is missing.
-- The ESP32 reports its last confirmed sensor readiness (ready, waiting_for_finger,
-- finger_detected, capture_failed, enrolling, ...) alongside connectivity.
-- Idempotent; existing records keep the 'not_ready' default.

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'device_readers'
          AND column_name = 'readiness'
    ) THEN
        ALTER TABLE public.device_readers
            ADD COLUMN readiness character varying(32) NOT NULL DEFAULT 'not_ready';
    END IF;
END $$;
