CREATE TABLE IF NOT EXISTS public.extension_requests (
    id bigint NOT NULL,
    borrow_record_id bigint NOT NULL,
    member_id bigint NOT NULL,
    requested_return_date date NOT NULL,
    reason text NOT NULL,
    status character varying DEFAULT 'pending'::character varying NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.extension_requests_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.extension_requests_id_seq OWNED BY public.extension_requests.id;

ALTER TABLE ONLY public.extension_requests ALTER COLUMN id SET DEFAULT nextval('public.extension_requests_id_seq'::regclass);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'extension_requests_pkey') THEN
        ALTER TABLE ONLY public.extension_requests ADD CONSTRAINT extension_requests_pkey PRIMARY KEY (id);
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS index_extension_requests_on_borrow_record_id ON public.extension_requests USING btree (borrow_record_id);
CREATE INDEX IF NOT EXISTS index_extension_requests_on_member_id ON public.extension_requests USING btree (member_id);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_rails_extension_requests_borrow_record') THEN
        ALTER TABLE ONLY public.extension_requests
            ADD CONSTRAINT fk_rails_extension_requests_borrow_record FOREIGN KEY (borrow_record_id) REFERENCES public.borrow_records(id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_rails_extension_requests_member') THEN
        ALTER TABLE ONLY public.extension_requests
            ADD CONSTRAINT fk_rails_extension_requests_member FOREIGN KEY (member_id) REFERENCES public.members(id);
    END IF;
END $$;
