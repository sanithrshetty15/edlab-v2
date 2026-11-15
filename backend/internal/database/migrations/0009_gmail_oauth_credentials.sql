CREATE TABLE IF NOT EXISTS public.gmail_oauth_credentials (
    id bigint PRIMARY KEY CHECK (id = 1),
    encrypted_refresh_token text NOT NULL,
    scopes text NOT NULL DEFAULT '',
    token_type character varying NOT NULL DEFAULT 'Bearer',
    authorized_at timestamp(6) without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at timestamp(6) without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp(6) without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP
);
