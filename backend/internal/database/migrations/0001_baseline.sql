--
-- PostgreSQL database dump
--


-- Dumped from database version 18.6
-- Dumped by pg_dump version 18.6

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: public; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA IF NOT EXISTS public;


--
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON SCHEMA public IS 'standard public schema';


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: academic_years; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.academic_years (
    id bigint NOT NULL,
    year integer NOT NULL,
    label character varying NOT NULL,
    is_current boolean DEFAULT false NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: academic_years_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.academic_years_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: academic_years_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.academic_years_id_seq OWNED BY public.academic_years.id;


--
-- Name: admin_transfers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.admin_transfers (
    id bigint NOT NULL,
    from_admin_id bigint NOT NULL,
    to_admin_id bigint NOT NULL,
    status character varying DEFAULT 'pending'::character varying NOT NULL,
    initiated_at timestamp(6) without time zone NOT NULL,
    completed_at timestamp(6) without time zone,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: admin_transfers_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.admin_transfers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: admin_transfers_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.admin_transfers_id_seq OWNED BY public.admin_transfers.id;


--
-- Name: admins; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.admins (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    name character varying NOT NULL,
    is_primary boolean DEFAULT false NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: admins_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.admins_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: admins_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.admins_id_seq OWNED BY public.admins.id;


--
-- Name: announcement_reads; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.announcement_reads (
    id bigint NOT NULL,
    announcement_id bigint NOT NULL,
    user_id bigint NOT NULL,
    read_at timestamp(6) without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: announcement_reads_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.announcement_reads_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: announcement_reads_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.announcement_reads_id_seq OWNED BY public.announcement_reads.id;


--
-- Name: announcements; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.announcements (
    id bigint NOT NULL,
    title character varying NOT NULL,
    content text NOT NULL,
    is_pinned boolean DEFAULT false NOT NULL,
    published_at timestamp(6) without time zone,
    scheduled_at timestamp(6) without time zone,
    created_by_id bigint NOT NULL,
    deleted_at timestamp(6) without time zone,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: announcements_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.announcements_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: announcements_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.announcements_id_seq OWNED BY public.announcements.id;


--
-- Name: ar_internal_metadata; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ar_internal_metadata (
    key character varying NOT NULL,
    value character varying,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: attendance_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.attendance_logs (
    id bigint NOT NULL,
    member_id bigint NOT NULL,
    entry_date date NOT NULL,
    entry_time timestamp(6) without time zone NOT NULL,
    exit_time timestamp(6) without time zone,
    source character varying DEFAULT 'manual'::character varying NOT NULL,
    device_reference character varying,
    created_by_id bigint,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: attendance_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.attendance_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: attendance_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.attendance_logs_id_seq OWNED BY public.attendance_logs.id;


--
-- Name: audit_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.audit_logs (
    id bigint NOT NULL,
    user_id bigint,
    action character varying NOT NULL,
    resource_type character varying NOT NULL,
    resource_id bigint,
    details jsonb DEFAULT '{}'::jsonb,
    ip_address character varying,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: audit_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.audit_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: audit_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.audit_logs_id_seq OWNED BY public.audit_logs.id;


--
-- Name: biometric_enrollments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.biometric_enrollments (
    id bigint NOT NULL,
    member_id bigint NOT NULL,
    device_registration_id bigint NOT NULL,
    device_user_id character varying NOT NULL,
    status character varying DEFAULT 'pending'::character varying NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: biometric_enrollments_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.biometric_enrollments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: biometric_enrollments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.biometric_enrollments_id_seq OWNED BY public.biometric_enrollments.id;


--
-- Name: borrow_records; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.borrow_records (
    id bigint NOT NULL,
    member_id bigint NOT NULL,
    inventory_item_id bigint NOT NULL,
    status character varying DEFAULT 'borrowed'::character varying NOT NULL,
    borrowed_at timestamp(6) without time zone NOT NULL,
    due_date date NOT NULL,
    returned_at timestamp(6) without time zone,
    condition_on_return character varying,
    idempotency_key character varying,
    approved_by_id bigint,
    approval_reason character varying,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    quantity integer DEFAULT 1 NOT NULL
);


--
-- Name: borrow_records_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.borrow_records_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: borrow_records_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.borrow_records_id_seq OWNED BY public.borrow_records.id;


--
-- Name: contact_messages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.contact_messages (
    id bigint NOT NULL,
    member_id bigint NOT NULL,
    subject character varying NOT NULL,
    category character varying NOT NULL,
    message text NOT NULL,
    status character varying DEFAULT 'new'::character varying NOT NULL,
    email_delivery_status character varying,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: contact_messages_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.contact_messages_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: contact_messages_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.contact_messages_id_seq OWNED BY public.contact_messages.id;


--
-- Name: content_blocks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.content_blocks (
    id bigint NOT NULL,
    key character varying NOT NULL,
    title character varying,
    content text,
    "position" integer DEFAULT 0 NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: content_blocks_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.content_blocks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: content_blocks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.content_blocks_id_seq OWNED BY public.content_blocks.id;


--
-- Name: device_events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_events (
    id bigint NOT NULL,
    device_registration_id bigint NOT NULL,
    event_type character varying NOT NULL,
    idempotency_key character varying NOT NULL,
    member_identifier character varying,
    hmac_signature character varying,
    event_timestamp timestamp(6) without time zone NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb,
    processing_status character varying DEFAULT 'pending'::character varying NOT NULL,
    processing_error text,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: device_events_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.device_events_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: device_events_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.device_events_id_seq OWNED BY public.device_events.id;


--
-- Name: device_registrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_registrations (
    id bigint NOT NULL,
    name character varying NOT NULL,
    device_type character varying NOT NULL,
    api_key_digest character varying NOT NULL,
    status character varying DEFAULT 'active'::character varying NOT NULL,
    last_seen_at timestamp(6) without time zone,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: device_registrations_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.device_registrations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: device_registrations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.device_registrations_id_seq OWNED BY public.device_registrations.id;


--
-- Name: events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.events (
    id bigint NOT NULL,
    title character varying NOT NULL,
    description text,
    event_type character varying NOT NULL,
    date date NOT NULL,
    start_time time without time zone NOT NULL,
    end_time time without time zone NOT NULL,
    venue character varying,
    external_links text[] DEFAULT '{}'::text[],
    created_by_id bigint NOT NULL,
    deleted_at timestamp(6) without time zone,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: events_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.events_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: events_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.events_id_seq OWNED BY public.events.id;


--
-- Name: extension_requests; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.extension_requests (
    id bigint NOT NULL,
    borrow_record_id bigint NOT NULL,
    member_id bigint NOT NULL,
    requested_return_date date NOT NULL,
    reason text NOT NULL,
    status character varying DEFAULT 'pending'::character varying NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: extension_requests_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.extension_requests_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: extension_requests_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.extension_requests_id_seq OWNED BY public.extension_requests.id;


--
-- Name: faq_categories; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.faq_categories (
    id bigint NOT NULL,
    name character varying NOT NULL,
    slug character varying NOT NULL,
    "position" integer DEFAULT 0 NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: faq_categories_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.faq_categories_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: faq_categories_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.faq_categories_id_seq OWNED BY public.faq_categories.id;


--
-- Name: faqs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.faqs (
    id bigint NOT NULL,
    faq_category_id bigint NOT NULL,
    question character varying NOT NULL,
    answer text NOT NULL,
    "position" integer DEFAULT 0 NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: faqs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.faqs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: faqs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.faqs_id_seq OWNED BY public.faqs.id;


--
-- Name: group_messages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.group_messages (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    content text,
    reply_to_id bigint,
    forwarded_from_id bigint,
    is_edited boolean DEFAULT false NOT NULL,
    is_deleted boolean DEFAULT false NOT NULL,
    is_pinned boolean DEFAULT false NOT NULL,
    message_type character varying DEFAULT 'text'::character varying NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: group_messages_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.group_messages_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: group_messages_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.group_messages_id_seq OWNED BY public.group_messages.id;


--
-- Name: group_settings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.group_settings (
    id bigint NOT NULL,
    permission character varying DEFAULT 'everyone'::character varying NOT NULL,
    description character varying,
    media_sharing_enabled boolean DEFAULT true NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: group_settings_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.group_settings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: group_settings_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.group_settings_id_seq OWNED BY public.group_settings.id;


--
-- Name: inventory_imports; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.inventory_imports (
    id bigint NOT NULL,
    filename character varying NOT NULL,
    total_rows integer NOT NULL,
    imported_rows integer DEFAULT 0 NOT NULL,
    error_rows integer DEFAULT 0 NOT NULL,
    status character varying DEFAULT 'completed'::character varying NOT NULL,
    imported_by_id bigint NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: inventory_imports_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.inventory_imports_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: inventory_imports_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.inventory_imports_id_seq OWNED BY public.inventory_imports.id;


--
-- Name: inventory_items; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.inventory_items (
    id bigint NOT NULL,
    name character varying NOT NULL,
    category character varying NOT NULL,
    description text,
    total_quantity integer DEFAULT 0 NOT NULL,
    available_quantity integer DEFAULT 0 NOT NULL,
    storage_location character varying,
    condition character varying,
    is_active boolean DEFAULT true NOT NULL,
    deleted_at timestamp(6) without time zone,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    CONSTRAINT check_available_quantity CHECK (((available_quantity >= 0) AND (available_quantity <= total_quantity)))
);


--
-- Name: inventory_items_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.inventory_items_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: inventory_items_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.inventory_items_id_seq OWNED BY public.inventory_items.id;


--
-- Name: lab_rules_acceptances; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lab_rules_acceptances (
    id bigint NOT NULL,
    member_id bigint NOT NULL,
    lab_rules_version_id bigint NOT NULL,
    accepted_at timestamp(6) without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: lab_rules_acceptances_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.lab_rules_acceptances_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: lab_rules_acceptances_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.lab_rules_acceptances_id_seq OWNED BY public.lab_rules_acceptances.id;


--
-- Name: lab_rules_versions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lab_rules_versions (
    id bigint NOT NULL,
    version_number integer NOT NULL,
    content text NOT NULL,
    is_current boolean DEFAULT false NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: lab_rules_versions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.lab_rules_versions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: lab_rules_versions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.lab_rules_versions_id_seq OWNED BY public.lab_rules_versions.id;


--
-- Name: link_previews; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.link_previews (
    id bigint NOT NULL,
    group_message_id bigint NOT NULL,
    url character varying NOT NULL,
    title character varying,
    description text,
    image_url character varying,
    site_name character varying,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: link_previews_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.link_previews_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: link_previews_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.link_previews_id_seq OWNED BY public.link_previews.id;


--
-- Name: members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.members (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    name character varying NOT NULL,
    usn character varying,
    roll_number character varying,
    login_id character varying NOT NULL,
    academic_year integer NOT NULL,
    branch character varying NOT NULL,
    section character varying,
    email character varying NOT NULL,
    phone character varying NOT NULL,
    must_change_password boolean DEFAULT true NOT NULL,
    login_id_changed boolean DEFAULT false NOT NULL,
    login_id_changed_at timestamp(6) without time zone,
    status character varying DEFAULT 'active'::character varying NOT NULL,
    github_url character varying,
    linkedin_url character varying,
    portfolio_url character varying,
    bio text,
    interests text[] DEFAULT '{}'::text[],
    deleted_at timestamp(6) without time zone,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    phone_verified boolean DEFAULT false NOT NULL
);


--
-- Name: members_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.members_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: members_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.members_id_seq OWNED BY public.members.id;


--
-- Name: message_attachments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.message_attachments (
    id bigint NOT NULL,
    group_message_id bigint NOT NULL,
    filename character varying NOT NULL,
    content_type character varying NOT NULL,
    byte_size integer NOT NULL,
    storage_key character varying NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: message_attachments_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.message_attachments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: message_attachments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.message_attachments_id_seq OWNED BY public.message_attachments.id;


--
-- Name: message_pins; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.message_pins (
    id bigint NOT NULL,
    group_message_id bigint NOT NULL,
    pinned_by_id bigint NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: message_pins_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.message_pins_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: message_pins_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.message_pins_id_seq OWNED BY public.message_pins.id;


--
-- Name: message_reactions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.message_reactions (
    id bigint NOT NULL,
    group_message_id bigint NOT NULL,
    user_id bigint NOT NULL,
    emoji character varying NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: message_reactions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.message_reactions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: message_reactions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.message_reactions_id_seq OWNED BY public.message_reactions.id;


--
-- Name: message_receipts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.message_receipts (
    id bigint NOT NULL,
    group_message_id bigint NOT NULL,
    user_id bigint NOT NULL,
    delivered_at timestamp(6) without time zone,
    read_at timestamp(6) without time zone,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: message_receipts_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.message_receipts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: message_receipts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.message_receipts_id_seq OWNED BY public.message_receipts.id;


--
-- Name: message_stars; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.message_stars (
    id bigint NOT NULL,
    group_message_id bigint NOT NULL,
    user_id bigint NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: message_stars_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.message_stars_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: message_stars_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.message_stars_id_seq OWNED BY public.message_stars.id;


--
-- Name: notifications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notifications (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    title character varying NOT NULL,
    body text,
    notification_type character varying NOT NULL,
    subject_type character varying,
    subject_id bigint,
    is_read boolean DEFAULT false NOT NULL,
    read_at timestamp(6) without time zone,
    email_status character varying,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: notifications_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.notifications_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: notifications_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.notifications_id_seq OWNED BY public.notifications.id;


--
-- Name: otp_codes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.otp_codes (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    otp_digest character varying NOT NULL,
    delivery_method character varying NOT NULL,
    purpose character varying NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    used boolean DEFAULT false NOT NULL,
    expires_at timestamp(6) without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: otp_codes_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.otp_codes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: otp_codes_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.otp_codes_id_seq OWNED BY public.otp_codes.id;


--
-- Name: platform_settings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.platform_settings (
    id bigint NOT NULL,
    key character varying NOT NULL,
    value text,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: platform_settings_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.platform_settings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: platform_settings_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.platform_settings_id_seq OWNED BY public.platform_settings.id;


--
-- Name: poll_options; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.poll_options (
    id bigint NOT NULL,
    poll_id bigint NOT NULL,
    text character varying NOT NULL,
    "position" integer DEFAULT 0 NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: poll_options_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.poll_options_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: poll_options_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.poll_options_id_seq OWNED BY public.poll_options.id;


--
-- Name: poll_votes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.poll_votes (
    id bigint NOT NULL,
    poll_id bigint NOT NULL,
    poll_option_id bigint NOT NULL,
    user_id bigint NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: poll_votes_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.poll_votes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: poll_votes_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.poll_votes_id_seq OWNED BY public.poll_votes.id;


--
-- Name: polls; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.polls (
    id bigint NOT NULL,
    group_message_id bigint NOT NULL,
    question character varying NOT NULL,
    is_multiple_choice boolean DEFAULT false NOT NULL,
    closes_at timestamp(6) without time zone,
    is_closed boolean DEFAULT false NOT NULL,
    created_by_id bigint NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: polls_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.polls_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: polls_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.polls_id_seq OWNED BY public.polls.id;


--
-- Name: return_requests; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.return_requests (
    id bigint NOT NULL,
    borrow_record_id bigint NOT NULL,
    member_id bigint NOT NULL,
    status character varying DEFAULT 'pending'::character varying NOT NULL,
    confirmed_at timestamp(6) without time zone,
    confirmed_by_id bigint,
    condition_note character varying,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: return_requests_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.return_requests_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: return_requests_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.return_requests_id_seq OWNED BY public.return_requests.id;


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version character varying NOT NULL
);


--
-- Name: semesters; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.semesters (
    id bigint NOT NULL,
    name character varying NOT NULL,
    start_date date NOT NULL,
    end_date date NOT NULL,
    is_current boolean DEFAULT false NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: semesters_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.semesters_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: semesters_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.semesters_id_seq OWNED BY public.semesters.id;


--
-- Name: sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sessions (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    token_digest character varying NOT NULL,
    ip_address character varying,
    user_agent character varying,
    last_active_at timestamp(6) without time zone NOT NULL,
    expires_at timestamp(6) without time zone,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: sessions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.sessions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: sessions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.sessions_id_seq OWNED BY public.sessions.id;


--
-- Name: task_submission_statuses; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_submission_statuses (
    id bigint NOT NULL,
    task_id bigint NOT NULL,
    member_id bigint NOT NULL,
    status character varying DEFAULT 'pending'::character varying NOT NULL,
    submitted_at timestamp(6) without time zone,
    verified_at timestamp(6) without time zone,
    verified_by_id bigint,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: task_submission_statuses_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.task_submission_statuses_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: task_submission_statuses_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.task_submission_statuses_id_seq OWNED BY public.task_submission_statuses.id;


--
-- Name: task_targets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_targets (
    id bigint NOT NULL,
    task_id bigint NOT NULL,
    academic_year integer NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: task_targets_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.task_targets_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: task_targets_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.task_targets_id_seq OWNED BY public.task_targets.id;


--
-- Name: tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tasks (
    id bigint NOT NULL,
    title character varying NOT NULL,
    description text,
    task_type character varying NOT NULL,
    published_at timestamp(6) without time zone,
    deadline timestamp(6) without time zone NOT NULL,
    instructions text,
    external_link character varying,
    required_attachments character varying,
    created_by_id bigint NOT NULL,
    deleted_at timestamp(6) without time zone,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: tasks_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.tasks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: tasks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.tasks_id_seq OWNED BY public.tasks.id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id bigint NOT NULL,
    email character varying NOT NULL,
    phone character varying,
    password_digest character varying,
    role character varying DEFAULT 'member'::character varying NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: academic_years id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_years ALTER COLUMN id SET DEFAULT nextval('public.academic_years_id_seq'::regclass);


--
-- Name: admin_transfers id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admin_transfers ALTER COLUMN id SET DEFAULT nextval('public.admin_transfers_id_seq'::regclass);


--
-- Name: admins id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admins ALTER COLUMN id SET DEFAULT nextval('public.admins_id_seq'::regclass);


--
-- Name: announcement_reads id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.announcement_reads ALTER COLUMN id SET DEFAULT nextval('public.announcement_reads_id_seq'::regclass);


--
-- Name: announcements id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.announcements ALTER COLUMN id SET DEFAULT nextval('public.announcements_id_seq'::regclass);


--
-- Name: attendance_logs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.attendance_logs ALTER COLUMN id SET DEFAULT nextval('public.attendance_logs_id_seq'::regclass);


--
-- Name: audit_logs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_logs ALTER COLUMN id SET DEFAULT nextval('public.audit_logs_id_seq'::regclass);


--
-- Name: biometric_enrollments id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.biometric_enrollments ALTER COLUMN id SET DEFAULT nextval('public.biometric_enrollments_id_seq'::regclass);


--
-- Name: borrow_records id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.borrow_records ALTER COLUMN id SET DEFAULT nextval('public.borrow_records_id_seq'::regclass);


--
-- Name: contact_messages id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.contact_messages ALTER COLUMN id SET DEFAULT nextval('public.contact_messages_id_seq'::regclass);


--
-- Name: content_blocks id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.content_blocks ALTER COLUMN id SET DEFAULT nextval('public.content_blocks_id_seq'::regclass);


--
-- Name: device_events id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_events ALTER COLUMN id SET DEFAULT nextval('public.device_events_id_seq'::regclass);


--
-- Name: device_registrations id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_registrations ALTER COLUMN id SET DEFAULT nextval('public.device_registrations_id_seq'::regclass);


--
-- Name: events id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.events ALTER COLUMN id SET DEFAULT nextval('public.events_id_seq'::regclass);


--
-- Name: extension_requests id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.extension_requests ALTER COLUMN id SET DEFAULT nextval('public.extension_requests_id_seq'::regclass);


--
-- Name: faq_categories id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.faq_categories ALTER COLUMN id SET DEFAULT nextval('public.faq_categories_id_seq'::regclass);


--
-- Name: faqs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.faqs ALTER COLUMN id SET DEFAULT nextval('public.faqs_id_seq'::regclass);


--
-- Name: group_messages id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_messages ALTER COLUMN id SET DEFAULT nextval('public.group_messages_id_seq'::regclass);


--
-- Name: group_settings id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_settings ALTER COLUMN id SET DEFAULT nextval('public.group_settings_id_seq'::regclass);


--
-- Name: inventory_imports id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inventory_imports ALTER COLUMN id SET DEFAULT nextval('public.inventory_imports_id_seq'::regclass);


--
-- Name: inventory_items id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inventory_items ALTER COLUMN id SET DEFAULT nextval('public.inventory_items_id_seq'::regclass);


--
-- Name: lab_rules_acceptances id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lab_rules_acceptances ALTER COLUMN id SET DEFAULT nextval('public.lab_rules_acceptances_id_seq'::regclass);


--
-- Name: lab_rules_versions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lab_rules_versions ALTER COLUMN id SET DEFAULT nextval('public.lab_rules_versions_id_seq'::regclass);


--
-- Name: link_previews id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.link_previews ALTER COLUMN id SET DEFAULT nextval('public.link_previews_id_seq'::regclass);


--
-- Name: members id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.members ALTER COLUMN id SET DEFAULT nextval('public.members_id_seq'::regclass);


--
-- Name: message_attachments id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_attachments ALTER COLUMN id SET DEFAULT nextval('public.message_attachments_id_seq'::regclass);


--
-- Name: message_pins id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_pins ALTER COLUMN id SET DEFAULT nextval('public.message_pins_id_seq'::regclass);


--
-- Name: message_reactions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_reactions ALTER COLUMN id SET DEFAULT nextval('public.message_reactions_id_seq'::regclass);


--
-- Name: message_receipts id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_receipts ALTER COLUMN id SET DEFAULT nextval('public.message_receipts_id_seq'::regclass);


--
-- Name: message_stars id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_stars ALTER COLUMN id SET DEFAULT nextval('public.message_stars_id_seq'::regclass);


--
-- Name: notifications id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications ALTER COLUMN id SET DEFAULT nextval('public.notifications_id_seq'::regclass);


--
-- Name: otp_codes id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.otp_codes ALTER COLUMN id SET DEFAULT nextval('public.otp_codes_id_seq'::regclass);


--
-- Name: platform_settings id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.platform_settings ALTER COLUMN id SET DEFAULT nextval('public.platform_settings_id_seq'::regclass);


--
-- Name: poll_options id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.poll_options ALTER COLUMN id SET DEFAULT nextval('public.poll_options_id_seq'::regclass);


--
-- Name: poll_votes id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.poll_votes ALTER COLUMN id SET DEFAULT nextval('public.poll_votes_id_seq'::regclass);


--
-- Name: polls id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.polls ALTER COLUMN id SET DEFAULT nextval('public.polls_id_seq'::regclass);


--
-- Name: return_requests id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.return_requests ALTER COLUMN id SET DEFAULT nextval('public.return_requests_id_seq'::regclass);


--
-- Name: semesters id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.semesters ALTER COLUMN id SET DEFAULT nextval('public.semesters_id_seq'::regclass);


--
-- Name: sessions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sessions ALTER COLUMN id SET DEFAULT nextval('public.sessions_id_seq'::regclass);


--
-- Name: task_submission_statuses id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_submission_statuses ALTER COLUMN id SET DEFAULT nextval('public.task_submission_statuses_id_seq'::regclass);


--
-- Name: task_targets id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_targets ALTER COLUMN id SET DEFAULT nextval('public.task_targets_id_seq'::regclass);


--
-- Name: tasks id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks ALTER COLUMN id SET DEFAULT nextval('public.tasks_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Name: academic_years academic_years_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.academic_years
    ADD CONSTRAINT academic_years_pkey PRIMARY KEY (id);


--
-- Name: admin_transfers admin_transfers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admin_transfers
    ADD CONSTRAINT admin_transfers_pkey PRIMARY KEY (id);


--
-- Name: admins admins_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admins
    ADD CONSTRAINT admins_pkey PRIMARY KEY (id);


--
-- Name: announcement_reads announcement_reads_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.announcement_reads
    ADD CONSTRAINT announcement_reads_pkey PRIMARY KEY (id);


--
-- Name: announcements announcements_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.announcements
    ADD CONSTRAINT announcements_pkey PRIMARY KEY (id);


--
-- Name: ar_internal_metadata ar_internal_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ar_internal_metadata
    ADD CONSTRAINT ar_internal_metadata_pkey PRIMARY KEY (key);


--
-- Name: attendance_logs attendance_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.attendance_logs
    ADD CONSTRAINT attendance_logs_pkey PRIMARY KEY (id);


--
-- Name: audit_logs audit_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_logs
    ADD CONSTRAINT audit_logs_pkey PRIMARY KEY (id);


--
-- Name: biometric_enrollments biometric_enrollments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.biometric_enrollments
    ADD CONSTRAINT biometric_enrollments_pkey PRIMARY KEY (id);


--
-- Name: borrow_records borrow_records_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.borrow_records
    ADD CONSTRAINT borrow_records_pkey PRIMARY KEY (id);


--
-- Name: contact_messages contact_messages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.contact_messages
    ADD CONSTRAINT contact_messages_pkey PRIMARY KEY (id);


--
-- Name: content_blocks content_blocks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.content_blocks
    ADD CONSTRAINT content_blocks_pkey PRIMARY KEY (id);


--
-- Name: device_events device_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_events
    ADD CONSTRAINT device_events_pkey PRIMARY KEY (id);


--
-- Name: device_registrations device_registrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_registrations
    ADD CONSTRAINT device_registrations_pkey PRIMARY KEY (id);


--
-- Name: events events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.events
    ADD CONSTRAINT events_pkey PRIMARY KEY (id);


--
-- Name: extension_requests extension_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.extension_requests
    ADD CONSTRAINT extension_requests_pkey PRIMARY KEY (id);


--
-- Name: faq_categories faq_categories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.faq_categories
    ADD CONSTRAINT faq_categories_pkey PRIMARY KEY (id);


--
-- Name: faqs faqs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.faqs
    ADD CONSTRAINT faqs_pkey PRIMARY KEY (id);


--
-- Name: group_messages group_messages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_messages
    ADD CONSTRAINT group_messages_pkey PRIMARY KEY (id);


--
-- Name: group_settings group_settings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_settings
    ADD CONSTRAINT group_settings_pkey PRIMARY KEY (id);


--
-- Name: inventory_imports inventory_imports_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inventory_imports
    ADD CONSTRAINT inventory_imports_pkey PRIMARY KEY (id);


--
-- Name: inventory_items inventory_items_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inventory_items
    ADD CONSTRAINT inventory_items_pkey PRIMARY KEY (id);


--
-- Name: lab_rules_acceptances lab_rules_acceptances_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lab_rules_acceptances
    ADD CONSTRAINT lab_rules_acceptances_pkey PRIMARY KEY (id);


--
-- Name: lab_rules_versions lab_rules_versions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lab_rules_versions
    ADD CONSTRAINT lab_rules_versions_pkey PRIMARY KEY (id);


--
-- Name: link_previews link_previews_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.link_previews
    ADD CONSTRAINT link_previews_pkey PRIMARY KEY (id);


--
-- Name: members members_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.members
    ADD CONSTRAINT members_pkey PRIMARY KEY (id);


--
-- Name: message_attachments message_attachments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_attachments
    ADD CONSTRAINT message_attachments_pkey PRIMARY KEY (id);


--
-- Name: message_pins message_pins_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_pins
    ADD CONSTRAINT message_pins_pkey PRIMARY KEY (id);


--
-- Name: message_reactions message_reactions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_reactions
    ADD CONSTRAINT message_reactions_pkey PRIMARY KEY (id);


--
-- Name: message_receipts message_receipts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_receipts
    ADD CONSTRAINT message_receipts_pkey PRIMARY KEY (id);


--
-- Name: message_stars message_stars_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_stars
    ADD CONSTRAINT message_stars_pkey PRIMARY KEY (id);


--
-- Name: notifications notifications_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_pkey PRIMARY KEY (id);


--
-- Name: otp_codes otp_codes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.otp_codes
    ADD CONSTRAINT otp_codes_pkey PRIMARY KEY (id);


--
-- Name: platform_settings platform_settings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.platform_settings
    ADD CONSTRAINT platform_settings_pkey PRIMARY KEY (id);


--
-- Name: poll_options poll_options_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.poll_options
    ADD CONSTRAINT poll_options_pkey PRIMARY KEY (id);


--
-- Name: poll_votes poll_votes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.poll_votes
    ADD CONSTRAINT poll_votes_pkey PRIMARY KEY (id);


--
-- Name: polls polls_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.polls
    ADD CONSTRAINT polls_pkey PRIMARY KEY (id);


--
-- Name: return_requests return_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.return_requests
    ADD CONSTRAINT return_requests_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: semesters semesters_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.semesters
    ADD CONSTRAINT semesters_pkey PRIMARY KEY (id);


--
-- Name: sessions sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_pkey PRIMARY KEY (id);


--
-- Name: task_submission_statuses task_submission_statuses_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_submission_statuses
    ADD CONSTRAINT task_submission_statuses_pkey PRIMARY KEY (id);


--
-- Name: task_targets task_targets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_targets
    ADD CONSTRAINT task_targets_pkey PRIMARY KEY (id);


--
-- Name: tasks tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT tasks_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: idx_biometric_enrollment_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_biometric_enrollment_unique ON public.biometric_enrollments USING btree (member_id, device_registration_id);


--
-- Name: idx_notification_dedup; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notification_dedup ON public.notifications USING btree (user_id, notification_type, subject_id);


--
-- Name: idx_poll_vote_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_poll_vote_unique ON public.poll_votes USING btree (poll_id, user_id, poll_option_id);


--
-- Name: idx_reaction_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_reaction_unique ON public.message_reactions USING btree (group_message_id, user_id, emoji);


--
-- Name: idx_receipt_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_receipt_unique ON public.message_receipts USING btree (group_message_id, user_id);


--
-- Name: idx_rules_acceptance_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_rules_acceptance_unique ON public.lab_rules_acceptances USING btree (member_id, lab_rules_version_id);


--
-- Name: idx_task_submission_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_task_submission_unique ON public.task_submission_statuses USING btree (task_id, member_id);


--
-- Name: index_academic_years_on_year; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_academic_years_on_year ON public.academic_years USING btree (year);


--
-- Name: index_admin_transfers_on_from_admin_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_admin_transfers_on_from_admin_id ON public.admin_transfers USING btree (from_admin_id);


--
-- Name: index_admin_transfers_on_to_admin_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_admin_transfers_on_to_admin_id ON public.admin_transfers USING btree (to_admin_id);


--
-- Name: index_admins_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_admins_on_user_id ON public.admins USING btree (user_id);


--
-- Name: index_announcement_reads_on_announcement_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_announcement_reads_on_announcement_id ON public.announcement_reads USING btree (announcement_id);


--
-- Name: index_announcement_reads_on_announcement_id_and_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_announcement_reads_on_announcement_id_and_user_id ON public.announcement_reads USING btree (announcement_id, user_id);


--
-- Name: index_announcement_reads_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_announcement_reads_on_user_id ON public.announcement_reads USING btree (user_id);


--
-- Name: index_announcements_on_created_by_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_announcements_on_created_by_id ON public.announcements USING btree (created_by_id);


--
-- Name: index_announcements_on_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_announcements_on_deleted_at ON public.announcements USING btree (deleted_at);


--
-- Name: index_announcements_on_published_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_announcements_on_published_at ON public.announcements USING btree (published_at);


--
-- Name: index_attendance_logs_on_created_by_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_attendance_logs_on_created_by_id ON public.attendance_logs USING btree (created_by_id);


--
-- Name: index_attendance_logs_on_entry_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_attendance_logs_on_entry_date ON public.attendance_logs USING btree (entry_date);


--
-- Name: index_attendance_logs_on_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_attendance_logs_on_member_id ON public.attendance_logs USING btree (member_id);


--
-- Name: index_attendance_logs_on_member_id_and_entry_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_attendance_logs_on_member_id_and_entry_date ON public.attendance_logs USING btree (member_id, entry_date);


--
-- Name: index_audit_logs_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_audit_logs_on_created_at ON public.audit_logs USING btree (created_at);


--
-- Name: index_audit_logs_on_resource_type_and_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_audit_logs_on_resource_type_and_resource_id ON public.audit_logs USING btree (resource_type, resource_id);


--
-- Name: index_audit_logs_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_audit_logs_on_user_id ON public.audit_logs USING btree (user_id);


--
-- Name: index_biometric_enrollments_on_device_registration_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_biometric_enrollments_on_device_registration_id ON public.biometric_enrollments USING btree (device_registration_id);


--
-- Name: index_biometric_enrollments_on_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_biometric_enrollments_on_member_id ON public.biometric_enrollments USING btree (member_id);


--
-- Name: index_borrow_records_on_approved_by_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_borrow_records_on_approved_by_id ON public.borrow_records USING btree (approved_by_id);


--
-- Name: index_borrow_records_on_idempotency_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_borrow_records_on_idempotency_key ON public.borrow_records USING btree (idempotency_key) WHERE (idempotency_key IS NOT NULL);


--
-- Name: index_borrow_records_on_inventory_item_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_borrow_records_on_inventory_item_id ON public.borrow_records USING btree (inventory_item_id);


--
-- Name: index_borrow_records_on_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_borrow_records_on_member_id ON public.borrow_records USING btree (member_id);


--
-- Name: index_borrow_records_on_member_id_and_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_borrow_records_on_member_id_and_status ON public.borrow_records USING btree (member_id, status);


--
-- Name: index_borrow_records_on_status_and_due_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_borrow_records_on_status_and_due_date ON public.borrow_records USING btree (status, due_date);


--
-- Name: index_contact_messages_on_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_contact_messages_on_member_id ON public.contact_messages USING btree (member_id);


--
-- Name: index_contact_messages_on_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_contact_messages_on_status ON public.contact_messages USING btree (status);


--
-- Name: index_content_blocks_on_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_content_blocks_on_key ON public.content_blocks USING btree (key);


--
-- Name: index_device_events_on_device_registration_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_device_events_on_device_registration_id ON public.device_events USING btree (device_registration_id);


--
-- Name: index_device_events_on_idempotency_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_device_events_on_idempotency_key ON public.device_events USING btree (idempotency_key);


--
-- Name: index_device_events_on_processing_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_device_events_on_processing_status ON public.device_events USING btree (processing_status);


--
-- Name: index_device_registrations_on_api_key_digest; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_device_registrations_on_api_key_digest ON public.device_registrations USING btree (api_key_digest);


--
-- Name: index_events_on_created_by_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_events_on_created_by_id ON public.events USING btree (created_by_id);


--
-- Name: index_events_on_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_events_on_date ON public.events USING btree (date);


--
-- Name: index_events_on_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_events_on_deleted_at ON public.events USING btree (deleted_at);


--
-- Name: index_extension_requests_on_borrow_record_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_extension_requests_on_borrow_record_id ON public.extension_requests USING btree (borrow_record_id);


--
-- Name: index_extension_requests_on_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_extension_requests_on_member_id ON public.extension_requests USING btree (member_id);


--
-- Name: index_faq_categories_on_slug; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_faq_categories_on_slug ON public.faq_categories USING btree (slug);


--
-- Name: index_faqs_on_faq_category_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_faqs_on_faq_category_id ON public.faqs USING btree (faq_category_id);


--
-- Name: index_group_messages_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_group_messages_on_created_at ON public.group_messages USING btree (created_at);


--
-- Name: index_group_messages_on_forwarded_from_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_group_messages_on_forwarded_from_id ON public.group_messages USING btree (forwarded_from_id);


--
-- Name: index_group_messages_on_reply_to_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_group_messages_on_reply_to_id ON public.group_messages USING btree (reply_to_id);


--
-- Name: index_group_messages_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_group_messages_on_user_id ON public.group_messages USING btree (user_id);


--
-- Name: index_inventory_imports_on_imported_by_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_inventory_imports_on_imported_by_id ON public.inventory_imports USING btree (imported_by_id);


--
-- Name: index_inventory_items_on_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_inventory_items_on_deleted_at ON public.inventory_items USING btree (deleted_at);


--
-- Name: index_inventory_items_on_is_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_inventory_items_on_is_active ON public.inventory_items USING btree (is_active);


--
-- Name: index_inventory_items_on_name_and_category; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_inventory_items_on_name_and_category ON public.inventory_items USING btree (name, category);


--
-- Name: index_lab_rules_acceptances_on_lab_rules_version_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_lab_rules_acceptances_on_lab_rules_version_id ON public.lab_rules_acceptances USING btree (lab_rules_version_id);


--
-- Name: index_lab_rules_acceptances_on_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_lab_rules_acceptances_on_member_id ON public.lab_rules_acceptances USING btree (member_id);


--
-- Name: index_lab_rules_versions_on_version_number; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_lab_rules_versions_on_version_number ON public.lab_rules_versions USING btree (version_number);


--
-- Name: index_link_previews_on_group_message_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_link_previews_on_group_message_id ON public.link_previews USING btree (group_message_id);


--
-- Name: index_members_on_academic_year; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_members_on_academic_year ON public.members USING btree (academic_year);


--
-- Name: index_members_on_branch; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_members_on_branch ON public.members USING btree (branch);


--
-- Name: index_members_on_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_members_on_deleted_at ON public.members USING btree (deleted_at);


--
-- Name: index_members_on_login_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_members_on_login_id ON public.members USING btree (login_id);


--
-- Name: index_members_on_roll_number; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_members_on_roll_number ON public.members USING btree (roll_number) WHERE (roll_number IS NOT NULL);


--
-- Name: index_members_on_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_members_on_status ON public.members USING btree (status);


--
-- Name: index_members_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_members_on_user_id ON public.members USING btree (user_id);


--
-- Name: index_members_on_usn; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_members_on_usn ON public.members USING btree (usn) WHERE (usn IS NOT NULL);


--
-- Name: index_message_attachments_on_group_message_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_message_attachments_on_group_message_id ON public.message_attachments USING btree (group_message_id);


--
-- Name: index_message_pins_on_group_message_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_message_pins_on_group_message_id ON public.message_pins USING btree (group_message_id);


--
-- Name: index_message_pins_on_pinned_by_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_message_pins_on_pinned_by_id ON public.message_pins USING btree (pinned_by_id);


--
-- Name: index_message_reactions_on_group_message_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_message_reactions_on_group_message_id ON public.message_reactions USING btree (group_message_id);


--
-- Name: index_message_reactions_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_message_reactions_on_user_id ON public.message_reactions USING btree (user_id);


--
-- Name: index_message_receipts_on_group_message_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_message_receipts_on_group_message_id ON public.message_receipts USING btree (group_message_id);


--
-- Name: index_message_receipts_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_message_receipts_on_user_id ON public.message_receipts USING btree (user_id);


--
-- Name: index_message_receipts_on_user_id_and_read_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_message_receipts_on_user_id_and_read_at ON public.message_receipts USING btree (user_id, read_at);


--
-- Name: index_message_stars_on_group_message_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_message_stars_on_group_message_id ON public.message_stars USING btree (group_message_id);


--
-- Name: index_message_stars_on_group_message_id_and_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_message_stars_on_group_message_id_and_user_id ON public.message_stars USING btree (group_message_id, user_id);


--
-- Name: index_message_stars_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_message_stars_on_user_id ON public.message_stars USING btree (user_id);


--
-- Name: index_notifications_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_notifications_on_user_id ON public.notifications USING btree (user_id);


--
-- Name: index_notifications_on_user_id_and_is_read_and_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_notifications_on_user_id_and_is_read_and_created_at ON public.notifications USING btree (user_id, is_read, created_at);


--
-- Name: index_otp_codes_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_otp_codes_on_user_id ON public.otp_codes USING btree (user_id);


--
-- Name: index_otp_codes_on_user_id_and_purpose_and_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_otp_codes_on_user_id_and_purpose_and_created_at ON public.otp_codes USING btree (user_id, purpose, created_at);


--
-- Name: index_platform_settings_on_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_platform_settings_on_key ON public.platform_settings USING btree (key);


--
-- Name: index_poll_options_on_poll_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_poll_options_on_poll_id ON public.poll_options USING btree (poll_id);


--
-- Name: index_poll_votes_on_poll_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_poll_votes_on_poll_id ON public.poll_votes USING btree (poll_id);


--
-- Name: index_poll_votes_on_poll_option_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_poll_votes_on_poll_option_id ON public.poll_votes USING btree (poll_option_id);


--
-- Name: index_poll_votes_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_poll_votes_on_user_id ON public.poll_votes USING btree (user_id);


--
-- Name: index_polls_on_created_by_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_polls_on_created_by_id ON public.polls USING btree (created_by_id);


--
-- Name: index_polls_on_group_message_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_polls_on_group_message_id ON public.polls USING btree (group_message_id);


--
-- Name: index_return_requests_on_borrow_record_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_return_requests_on_borrow_record_id ON public.return_requests USING btree (borrow_record_id);


--
-- Name: index_return_requests_on_confirmed_by_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_return_requests_on_confirmed_by_id ON public.return_requests USING btree (confirmed_by_id);


--
-- Name: index_return_requests_on_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_return_requests_on_member_id ON public.return_requests USING btree (member_id);


--
-- Name: index_sessions_on_token_digest; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_sessions_on_token_digest ON public.sessions USING btree (token_digest);


--
-- Name: index_sessions_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_sessions_on_user_id ON public.sessions USING btree (user_id);


--
-- Name: index_sessions_on_user_id_and_last_active_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_sessions_on_user_id_and_last_active_at ON public.sessions USING btree (user_id, last_active_at);


--
-- Name: index_task_submission_statuses_on_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_task_submission_statuses_on_member_id ON public.task_submission_statuses USING btree (member_id);


--
-- Name: index_task_submission_statuses_on_task_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_task_submission_statuses_on_task_id ON public.task_submission_statuses USING btree (task_id);


--
-- Name: index_task_submission_statuses_on_verified_by_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_task_submission_statuses_on_verified_by_id ON public.task_submission_statuses USING btree (verified_by_id);


--
-- Name: index_task_targets_on_task_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_task_targets_on_task_id ON public.task_targets USING btree (task_id);


--
-- Name: index_task_targets_on_task_id_and_academic_year; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_task_targets_on_task_id_and_academic_year ON public.task_targets USING btree (task_id, academic_year);


--
-- Name: index_tasks_on_created_by_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_tasks_on_created_by_id ON public.tasks USING btree (created_by_id);


--
-- Name: index_tasks_on_deadline; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_tasks_on_deadline ON public.tasks USING btree (deadline);


--
-- Name: index_tasks_on_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_tasks_on_deleted_at ON public.tasks USING btree (deleted_at);


--
-- Name: index_users_on_email; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_users_on_email ON public.users USING btree (email);


--
-- Name: index_users_on_phone; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_users_on_phone ON public.users USING btree (phone) WHERE (phone IS NOT NULL);


--
-- Name: index_users_on_role; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_users_on_role ON public.users USING btree (role);


--
-- Name: borrow_records fk_rails_024c91ffde; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.borrow_records
    ADD CONSTRAINT fk_rails_024c91ffde FOREIGN KEY (approved_by_id) REFERENCES public.users(id);


--
-- Name: polls fk_rails_02adad54e8; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.polls
    ADD CONSTRAINT fk_rails_02adad54e8 FOREIGN KEY (group_message_id) REFERENCES public.group_messages(id);


--
-- Name: group_messages fk_rails_0a383d6560; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_messages
    ADD CONSTRAINT fk_rails_0a383d6560 FOREIGN KEY (forwarded_from_id) REFERENCES public.group_messages(id);


--
-- Name: task_targets fk_rails_188b42a8f3; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_targets
    ADD CONSTRAINT fk_rails_188b42a8f3 FOREIGN KEY (task_id) REFERENCES public.tasks(id);


--
-- Name: audit_logs fk_rails_1f26bc34ae; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_logs
    ADD CONSTRAINT fk_rails_1f26bc34ae FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: events fk_rails_1f2fddcdaa; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.events
    ADD CONSTRAINT fk_rails_1f2fddcdaa FOREIGN KEY (created_by_id) REFERENCES public.users(id);


--
-- Name: admin_transfers fk_rails_1f652f9a27; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admin_transfers
    ADD CONSTRAINT fk_rails_1f652f9a27 FOREIGN KEY (to_admin_id) REFERENCES public.admins(id);


--
-- Name: biometric_enrollments fk_rails_225c808d20; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.biometric_enrollments
    ADD CONSTRAINT fk_rails_225c808d20 FOREIGN KEY (device_registration_id) REFERENCES public.device_registrations(id);


--
-- Name: admin_transfers fk_rails_258fcf0bc9; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admin_transfers
    ADD CONSTRAINT fk_rails_258fcf0bc9 FOREIGN KEY (from_admin_id) REFERENCES public.admins(id);


--
-- Name: contact_messages fk_rails_28e8f25ee4; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.contact_messages
    ADD CONSTRAINT fk_rails_28e8f25ee4 FOREIGN KEY (member_id) REFERENCES public.members(id);


--
-- Name: message_pins fk_rails_29c76320e8; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_pins
    ADD CONSTRAINT fk_rails_29c76320e8 FOREIGN KEY (group_message_id) REFERENCES public.group_messages(id);


--
-- Name: message_pins fk_rails_2e2c1088e6; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_pins
    ADD CONSTRAINT fk_rails_2e2c1088e6 FOREIGN KEY (pinned_by_id) REFERENCES public.users(id);


--
-- Name: members fk_rails_2e88fb7ce9; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.members
    ADD CONSTRAINT fk_rails_2e88fb7ce9 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: biometric_enrollments fk_rails_3130abb224; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.biometric_enrollments
    ADD CONSTRAINT fk_rails_3130abb224 FOREIGN KEY (member_id) REFERENCES public.members(id);


--
-- Name: attendance_logs fk_rails_35102ba094; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.attendance_logs
    ADD CONSTRAINT fk_rails_35102ba094 FOREIGN KEY (member_id) REFERENCES public.members(id);


--
-- Name: admins fk_rails_378b9734e4; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admins
    ADD CONSTRAINT fk_rails_378b9734e4 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: task_submission_statuses fk_rails_389f300259; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_submission_statuses
    ADD CONSTRAINT fk_rails_389f300259 FOREIGN KEY (task_id) REFERENCES public.tasks(id);


--
-- Name: announcement_reads fk_rails_3bf795c88f; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.announcement_reads
    ADD CONSTRAINT fk_rails_3bf795c88f FOREIGN KEY (announcement_id) REFERENCES public.announcements(id);


--
-- Name: announcements fk_rails_3c036e8cea; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.announcements
    ADD CONSTRAINT fk_rails_3c036e8cea FOREIGN KEY (created_by_id) REFERENCES public.users(id);


--
-- Name: message_reactions fk_rails_3dfa2c0484; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_reactions
    ADD CONSTRAINT fk_rails_3dfa2c0484 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: otp_codes fk_rails_43b86b8e4d; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.otp_codes
    ADD CONSTRAINT fk_rails_43b86b8e4d FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: faqs fk_rails_474d4b16ab; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.faqs
    ADD CONSTRAINT fk_rails_474d4b16ab FOREIGN KEY (faq_category_id) REFERENCES public.faq_categories(id);


--
-- Name: message_stars fk_rails_4956a54b77; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_stars
    ADD CONSTRAINT fk_rails_4956a54b77 FOREIGN KEY (group_message_id) REFERENCES public.group_messages(id);


--
-- Name: task_submission_statuses fk_rails_4a83919add; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_submission_statuses
    ADD CONSTRAINT fk_rails_4a83919add FOREIGN KEY (verified_by_id) REFERENCES public.users(id);


--
-- Name: extension_requests fk_rails_4ea603bf85; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.extension_requests
    ADD CONSTRAINT fk_rails_4ea603bf85 FOREIGN KEY (borrow_record_id) REFERENCES public.borrow_records(id);


--
-- Name: message_attachments fk_rails_5606470e28; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_attachments
    ADD CONSTRAINT fk_rails_5606470e28 FOREIGN KEY (group_message_id) REFERENCES public.group_messages(id);


--
-- Name: task_submission_statuses fk_rails_6875a78786; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_submission_statuses
    ADD CONSTRAINT fk_rails_6875a78786 FOREIGN KEY (member_id) REFERENCES public.members(id);


--
-- Name: message_reactions fk_rails_74206c722e; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_reactions
    ADD CONSTRAINT fk_rails_74206c722e FOREIGN KEY (group_message_id) REFERENCES public.group_messages(id);


--
-- Name: sessions fk_rails_758836b4f0; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT fk_rails_758836b4f0 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: borrow_records fk_rails_7856a55ccd; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.borrow_records
    ADD CONSTRAINT fk_rails_7856a55ccd FOREIGN KEY (member_id) REFERENCES public.members(id);


--
-- Name: group_messages fk_rails_795ee8a403; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_messages
    ADD CONSTRAINT fk_rails_795ee8a403 FOREIGN KEY (reply_to_id) REFERENCES public.group_messages(id);


--
-- Name: link_previews fk_rails_7d6a0f8cae; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.link_previews
    ADD CONSTRAINT fk_rails_7d6a0f8cae FOREIGN KEY (group_message_id) REFERENCES public.group_messages(id);


--
-- Name: attendance_logs fk_rails_803b701b17; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.attendance_logs
    ADD CONSTRAINT fk_rails_803b701b17 FOREIGN KEY (created_by_id) REFERENCES public.users(id);


--
-- Name: return_requests fk_rails_80c7dca061; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.return_requests
    ADD CONSTRAINT fk_rails_80c7dca061 FOREIGN KEY (member_id) REFERENCES public.members(id);


--
-- Name: lab_rules_acceptances fk_rails_83c79cc30d; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lab_rules_acceptances
    ADD CONSTRAINT fk_rails_83c79cc30d FOREIGN KEY (member_id) REFERENCES public.members(id);


--
-- Name: poll_votes fk_rails_848ece0184; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.poll_votes
    ADD CONSTRAINT fk_rails_848ece0184 FOREIGN KEY (poll_option_id) REFERENCES public.poll_options(id);


--
-- Name: borrow_records fk_rails_945fc8fdf7; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.borrow_records
    ADD CONSTRAINT fk_rails_945fc8fdf7 FOREIGN KEY (inventory_item_id) REFERENCES public.inventory_items(id);


--
-- Name: return_requests fk_rails_9b8293de95; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.return_requests
    ADD CONSTRAINT fk_rails_9b8293de95 FOREIGN KEY (borrow_record_id) REFERENCES public.borrow_records(id);


--
-- Name: lab_rules_acceptances fk_rails_9ee7bcd475; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lab_rules_acceptances
    ADD CONSTRAINT fk_rails_9ee7bcd475 FOREIGN KEY (lab_rules_version_id) REFERENCES public.lab_rules_versions(id);


--
-- Name: group_messages fk_rails_a0d6893c0b; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_messages
    ADD CONSTRAINT fk_rails_a0d6893c0b FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: device_events fk_rails_a2ceaceca2; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_events
    ADD CONSTRAINT fk_rails_a2ceaceca2 FOREIGN KEY (device_registration_id) REFERENCES public.device_registrations(id);


--
-- Name: tasks fk_rails_a362a150d3; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT fk_rails_a362a150d3 FOREIGN KEY (created_by_id) REFERENCES public.users(id);


--
-- Name: polls fk_rails_a5c2e448c7; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.polls
    ADD CONSTRAINT fk_rails_a5c2e448c7 FOREIGN KEY (created_by_id) REFERENCES public.users(id);


--
-- Name: message_stars fk_rails_a63ad6d341; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_stars
    ADD CONSTRAINT fk_rails_a63ad6d341 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: poll_votes fk_rails_a6e6974b7e; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.poll_votes
    ADD CONSTRAINT fk_rails_a6e6974b7e FOREIGN KEY (poll_id) REFERENCES public.polls(id);


--
-- Name: poll_options fk_rails_aa85becb42; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.poll_options
    ADD CONSTRAINT fk_rails_aa85becb42 FOREIGN KEY (poll_id) REFERENCES public.polls(id);


--
-- Name: notifications fk_rails_b080fb4855; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT fk_rails_b080fb4855 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: poll_votes fk_rails_b64de9b025; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.poll_votes
    ADD CONSTRAINT fk_rails_b64de9b025 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: return_requests fk_rails_b7f3c678ae; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.return_requests
    ADD CONSTRAINT fk_rails_b7f3c678ae FOREIGN KEY (confirmed_by_id) REFERENCES public.users(id);


--
-- Name: extension_requests fk_rails_e14155367e; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.extension_requests
    ADD CONSTRAINT fk_rails_e14155367e FOREIGN KEY (member_id) REFERENCES public.members(id);


--
-- Name: message_receipts fk_rails_e34d709ae0; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_receipts
    ADD CONSTRAINT fk_rails_e34d709ae0 FOREIGN KEY (group_message_id) REFERENCES public.group_messages(id);


--
-- Name: message_receipts fk_rails_e41465cb0c; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.message_receipts
    ADD CONSTRAINT fk_rails_e41465cb0c FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: announcement_reads fk_rails_e929ca5a13; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.announcement_reads
    ADD CONSTRAINT fk_rails_e929ca5a13 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: inventory_imports fk_rails_f9662e836a; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inventory_imports
    ADD CONSTRAINT fk_rails_f9662e836a FOREIGN KEY (imported_by_id) REFERENCES public.users(id);


--
-- PostgreSQL database dump complete
--


