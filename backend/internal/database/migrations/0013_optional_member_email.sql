DROP INDEX IF EXISTS public.index_users_on_email;

CREATE UNIQUE INDEX IF NOT EXISTS index_users_on_email
    ON public.users USING btree (email)
    WHERE btrim(email) <> '';
