CREATE TABLE IF NOT EXISTS public.member_registration_emails (
    id bigserial PRIMARY KEY,
    member_id bigint NOT NULL,
    recipient character varying NOT NULL,
    status character varying NOT NULL DEFAULT 'pending',
    attempts integer NOT NULL DEFAULT 0,
    last_error text,
    queued_at timestamp(6) without time zone,
    sent_at timestamp(6) without time zone,
    created_at timestamp(6) without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp(6) without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS index_member_registration_emails_on_member_id
    ON public.member_registration_emails USING btree (member_id);

CREATE INDEX IF NOT EXISTS index_member_registration_emails_on_status
    ON public.member_registration_emails USING btree (status);
