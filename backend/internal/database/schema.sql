--
-- PostgreSQL database dump
--

\restrict qbAC2UrhMHE1zaE9MlIjM4PChl1wePPcdfTXnu2vElWFvdmB0WVQmT2ZVEiYeKK

-- Dumped from database version 14.20 (Homebrew)
-- Dumped by pg_dump version 14.20 (Homebrew)

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
-- Name: ai; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA ai;


--
-- Name: analytics; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA analytics;


--
-- Name: catalog; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA catalog;


--
-- Name: cms; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA cms;


--
-- Name: commerce; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA commerce;


--
-- Name: crm; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA crm;


--
-- Name: erp; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA erp;


--
-- Name: identity; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA identity;


--
-- Name: inventory; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA inventory;


--
-- Name: payments; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA payments;


--
-- Name: platform; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA platform;


--
-- Name: pricing; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA pricing;


--
-- Name: promotions; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA promotions;


--
-- Name: suppliers; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA suppliers;


--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: product_status; Type: TYPE; Schema: catalog; Owner: -
--

CREATE TYPE catalog.product_status AS ENUM (
    'draft',
    'published',
    'archived'
);


--
-- Name: unit_type; Type: TYPE; Schema: catalog; Owner: -
--

CREATE TYPE catalog.unit_type AS ENUM (
    'base',
    'derived'
);


--
-- Name: order_status; Type: TYPE; Schema: commerce; Owner: -
--

CREATE TYPE commerce.order_status AS ENUM (
    'placed',
    'confirmed',
    'processing',
    'shipped',
    'delivered',
    'cancelled'
);


--
-- Name: user_type; Type: TYPE; Schema: identity; Owner: -
--

CREATE TYPE identity.user_type AS ENUM (
    'buyer',
    'supplier',
    'staff',
    'platform'
);


--
-- Name: verification_decision; Type: TYPE; Schema: identity; Owner: -
--

CREATE TYPE identity.verification_decision AS ENUM (
    'approved',
    'rejected'
);


--
-- Name: verification_status; Type: TYPE; Schema: identity; Owner: -
--

CREATE TYPE identity.verification_status AS ENUM (
    'pending',
    'approved',
    'rejected',
    'resubmitted'
);


--
-- Name: lot_status; Type: TYPE; Schema: inventory; Owner: -
--

CREATE TYPE inventory.lot_status AS ENUM (
    'active',
    'depleted',
    'expired',
    'quarantined'
);


--
-- Name: quarantine_status; Type: TYPE; Schema: inventory; Owner: -
--

CREATE TYPE inventory.quarantine_status AS ENUM (
    'pending',
    'quarantined',
    'released',
    'disposed'
);


--
-- Name: reservation_status; Type: TYPE; Schema: inventory; Owner: -
--

CREATE TYPE inventory.reservation_status AS ENUM (
    'reserved',
    'allocated',
    'released',
    'expired'
);


--
-- Name: price_list_status; Type: TYPE; Schema: pricing; Owner: -
--

CREATE TYPE pricing.price_list_status AS ENUM (
    'draft',
    'active',
    'inactive',
    'archived'
);


--
-- Name: price_type; Type: TYPE; Schema: pricing; Owner: -
--

CREATE TYPE pricing.price_type AS ENUM (
    'standard',
    'contract',
    'promotional'
);


--
-- Name: catalog_handling_class_type; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.catalog_handling_class_type AS ENUM (
    'ambient',
    'chilled',
    'frozen'
);


--
-- Name: backfill_rollups(date, date); Type: FUNCTION; Schema: analytics; Owner: -
--

CREATE FUNCTION analytics.backfill_rollups(p_start_date date, p_end_date date) RETURNS void
    LANGUAGE plpgsql
    AS $$
DECLARE
    v_date DATE;
BEGIN
    IF p_start_date > p_end_date THEN
        RAISE EXCEPTION 'start_date must be <= end_date';
    END IF;

    FOR v_date IN SELECT generate_series(p_start_date, p_end_date, INTERVAL '1 day')::DATE
    LOOP
        PERFORM analytics.run_daily_order_rollup(v_date);
        PERFORM analytics.run_daily_payment_rollup(v_date);
        PERFORM analytics.run_daily_inventory_rollup(v_date);
        PERFORM analytics.run_daily_promotion_rollup(v_date);
        PERFORM analytics.run_daily_financial_rollup(v_date);
        PERFORM analytics.run_daily_erp_sync_rollup(v_date);
    END LOOP;
END;
$$;


--
-- Name: date_window(date); Type: FUNCTION; Schema: analytics; Owner: -
--

CREATE FUNCTION analytics.date_window(p_date date) RETURNS TABLE(window_start timestamp with time zone, window_end timestamp with time zone)
    LANGUAGE sql STABLE
    AS $$
    SELECT p_date::TIMESTAMPTZ AT TIME ZONE 'UTC', (p_date + INTERVAL '1 day')::TIMESTAMPTZ AT TIME ZONE 'UTC';
$$;


--
-- Name: run_daily_erp_sync_rollup(date); Type: FUNCTION; Schema: analytics; Owner: -
--

CREATE FUNCTION analytics.run_daily_erp_sync_rollup(p_date date) RETURNS void
    LANGUAGE plpgsql
    AS $$
DECLARE
    v_start TIMESTAMPTZ;
    v_end TIMESTAMPTZ;
BEGIN
    SELECT window_start, window_end INTO v_start, v_end FROM analytics.date_window(p_date);

    DELETE FROM analytics.daily_erp_sync_rollup WHERE rollup_date = p_date;

    INSERT INTO analytics.daily_erp_sync_rollup (
        rollup_date, sync_type, job_status,
        total_records, matched_records, created_records, updated_records, drift_count, duration_ms
    )
    SELECT
        p_date,
        sj.job_type AS sync_type,
        sj.status AS job_status,
        COALESCE(SUM(sj.total_records), 0) AS total_records,
        COALESCE(SUM(sj.matched_records), 0) AS matched_records,
        COALESCE(SUM(sj.created_records), 0) AS created_records,
        COALESCE(SUM(sj.updated_records), 0) AS updated_records,
        COALESCE(SUM(sj.drift_count), 0) AS drift_count,
        COALESCE(SUM(EXTRACT(EPOCH FROM (sj.completed_at - sj.started_at)) * 1000), 0) AS duration_ms
    FROM erp.sync_job sj
    WHERE sj.created_at >= v_start AND sj.created_at < v_end
    GROUP BY sj.job_type, sj.status;
END;
$$;


--
-- Name: run_daily_financial_rollup(date); Type: FUNCTION; Schema: analytics; Owner: -
--

CREATE FUNCTION analytics.run_daily_financial_rollup(p_date date) RETURNS void
    LANGUAGE plpgsql
    AS $$
DECLARE
    v_start TIMESTAMPTZ;
    v_end TIMESTAMPTZ;
BEGIN
    SELECT window_start, window_end INTO v_start, v_end FROM analytics.date_window(p_date);

    DELETE FROM analytics.daily_financial_rollup WHERE rollup_date = p_date;

    INSERT INTO analytics.daily_financial_rollup (
        rollup_date, buyer_code, currency,
        invoiced_minor, collected_minor, credited_minor, refunded_minor, outstanding_minor
    )
    SELECT
        p_date,
        b.code AS buyer_code,
        inv.currency,
        COALESCE(SUM(CASE WHEN inv.status = 'issued' THEN inv.total_minor END), 0) AS invoiced_minor,
        COALESCE(SUM(CASE WHEN inv.status = 'paid' THEN inv.total_minor END), 0) AS collected_minor,
        COALESCE(SUM(CASE WHEN cn.status = 'issued' THEN cn.total_minor END), 0) AS credited_minor,
        COALESCE(SUM(CASE WHEN rf.status = 'completed' THEN rf.amount_minor END), 0) AS refunded_minor,
        COALESCE(SUM(CASE WHEN inv.status IN ('issued', 'partially_paid', 'overdue') THEN inv.total_minor - COALESCE(inv.paid_minor, 0) END), 0) AS outstanding_minor
    FROM identity.buyer b
    LEFT JOIN commerce.invoice inv ON inv.buyer_id = b.id AND inv.created_at >= v_start AND inv.created_at < v_end
    LEFT JOIN commerce.credit_note cn ON cn.buyer_id = b.id AND cn.created_at >= v_start AND cn.created_at < v_end
    LEFT JOIN payments.payment_refund rf ON rf.buyer_id = b.id AND rf.created_at >= v_start AND rf.created_at < v_end
    WHERE inv.id IS NOT NULL OR cn.id IS NOT NULL OR rf.id IS NOT NULL
    GROUP BY b.code, inv.currency;
END;
$$;


--
-- Name: run_daily_inventory_rollup(date); Type: FUNCTION; Schema: analytics; Owner: -
--

CREATE FUNCTION analytics.run_daily_inventory_rollup(p_date date) RETURNS void
    LANGUAGE plpgsql
    AS $$
DECLARE
    v_start TIMESTAMPTZ;
    v_end TIMESTAMPTZ;
BEGIN
    SELECT window_start, window_end INTO v_start, v_end FROM analytics.date_window(p_date);

    DELETE FROM analytics.daily_inventory_rollup WHERE rollup_date = p_date;

    INSERT INTO analytics.daily_inventory_rollup (
        rollup_date, product_code, supplier_code, location_code,
        stock_on_hand, stock_reserved, stock_available,
        lots_count, expiring_soon_count
    )
    SELECT
        p_date,
        p.code AS product_code,
        s.code AS supplier_code,
        COALESCE(sl.location_code, 'default') AS location_code,
        COALESCE(SUM(sl.quantity_on_hand), 0) AS stock_on_hand,
        COALESCE(SUM(sl.quantity_reserved), 0) AS stock_reserved,
        COALESCE(SUM(sl.quantity_on_hand - sl.quantity_reserved), 0) AS stock_available,
        COUNT(DISTINCT l.id) AS lots_count,
        COUNT(DISTINCT CASE WHEN l.expiry_date <= (p_date + INTERVAL '30 days') THEN l.id END) AS expiring_soon_count
    FROM inventory.stock_level sl
    JOIN catalog.product p ON p.id = sl.product_id
    JOIN catalog.supplier s ON s.id = p.supplier_id
    LEFT JOIN inventory.lot l ON l.stock_level_id = sl.id AND l.status = 'available'
    GROUP BY p.code, s.code, sl.location_code;
END;
$$;


--
-- Name: run_daily_order_rollup(date); Type: FUNCTION; Schema: analytics; Owner: -
--

CREATE FUNCTION analytics.run_daily_order_rollup(p_date date) RETURNS void
    LANGUAGE plpgsql
    AS $$
DECLARE
    v_start TIMESTAMPTZ;
    v_end TIMESTAMPTZ;
BEGIN
    -- Get date window
    SELECT window_start, window_end INTO v_start, v_end FROM analytics.date_window(p_date);

    -- Idempotent: delete existing data for this date
    DELETE FROM analytics.daily_order_rollup WHERE rollup_date = p_date;

    -- Aggregate from commerce.order and commerce.order_line
    INSERT INTO analytics.daily_order_rollup (
        rollup_date, buyer_code, supplier_code, order_count, line_count,
        total_quantity, revenue_minor, currency, status
    )
    SELECT
        p_date,
        o.buyer_code,
        ol.supplier_code,
        COUNT(DISTINCT o.id) AS order_count,
        COUNT(ol.id) AS line_count,
        COALESCE(SUM(ol.quantity), 0) AS total_quantity,
        COALESCE(SUM(ol.total_minor), 0) AS revenue_minor,
        ol.currency,
        o.status
    FROM commerce."order" o
    JOIN commerce.order_line ol ON ol.order_id = o.id
    WHERE o.placed_at >= v_start AND o.placed_at < v_end
    GROUP BY o.buyer_code, ol.supplier_code, ol.currency, o.status;
END;
$$;


--
-- Name: run_daily_payment_rollup(date); Type: FUNCTION; Schema: analytics; Owner: -
--

CREATE FUNCTION analytics.run_daily_payment_rollup(p_date date) RETURNS void
    LANGUAGE plpgsql
    AS $$
DECLARE
    v_start TIMESTAMPTZ;
    v_end TIMESTAMPTZ;
BEGIN
    SELECT window_start, window_end INTO v_start, v_end FROM analytics.date_window(p_date);

    DELETE FROM analytics.daily_payment_rollup WHERE rollup_date = p_date;

    INSERT INTO analytics.daily_payment_rollup (
        rollup_date, buyer_code, payment_method, payment_status,
        transaction_count, amount_minor, currency
    )
    SELECT
        p_date,
        b.code AS buyer_code,
        pi.method AS payment_method,
        pi.status AS payment_status,
        COUNT(pi.id) AS transaction_count,
        COALESCE(SUM(pi.amount_minor), 0) AS amount_minor,
        pi.currency
    FROM payments.payment_intent pi
    JOIN identity.buyer b ON b.id = pi.buyer_id
    WHERE pi.created_at >= v_start AND pi.created_at < v_end
    GROUP BY b.code, pi.method, pi.status, pi.currency;
END;
$$;


--
-- Name: run_daily_promotion_rollup(date); Type: FUNCTION; Schema: analytics; Owner: -
--

CREATE FUNCTION analytics.run_daily_promotion_rollup(p_date date) RETURNS void
    LANGUAGE plpgsql
    AS $$
DECLARE
    v_start TIMESTAMPTZ;
    v_end TIMESTAMPTZ;
BEGIN
    SELECT window_start, window_end INTO v_start, v_end FROM analytics.date_window(p_date);

    DELETE FROM analytics.daily_promotion_rollup WHERE rollup_date = p_date;

    INSERT INTO analytics.daily_promotion_rollup (
        rollup_date, promotion_code, voucher_code, redemptions, discount_minor, currency
    )
    SELECT
        p_date,
        pr.code AS promotion_code,
        v.code AS voucher_code,
        COUNT(r.id) AS redemptions,
        COALESCE(SUM(r.discount_minor), 0) AS discount_minor,
        r.currency
    FROM promotions.redemption r
    JOIN promotions.promotion pr ON pr.id = r.promotion_id
    LEFT JOIN promotions.voucher v ON v.id = r.voucher_id
    WHERE r.redeemed_at >= v_start AND r.redeemed_at < v_end
    GROUP BY pr.code, v.code, r.currency;
END;
$$;


--
-- Name: update_updated_at_column(); Type: FUNCTION; Schema: catalog; Owner: -
--

CREATE FUNCTION catalog.update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;


--
-- Name: update_updated_at_column(); Type: FUNCTION; Schema: commerce; Owner: -
--

CREATE FUNCTION commerce.update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;


--
-- Name: update_updated_at_column(); Type: FUNCTION; Schema: identity; Owner: -
--

CREATE FUNCTION identity.update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;


--
-- Name: update_updated_at_column(); Type: FUNCTION; Schema: inventory; Owner: -
--

CREATE FUNCTION inventory.update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;


--
-- Name: update_updated_at_column(); Type: FUNCTION; Schema: pricing; Owner: -
--

CREATE FUNCTION pricing.update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: budget; Type: TABLE; Schema: ai; Owner: -
--

CREATE TABLE ai.budget (
    id bigint NOT NULL,
    scope_key character varying(200) NOT NULL,
    period_start date NOT NULL,
    period_end date NOT NULL,
    ceiling_micros bigint NOT NULL,
    used_micros bigint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: budget_id_seq; Type: SEQUENCE; Schema: ai; Owner: -
--

CREATE SEQUENCE ai.budget_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: budget_id_seq; Type: SEQUENCE OWNED BY; Schema: ai; Owner: -
--

ALTER SEQUENCE ai.budget_id_seq OWNED BY ai.budget.id;


--
-- Name: model_routing; Type: TABLE; Schema: ai; Owner: -
--

CREATE TABLE ai.model_routing (
    id bigint NOT NULL,
    feature_key character varying(100) NOT NULL,
    provider character varying(100) NOT NULL,
    model_name character varying(100) NOT NULL,
    config jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: model_routing_id_seq; Type: SEQUENCE; Schema: ai; Owner: -
--

CREATE SEQUENCE ai.model_routing_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: model_routing_id_seq; Type: SEQUENCE OWNED BY; Schema: ai; Owner: -
--

ALTER SEQUENCE ai.model_routing_id_seq OWNED BY ai.model_routing.id;


--
-- Name: prompt; Type: TABLE; Schema: ai; Owner: -
--

CREATE TABLE ai.prompt (
    id bigint NOT NULL,
    key character varying(100) NOT NULL,
    name character varying(200) NOT NULL,
    description text,
    owner character varying(100),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: prompt_id_seq; Type: SEQUENCE; Schema: ai; Owner: -
--

CREATE SEQUENCE ai.prompt_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: prompt_id_seq; Type: SEQUENCE OWNED BY; Schema: ai; Owner: -
--

ALTER SEQUENCE ai.prompt_id_seq OWNED BY ai.prompt.id;


--
-- Name: prompt_version; Type: TABLE; Schema: ai; Owner: -
--

CREATE TABLE ai.prompt_version (
    id bigint NOT NULL,
    prompt_id bigint NOT NULL,
    version integer NOT NULL,
    content text NOT NULL,
    status character varying(20) DEFAULT 'draft'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT prompt_version_status_check CHECK (((status)::text = ANY ((ARRAY['draft'::character varying, 'active'::character varying, 'archived'::character varying])::text[])))
);


--
-- Name: prompt_version_id_seq; Type: SEQUENCE; Schema: ai; Owner: -
--

CREATE SEQUENCE ai.prompt_version_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: prompt_version_id_seq; Type: SEQUENCE OWNED BY; Schema: ai; Owner: -
--

ALTER SEQUENCE ai.prompt_version_id_seq OWNED BY ai.prompt_version.id;


--
-- Name: proposal; Type: TABLE; Schema: ai; Owner: -
--

CREATE TABLE ai.proposal (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    feature character varying(100) NOT NULL,
    status character varying(20) DEFAULT 'proposed'::character varying NOT NULL,
    confidence numeric(3,2),
    items jsonb DEFAULT '[]'::jsonb NOT NULL,
    unresolved jsonb DEFAULT '[]'::jsonb NOT NULL,
    provenance jsonb NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    tenant_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    confirmed_at timestamp with time zone,
    rejected_at timestamp with time zone,
    CONSTRAINT proposal_confidence_check CHECK (((confidence >= (0)::numeric) AND (confidence <= (1)::numeric))),
    CONSTRAINT proposal_status_check CHECK (((status)::text = ANY ((ARRAY['proposed'::character varying, 'confirmed'::character varying, 'rejected'::character varying, 'expired'::character varying])::text[])))
);


--
-- Name: proposal_id_seq; Type: SEQUENCE; Schema: ai; Owner: -
--

CREATE SEQUENCE ai.proposal_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: proposal_id_seq; Type: SEQUENCE OWNED BY; Schema: ai; Owner: -
--

ALTER SEQUENCE ai.proposal_id_seq OWNED BY ai.proposal.id;


--
-- Name: usage_record; Type: TABLE; Schema: ai; Owner: -
--

CREATE TABLE ai.usage_record (
    id bigint NOT NULL,
    proposal_id bigint,
    prompt_key character varying(100) NOT NULL,
    prompt_version integer NOT NULL,
    model character varying(200) NOT NULL,
    input_tokens integer DEFAULT 0 NOT NULL,
    output_tokens integer DEFAULT 0 NOT NULL,
    latency_ms integer DEFAULT 0 NOT NULL,
    cost_micros bigint DEFAULT 0 NOT NULL,
    tenant_id bigint,
    feature character varying(100) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: usage_record_id_seq; Type: SEQUENCE; Schema: ai; Owner: -
--

CREATE SEQUENCE ai.usage_record_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: usage_record_id_seq; Type: SEQUENCE OWNED BY; Schema: ai; Owner: -
--

ALTER SEQUENCE ai.usage_record_id_seq OWNED BY ai.usage_record.id;


--
-- Name: daily_erp_sync_rollup; Type: TABLE; Schema: analytics; Owner: -
--

CREATE TABLE analytics.daily_erp_sync_rollup (
    rollup_date date NOT NULL,
    sync_type character varying(32) NOT NULL,
    job_status character varying(32) NOT NULL,
    total_records bigint DEFAULT 0 NOT NULL,
    matched_records bigint DEFAULT 0 NOT NULL,
    created_records bigint DEFAULT 0 NOT NULL,
    updated_records bigint DEFAULT 0 NOT NULL,
    drift_count bigint DEFAULT 0 NOT NULL,
    duration_ms bigint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: daily_financial_rollup; Type: TABLE; Schema: analytics; Owner: -
--

CREATE TABLE analytics.daily_financial_rollup (
    rollup_date date NOT NULL,
    buyer_code character varying(32) NOT NULL,
    currency character(3) NOT NULL,
    invoiced_minor bigint DEFAULT 0 NOT NULL,
    collected_minor bigint DEFAULT 0 NOT NULL,
    credited_minor bigint DEFAULT 0 NOT NULL,
    refunded_minor bigint DEFAULT 0 NOT NULL,
    outstanding_minor bigint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: daily_inventory_rollup; Type: TABLE; Schema: analytics; Owner: -
--

CREATE TABLE analytics.daily_inventory_rollup (
    rollup_date date NOT NULL,
    product_code character varying(64) NOT NULL,
    supplier_code character varying(32) NOT NULL,
    location_code character varying(32) NOT NULL,
    stock_on_hand numeric(18,4) DEFAULT 0 NOT NULL,
    stock_reserved numeric(18,4) DEFAULT 0 NOT NULL,
    stock_available numeric(18,4) DEFAULT 0 NOT NULL,
    lots_count integer DEFAULT 0 NOT NULL,
    expiring_soon_count integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: daily_order_rollup; Type: TABLE; Schema: analytics; Owner: -
--

CREATE TABLE analytics.daily_order_rollup (
    rollup_date date NOT NULL,
    buyer_code character varying(32) NOT NULL,
    supplier_code character varying(32) NOT NULL,
    order_count bigint DEFAULT 0 NOT NULL,
    line_count bigint DEFAULT 0 NOT NULL,
    total_quantity numeric(18,4) DEFAULT 0 NOT NULL,
    revenue_minor bigint DEFAULT 0 NOT NULL,
    currency character(3) NOT NULL,
    status character varying(32) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: daily_payment_rollup; Type: TABLE; Schema: analytics; Owner: -
--

CREATE TABLE analytics.daily_payment_rollup (
    rollup_date date NOT NULL,
    buyer_code character varying(32) NOT NULL,
    payment_method character varying(64) NOT NULL,
    payment_status character varying(32) NOT NULL,
    transaction_count bigint DEFAULT 0 NOT NULL,
    amount_minor bigint DEFAULT 0 NOT NULL,
    currency character(3) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: daily_promotion_rollup; Type: TABLE; Schema: analytics; Owner: -
--

CREATE TABLE analytics.daily_promotion_rollup (
    rollup_date date NOT NULL,
    promotion_code character varying(64) NOT NULL,
    voucher_code character varying(64) NOT NULL,
    redemptions bigint DEFAULT 0 NOT NULL,
    discount_minor bigint DEFAULT 0 NOT NULL,
    currency character(3) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: attribute; Type: TABLE; Schema: catalog; Owner: -
--

CREATE TABLE catalog.attribute (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    name character varying(100) NOT NULL,
    attribute_type character varying(50) NOT NULL,
    is_filterable boolean DEFAULT false NOT NULL,
    is_required boolean DEFAULT false NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: attribute_id_seq; Type: SEQUENCE; Schema: catalog; Owner: -
--

CREATE SEQUENCE catalog.attribute_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: attribute_id_seq; Type: SEQUENCE OWNED BY; Schema: catalog; Owner: -
--

ALTER SEQUENCE catalog.attribute_id_seq OWNED BY catalog.attribute.id;


--
-- Name: attribute_value; Type: TABLE; Schema: catalog; Owner: -
--

CREATE TABLE catalog.attribute_value (
    id bigint NOT NULL,
    attribute_id bigint NOT NULL,
    value character varying(255) NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: attribute_value_id_seq; Type: SEQUENCE; Schema: catalog; Owner: -
--

CREATE SEQUENCE catalog.attribute_value_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: attribute_value_id_seq; Type: SEQUENCE OWNED BY; Schema: catalog; Owner: -
--

ALTER SEQUENCE catalog.attribute_value_id_seq OWNED BY catalog.attribute_value.id;


--
-- Name: brand; Type: TABLE; Schema: catalog; Owner: -
--

CREATE TABLE catalog.brand (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    name character varying(255) NOT NULL,
    slug character varying(255) NOT NULL,
    description text,
    logo_url character varying(500),
    website_url character varying(500),
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: brand_id_seq; Type: SEQUENCE; Schema: catalog; Owner: -
--

CREATE SEQUENCE catalog.brand_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: brand_id_seq; Type: SEQUENCE OWNED BY; Schema: catalog; Owner: -
--

ALTER SEQUENCE catalog.brand_id_seq OWNED BY catalog.brand.id;


--
-- Name: category; Type: TABLE; Schema: catalog; Owner: -
--

CREATE TABLE catalog.category (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    name character varying(255) NOT NULL,
    slug character varying(255) NOT NULL,
    description text,
    parent_id bigint,
    sort_order integer DEFAULT 0 NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    seo_title character varying(255),
    seo_description text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: category_id_seq; Type: SEQUENCE; Schema: catalog; Owner: -
--

CREATE SEQUENCE catalog.category_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: category_id_seq; Type: SEQUENCE OWNED BY; Schema: catalog; Owner: -
--

ALTER SEQUENCE catalog.category_id_seq OWNED BY catalog.category.id;


--
-- Name: handling_class; Type: TABLE; Schema: catalog; Owner: -
--

CREATE TABLE catalog.handling_class (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    name public.catalog_handling_class_type NOT NULL,
    description text,
    sort_order integer DEFAULT 0 NOT NULL,
    temperature_min_c integer,
    temperature_max_c integer,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: handling_class_id_seq; Type: SEQUENCE; Schema: catalog; Owner: -
--

CREATE SEQUENCE catalog.handling_class_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: handling_class_id_seq; Type: SEQUENCE OWNED BY; Schema: catalog; Owner: -
--

ALTER SEQUENCE catalog.handling_class_id_seq OWNED BY catalog.handling_class.id;


--
-- Name: product; Type: TABLE; Schema: catalog; Owner: -
--

CREATE TABLE catalog.product (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    slug character varying(255) NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    short_description character varying(500),
    category_id bigint NOT NULL,
    brand_id bigint,
    handling_class public.catalog_handling_class_type NOT NULL,
    base_unit_id bigint NOT NULL,
    supplier_id bigint,
    status catalog.product_status DEFAULT 'draft'::catalog.product_status NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    is_featured boolean DEFAULT false NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    seo_title character varying(255),
    seo_description text,
    seo_keywords text,
    base_price_minor bigint,
    currency character(3) DEFAULT 'USD'::bpchar NOT NULL,
    track_inventory boolean DEFAULT true NOT NULL,
    weight_grams integer,
    length_mm integer,
    width_mm integer,
    height_mm integer,
    gtin character varying(14),
    sku character varying(100),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    published_at timestamp with time zone,
    deleted_at timestamp with time zone
);


--
-- Name: product_attribute; Type: TABLE; Schema: catalog; Owner: -
--

CREATE TABLE catalog.product_attribute (
    id bigint NOT NULL,
    product_id bigint NOT NULL,
    attribute_id bigint NOT NULL,
    attribute_value_id bigint,
    text_value text,
    number_value numeric(18,6),
    boolean_value boolean,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: product_attribute_id_seq; Type: SEQUENCE; Schema: catalog; Owner: -
--

CREATE SEQUENCE catalog.product_attribute_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: product_attribute_id_seq; Type: SEQUENCE OWNED BY; Schema: catalog; Owner: -
--

ALTER SEQUENCE catalog.product_attribute_id_seq OWNED BY catalog.product_attribute.id;


--
-- Name: product_id_seq; Type: SEQUENCE; Schema: catalog; Owner: -
--

CREATE SEQUENCE catalog.product_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: product_id_seq; Type: SEQUENCE OWNED BY; Schema: catalog; Owner: -
--

ALTER SEQUENCE catalog.product_id_seq OWNED BY catalog.product.id;


--
-- Name: product_media; Type: TABLE; Schema: catalog; Owner: -
--

CREATE TABLE catalog.product_media (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    product_id bigint NOT NULL,
    url character varying(500) NOT NULL,
    alt_text character varying(255),
    media_type character varying(50) DEFAULT 'image'::character varying NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    is_primary boolean DEFAULT false NOT NULL,
    width_px integer,
    height_px integer,
    file_size_bytes bigint,
    mime_type character varying(100),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: product_media_id_seq; Type: SEQUENCE; Schema: catalog; Owner: -
--

CREATE SEQUENCE catalog.product_media_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: product_media_id_seq; Type: SEQUENCE OWNED BY; Schema: catalog; Owner: -
--

ALTER SEQUENCE catalog.product_media_id_seq OWNED BY catalog.product_media.id;


--
-- Name: product_unit; Type: TABLE; Schema: catalog; Owner: -
--

CREATE TABLE catalog.product_unit (
    id bigint NOT NULL,
    product_id bigint NOT NULL,
    unit_id bigint NOT NULL,
    conversion_factor numeric(18,6) NOT NULL,
    is_default boolean DEFAULT false NOT NULL,
    price_minor bigint,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: product_unit_id_seq; Type: SEQUENCE; Schema: catalog; Owner: -
--

CREATE SEQUENCE catalog.product_unit_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: product_unit_id_seq; Type: SEQUENCE OWNED BY; Schema: catalog; Owner: -
--

ALTER SEQUENCE catalog.product_unit_id_seq OWNED BY catalog.product_unit.id;


--
-- Name: supplier; Type: TABLE; Schema: catalog; Owner: -
--

CREATE TABLE catalog.supplier (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    supplier_id bigint NOT NULL,
    name character varying(255) NOT NULL,
    slug character varying(255) NOT NULL,
    description text,
    logo_url character varying(500),
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: supplier_id_seq; Type: SEQUENCE; Schema: catalog; Owner: -
--

CREATE SEQUENCE catalog.supplier_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: supplier_id_seq; Type: SEQUENCE OWNED BY; Schema: catalog; Owner: -
--

ALTER SEQUENCE catalog.supplier_id_seq OWNED BY catalog.supplier.id;


--
-- Name: unit; Type: TABLE; Schema: catalog; Owner: -
--

CREATE TABLE catalog.unit (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    name character varying(100) NOT NULL,
    symbol character varying(20) NOT NULL,
    unit_type catalog.unit_type DEFAULT 'base'::catalog.unit_type NOT NULL,
    base_unit_id bigint,
    conversion_factor numeric(18,6) DEFAULT 1 NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: unit_id_seq; Type: SEQUENCE; Schema: catalog; Owner: -
--

CREATE SEQUENCE catalog.unit_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: unit_id_seq; Type: SEQUENCE OWNED BY; Schema: catalog; Owner: -
--

ALTER SEQUENCE catalog.unit_id_seq OWNED BY catalog.unit.id;


--
-- Name: article; Type: TABLE; Schema: cms; Owner: -
--

CREATE TABLE cms.article (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    slug character varying(255) NOT NULL,
    title character varying(255) NOT NULL,
    excerpt text,
    body text NOT NULL,
    author character varying(255),
    status character varying(16) DEFAULT 'draft'::character varying NOT NULL,
    published_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT article_status_check CHECK (((status)::text = ANY ((ARRAY['draft'::character varying, 'published'::character varying, 'archived'::character varying])::text[])))
);


--
-- Name: article_id_seq; Type: SEQUENCE; Schema: cms; Owner: -
--

CREATE SEQUENCE cms.article_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: article_id_seq; Type: SEQUENCE OWNED BY; Schema: cms; Owner: -
--

ALTER SEQUENCE cms.article_id_seq OWNED BY cms.article.id;


--
-- Name: banner; Type: TABLE; Schema: cms; Owner: -
--

CREATE TABLE cms.banner (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    title character varying(255) NOT NULL,
    image_url character varying(500) NOT NULL,
    link_url character varying(500),
    "position" character varying(64) DEFAULT 'home_hero'::character varying NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: banner_id_seq; Type: SEQUENCE; Schema: cms; Owner: -
--

CREATE SEQUENCE cms.banner_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: banner_id_seq; Type: SEQUENCE OWNED BY; Schema: cms; Owner: -
--

ALTER SEQUENCE cms.banner_id_seq OWNED BY cms.banner.id;


--
-- Name: contact_enquiry; Type: TABLE; Schema: cms; Owner: -
--

CREATE TABLE cms.contact_enquiry (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    name character varying(200) NOT NULL,
    email character varying(255) NOT NULL,
    phone character varying(50),
    company character varying(200),
    subject character varying(200) NOT NULL,
    message text NOT NULL,
    source character varying(50) DEFAULT 'contact_form'::character varying NOT NULL,
    status character varying(50) DEFAULT 'new'::character varying NOT NULL,
    lead_id bigint,
    ip_address character varying(45),
    user_agent text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    routed_at timestamp with time zone
);


--
-- Name: contact_enquiry_id_seq; Type: SEQUENCE; Schema: cms; Owner: -
--

CREATE SEQUENCE cms.contact_enquiry_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: contact_enquiry_id_seq; Type: SEQUENCE OWNED BY; Schema: cms; Owner: -
--

ALTER SEQUENCE cms.contact_enquiry_id_seq OWNED BY cms.contact_enquiry.id;


--
-- Name: faq; Type: TABLE; Schema: cms; Owner: -
--

CREATE TABLE cms.faq (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    question text NOT NULL,
    answer text NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    category character varying(100)
);


--
-- Name: faq_id_seq; Type: SEQUENCE; Schema: cms; Owner: -
--

CREATE SEQUENCE cms.faq_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: faq_id_seq; Type: SEQUENCE OWNED BY; Schema: cms; Owner: -
--

ALTER SEQUENCE cms.faq_id_seq OWNED BY cms.faq.id;


--
-- Name: homepage_layout; Type: TABLE; Schema: cms; Owner: -
--

CREATE TABLE cms.homepage_layout (
    id bigint NOT NULL,
    draft_sections jsonb DEFAULT '[]'::jsonb NOT NULL,
    published_sections jsonb DEFAULT '[]'::jsonb NOT NULL,
    published_at timestamp with time zone,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by text
);


--
-- Name: homepage_layout_id_seq; Type: SEQUENCE; Schema: cms; Owner: -
--

CREATE SEQUENCE cms.homepage_layout_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: homepage_layout_id_seq; Type: SEQUENCE OWNED BY; Schema: cms; Owner: -
--

ALTER SEQUENCE cms.homepage_layout_id_seq OWNED BY cms.homepage_layout.id;


--
-- Name: legal_document; Type: TABLE; Schema: cms; Owner: -
--

CREATE TABLE cms.legal_document (
    doc_type text NOT NULL,
    title text NOT NULL,
    current_version integer,
    effective_at timestamp with time zone,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by text,
    draft_title text,
    draft_body text,
    draft_body_format text DEFAULT 'markdown'::text NOT NULL,
    draft_effective_at timestamp with time zone,
    draft_updated_at timestamp with time zone,
    draft_updated_by text,
    CONSTRAINT legal_document_draft_body_format_check CHECK ((draft_body_format = ANY (ARRAY['markdown'::text, 'html'::text, 'plaintext'::text])))
);


--
-- Name: legal_document_version; Type: TABLE; Schema: cms; Owner: -
--

CREATE TABLE cms.legal_document_version (
    id bigint NOT NULL,
    doc_type text NOT NULL,
    version integer NOT NULL,
    title text NOT NULL,
    body text NOT NULL,
    body_format text DEFAULT 'markdown'::text NOT NULL,
    status text DEFAULT 'published'::text NOT NULL,
    effective_at timestamp with time zone NOT NULL,
    published_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by text,
    CONSTRAINT legal_document_version_body_format_check CHECK ((body_format = ANY (ARRAY['markdown'::text, 'html'::text, 'plaintext'::text]))),
    CONSTRAINT legal_document_version_status_check CHECK ((status = ANY (ARRAY['published'::text, 'superseded'::text])))
);


--
-- Name: legal_document_version_id_seq; Type: SEQUENCE; Schema: cms; Owner: -
--

CREATE SEQUENCE cms.legal_document_version_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: legal_document_version_id_seq; Type: SEQUENCE OWNED BY; Schema: cms; Owner: -
--

ALTER SEQUENCE cms.legal_document_version_id_seq OWNED BY cms.legal_document_version.id;


--
-- Name: menu_item; Type: TABLE; Schema: cms; Owner: -
--

CREATE TABLE cms.menu_item (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    location character varying(64) NOT NULL,
    label character varying(255) NOT NULL,
    url character varying(500) NOT NULL,
    parent_id bigint,
    sort_order integer DEFAULT 0 NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: menu_item_id_seq; Type: SEQUENCE; Schema: cms; Owner: -
--

CREATE SEQUENCE cms.menu_item_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: menu_item_id_seq; Type: SEQUENCE OWNED BY; Schema: cms; Owner: -
--

ALTER SEQUENCE cms.menu_item_id_seq OWNED BY cms.menu_item.id;


--
-- Name: newsletter_subscriber; Type: TABLE; Schema: cms; Owner: -
--

CREATE TABLE cms.newsletter_subscriber (
    id bigint NOT NULL,
    code text NOT NULL,
    email text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    first_name text,
    source text,
    confirmation_token_hash text,
    token_expires_at timestamp with time zone,
    unsubscribe_token_hash text,
    unsubscribe_token_expires_at timestamp with time zone,
    subscribed_at timestamp with time zone,
    confirmed_at timestamp with time zone,
    unsubscribed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT newsletter_subscriber_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'confirmed'::text, 'unsubscribed'::text])))
);


--
-- Name: newsletter_subscriber_id_seq; Type: SEQUENCE; Schema: cms; Owner: -
--

CREATE SEQUENCE cms.newsletter_subscriber_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: newsletter_subscriber_id_seq; Type: SEQUENCE OWNED BY; Schema: cms; Owner: -
--

ALTER SEQUENCE cms.newsletter_subscriber_id_seq OWNED BY cms.newsletter_subscriber.id;


--
-- Name: page; Type: TABLE; Schema: cms; Owner: -
--

CREATE TABLE cms.page (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    slug character varying(255) NOT NULL,
    title character varying(255) NOT NULL,
    body text NOT NULL,
    meta_title character varying(255),
    meta_description text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: page_id_seq; Type: SEQUENCE; Schema: cms; Owner: -
--

CREATE SEQUENCE cms.page_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: page_id_seq; Type: SEQUENCE OWNED BY; Schema: cms; Owner: -
--

ALTER SEQUENCE cms.page_id_seq OWNED BY cms.page.id;


--
-- Name: seo_settings; Type: TABLE; Schema: cms; Owner: -
--

CREATE TABLE cms.seo_settings (
    id bigint NOT NULL,
    key character varying(100) NOT NULL,
    value text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: seo_settings_id_seq; Type: SEQUENCE; Schema: cms; Owner: -
--

CREATE SEQUENCE cms.seo_settings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: seo_settings_id_seq; Type: SEQUENCE OWNED BY; Schema: cms; Owner: -
--

ALTER SEQUENCE cms.seo_settings_id_seq OWNED BY cms.seo_settings.id;


--
-- Name: seo_template; Type: TABLE; Schema: cms; Owner: -
--

CREATE TABLE cms.seo_template (
    id bigint NOT NULL,
    key character varying(100) NOT NULL,
    name character varying(200) NOT NULL,
    title_template text NOT NULL,
    description_template text,
    structured_data jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: seo_template_id_seq; Type: SEQUENCE; Schema: cms; Owner: -
--

CREATE SEQUENCE cms.seo_template_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: seo_template_id_seq; Type: SEQUENCE OWNED BY; Schema: cms; Owner: -
--

ALTER SEQUENCE cms.seo_template_id_seq OWNED BY cms.seo_template.id;


--
-- Name: setting; Type: TABLE; Schema: cms; Owner: -
--

CREATE TABLE cms.setting (
    id bigint NOT NULL,
    key character varying(255) NOT NULL,
    value text NOT NULL,
    group_name character varying(64) DEFAULT 'general'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: setting_id_seq; Type: SEQUENCE; Schema: cms; Owner: -
--

CREATE SEQUENCE cms.setting_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: setting_id_seq; Type: SEQUENCE OWNED BY; Schema: cms; Owner: -
--

ALTER SEQUENCE cms.setting_id_seq OWNED BY cms.setting.id;


--
-- Name: cart; Type: TABLE; Schema: commerce; Owner: -
--

CREATE TABLE commerce.cart (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    buyer_id bigint NOT NULL,
    currency character(3) DEFAULT 'USD'::bpchar NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    voucher_code character varying(64)
);


--
-- Name: cart_id_seq; Type: SEQUENCE; Schema: commerce; Owner: -
--

CREATE SEQUENCE commerce.cart_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: cart_id_seq; Type: SEQUENCE OWNED BY; Schema: commerce; Owner: -
--

ALTER SEQUENCE commerce.cart_id_seq OWNED BY commerce.cart.id;


--
-- Name: cart_line; Type: TABLE; Schema: commerce; Owner: -
--

CREATE TABLE commerce.cart_line (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    cart_id bigint NOT NULL,
    product_id bigint NOT NULL,
    supplier_id bigint NOT NULL,
    unit_id bigint NOT NULL,
    quantity integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT cart_line_quantity_check CHECK ((quantity > 0))
);


--
-- Name: cart_line_id_seq; Type: SEQUENCE; Schema: commerce; Owner: -
--

CREATE SEQUENCE commerce.cart_line_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: cart_line_id_seq; Type: SEQUENCE OWNED BY; Schema: commerce; Owner: -
--

ALTER SEQUENCE commerce.cart_line_id_seq OWNED BY commerce.cart_line.id;


--
-- Name: checkout_idempotency; Type: TABLE; Schema: commerce; Owner: -
--

CREATE TABLE commerce.checkout_idempotency (
    id bigint NOT NULL,
    buyer_id bigint NOT NULL,
    idem_key character varying(100) NOT NULL,
    payload_hash character(64) NOT NULL,
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    result jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT checkout_idempotency_status_check CHECK (((status)::text = ANY ((ARRAY['pending'::character varying, 'completed'::character varying])::text[])))
);


--
-- Name: checkout_idempotency_id_seq; Type: SEQUENCE; Schema: commerce; Owner: -
--

CREATE SEQUENCE commerce.checkout_idempotency_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: checkout_idempotency_id_seq; Type: SEQUENCE OWNED BY; Schema: commerce; Owner: -
--

ALTER SEQUENCE commerce.checkout_idempotency_id_seq OWNED BY commerce.checkout_idempotency.id;


--
-- Name: credit_note; Type: TABLE; Schema: commerce; Owner: -
--

CREATE TABLE commerce.credit_note (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    batch_code character varying(26) NOT NULL,
    order_id bigint NOT NULL,
    invoice_id bigint,
    return_id bigint,
    amount_minor bigint NOT NULL,
    currency character(3) DEFAULT 'USD'::bpchar NOT NULL,
    reason text NOT NULL,
    status character varying(16) DEFAULT 'applied'::character varying NOT NULL,
    actor bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT credit_note_amount_minor_check CHECK ((amount_minor > 0)),
    CONSTRAINT credit_note_reason_check CHECK ((char_length(reason) > 0)),
    CONSTRAINT credit_note_status_check CHECK (((status)::text = ANY ((ARRAY['applied'::character varying, 'void'::character varying])::text[])))
);


--
-- Name: credit_note_id_seq; Type: SEQUENCE; Schema: commerce; Owner: -
--

CREATE SEQUENCE commerce.credit_note_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: credit_note_id_seq; Type: SEQUENCE OWNED BY; Schema: commerce; Owner: -
--

ALTER SEQUENCE commerce.credit_note_id_seq OWNED BY commerce.credit_note.id;


--
-- Name: invoice; Type: TABLE; Schema: commerce; Owner: -
--

CREATE TABLE commerce.invoice (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    order_id bigint NOT NULL,
    subtotal_minor bigint DEFAULT 0 NOT NULL,
    total_minor bigint DEFAULT 0 NOT NULL,
    balance_minor bigint DEFAULT 0 NOT NULL,
    currency character(3) DEFAULT 'USD'::bpchar NOT NULL,
    status character varying(16) DEFAULT 'draft'::character varying NOT NULL,
    issued_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    replaces_code character varying(26),
    CONSTRAINT invoice_balance_minor_check CHECK ((balance_minor >= 0)),
    CONSTRAINT invoice_status_check CHECK (((status)::text = ANY ((ARRAY['draft'::character varying, 'issued'::character varying, 'void'::character varying])::text[]))),
    CONSTRAINT invoice_subtotal_minor_check CHECK ((subtotal_minor >= 0)),
    CONSTRAINT invoice_total_minor_check CHECK ((total_minor >= 0))
);


--
-- Name: invoice_id_seq; Type: SEQUENCE; Schema: commerce; Owner: -
--

CREATE SEQUENCE commerce.invoice_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: invoice_id_seq; Type: SEQUENCE OWNED BY; Schema: commerce; Owner: -
--

ALTER SEQUENCE commerce.invoice_id_seq OWNED BY commerce.invoice.id;


--
-- Name: order; Type: TABLE; Schema: commerce; Owner: -
--

CREATE TABLE commerce."order" (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    buyer_id bigint NOT NULL,
    supplier_id bigint NOT NULL,
    cart_id bigint,
    cart_code character varying(26),
    currency character(3) DEFAULT 'USD'::bpchar NOT NULL,
    subtotal_minor bigint DEFAULT 0 NOT NULL,
    discounts_minor bigint DEFAULT 0 NOT NULL,
    total_minor bigint DEFAULT 0 NOT NULL,
    status commerce.order_status DEFAULT 'placed'::commerce.order_status NOT NULL,
    placed_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    on_hold boolean DEFAULT false NOT NULL,
    hold_reason text,
    idem_key character varying(100),
    CONSTRAINT order_discounts_minor_check CHECK ((discounts_minor >= 0)),
    CONSTRAINT order_subtotal_minor_check CHECK ((subtotal_minor >= 0)),
    CONSTRAINT order_total_minor_check CHECK ((total_minor >= 0))
);


--
-- Name: order_history; Type: TABLE; Schema: commerce; Owner: -
--

CREATE TABLE commerce.order_history (
    id bigint NOT NULL,
    order_id bigint NOT NULL,
    from_status commerce.order_status,
    to_status commerce.order_status NOT NULL,
    actor bigint,
    reason text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: order_history_id_seq; Type: SEQUENCE; Schema: commerce; Owner: -
--

CREATE SEQUENCE commerce.order_history_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: order_history_id_seq; Type: SEQUENCE OWNED BY; Schema: commerce; Owner: -
--

ALTER SEQUENCE commerce.order_history_id_seq OWNED BY commerce.order_history.id;


--
-- Name: order_id_seq; Type: SEQUENCE; Schema: commerce; Owner: -
--

CREATE SEQUENCE commerce.order_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: order_id_seq; Type: SEQUENCE OWNED BY; Schema: commerce; Owner: -
--

ALTER SEQUENCE commerce.order_id_seq OWNED BY commerce."order".id;


--
-- Name: order_line; Type: TABLE; Schema: commerce; Owner: -
--

CREATE TABLE commerce.order_line (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    order_id bigint NOT NULL,
    product_id bigint NOT NULL,
    supplier_id bigint NOT NULL,
    unit_id bigint NOT NULL,
    quantity integer NOT NULL,
    unit_price_minor bigint NOT NULL,
    total_price_minor bigint NOT NULL,
    currency character(3) DEFAULT 'USD'::bpchar NOT NULL,
    product_code character varying(64) NOT NULL,
    product_name character varying(255) NOT NULL,
    unit_code character varying(32) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT order_line_quantity_check CHECK ((quantity > 0)),
    CONSTRAINT order_line_total_price_minor_check CHECK ((total_price_minor >= 0)),
    CONSTRAINT order_line_unit_price_minor_check CHECK ((unit_price_minor >= 0))
);


--
-- Name: order_line_id_seq; Type: SEQUENCE; Schema: commerce; Owner: -
--

CREATE SEQUENCE commerce.order_line_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: order_line_id_seq; Type: SEQUENCE OWNED BY; Schema: commerce; Owner: -
--

ALTER SEQUENCE commerce.order_line_id_seq OWNED BY commerce.order_line.id;


--
-- Name: return_line; Type: TABLE; Schema: commerce; Owner: -
--

CREATE TABLE commerce.return_line (
    id bigint NOT NULL,
    return_id bigint NOT NULL,
    order_line_id bigint NOT NULL,
    quantity integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT return_line_quantity_check CHECK ((quantity > 0))
);


--
-- Name: return_line_id_seq; Type: SEQUENCE; Schema: commerce; Owner: -
--

CREATE SEQUENCE commerce.return_line_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: return_line_id_seq; Type: SEQUENCE OWNED BY; Schema: commerce; Owner: -
--

ALTER SEQUENCE commerce.return_line_id_seq OWNED BY commerce.return_line.id;


--
-- Name: return_request; Type: TABLE; Schema: commerce; Owner: -
--

CREATE TABLE commerce.return_request (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    order_id bigint NOT NULL,
    buyer_id bigint NOT NULL,
    reason text NOT NULL,
    status character varying(16) DEFAULT 'requested'::character varying NOT NULL,
    requested_by bigint,
    decided_by bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT return_request_reason_check CHECK ((char_length(reason) > 0)),
    CONSTRAINT return_request_status_check CHECK (((status)::text = ANY ((ARRAY['requested'::character varying, 'approved'::character varying, 'rejected'::character varying, 'completed'::character varying])::text[])))
);


--
-- Name: return_request_id_seq; Type: SEQUENCE; Schema: commerce; Owner: -
--

CREATE SEQUENCE commerce.return_request_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: return_request_id_seq; Type: SEQUENCE OWNED BY; Schema: commerce; Owner: -
--

ALTER SEQUENCE commerce.return_request_id_seq OWNED BY commerce.return_request.id;


--
-- Name: shipment; Type: TABLE; Schema: commerce; Owner: -
--

CREATE TABLE commerce.shipment (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    order_id bigint NOT NULL,
    carrier character varying(64),
    tracking_code character varying(128),
    status character varying(16) DEFAULT 'preparing'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT shipment_status_check CHECK (((status)::text = ANY ((ARRAY['preparing'::character varying, 'shipped'::character varying, 'delivered'::character varying, 'cancelled'::character varying])::text[])))
);


--
-- Name: shipment_id_seq; Type: SEQUENCE; Schema: commerce; Owner: -
--

CREATE SEQUENCE commerce.shipment_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: shipment_id_seq; Type: SEQUENCE OWNED BY; Schema: commerce; Owner: -
--

ALTER SEQUENCE commerce.shipment_id_seq OWNED BY commerce.shipment.id;


--
-- Name: shipment_line; Type: TABLE; Schema: commerce; Owner: -
--

CREATE TABLE commerce.shipment_line (
    id bigint NOT NULL,
    shipment_id bigint NOT NULL,
    order_line_id bigint NOT NULL,
    quantity integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT shipment_line_quantity_check CHECK ((quantity > 0))
);


--
-- Name: shipment_line_id_seq; Type: SEQUENCE; Schema: commerce; Owner: -
--

CREATE SEQUENCE commerce.shipment_line_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: shipment_line_id_seq; Type: SEQUENCE OWNED BY; Schema: commerce; Owner: -
--

ALTER SEQUENCE commerce.shipment_line_id_seq OWNED BY commerce.shipment_line.id;


--
-- Name: lead; Type: TABLE; Schema: crm; Owner: -
--

CREATE TABLE crm.lead (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    contact_name character varying(255) NOT NULL,
    business_name character varying(255) NOT NULL,
    email character varying(255),
    phone character varying(64),
    message text,
    status character varying(16) DEFAULT 'new'::character varying NOT NULL,
    partner_id bigint,
    referral_id bigint,
    assigned_to bigint,
    buyer_id bigint,
    notes text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT lead_status_check CHECK (((status)::text = ANY ((ARRAY['new'::character varying, 'assigned'::character varying, 'contacted'::character varying, 'converted'::character varying, 'closed'::character varying])::text[])))
);


--
-- Name: lead_id_seq; Type: SEQUENCE; Schema: crm; Owner: -
--

CREATE SEQUENCE crm.lead_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: lead_id_seq; Type: SEQUENCE OWNED BY; Schema: crm; Owner: -
--

ALTER SEQUENCE crm.lead_id_seq OWNED BY crm.lead.id;


--
-- Name: partner; Type: TABLE; Schema: crm; Owner: -
--

CREATE TABLE crm.partner (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    name character varying(255) NOT NULL,
    slug character varying(255) NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: partner_id_seq; Type: SEQUENCE; Schema: crm; Owner: -
--

CREATE SEQUENCE crm.partner_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: partner_id_seq; Type: SEQUENCE OWNED BY; Schema: crm; Owner: -
--

ALTER SEQUENCE crm.partner_id_seq OWNED BY crm.partner.id;


--
-- Name: referral_code; Type: TABLE; Schema: crm; Owner: -
--

CREATE TABLE crm.referral_code (
    id bigint NOT NULL,
    code character varying(64) NOT NULL,
    partner_id bigint NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: referral_code_id_seq; Type: SEQUENCE; Schema: crm; Owner: -
--

CREATE SEQUENCE crm.referral_code_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: referral_code_id_seq; Type: SEQUENCE OWNED BY; Schema: crm; Owner: -
--

ALTER SEQUENCE crm.referral_code_id_seq OWNED BY crm.referral_code.id;


--
-- Name: address; Type: TABLE; Schema: identity; Owner: -
--

CREATE TABLE identity.address (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    owner_user_id bigint NOT NULL,
    owner_type identity.user_type NOT NULL,
    label character varying(100) NOT NULL,
    recipient_name character varying(255),
    company_name character varying(255),
    line1 character varying(255) NOT NULL,
    line2 character varying(255),
    city character varying(100) NOT NULL,
    state_province character varying(100),
    postal_code character varying(20) NOT NULL,
    country character(2) DEFAULT 'US'::bpchar NOT NULL,
    phone character varying(50),
    is_default boolean DEFAULT false NOT NULL,
    handling_class character varying(20),
    delivery_instructions text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: address_id_seq; Type: SEQUENCE; Schema: identity; Owner: -
--

CREATE SEQUENCE identity.address_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: address_id_seq; Type: SEQUENCE OWNED BY; Schema: identity; Owner: -
--

ALTER SEQUENCE identity.address_id_seq OWNED BY identity.address.id;


--
-- Name: buyer_profile; Type: TABLE; Schema: identity; Owner: -
--

CREATE TABLE identity.buyer_profile (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    code character varying(26) NOT NULL,
    business_name character varying(255) NOT NULL,
    trading_name character varying(255),
    tax_id character varying(50),
    registration_number character varying(50),
    phone character varying(50),
    website character varying(255),
    industry character varying(100),
    employee_count integer,
    annual_revenue_minor bigint,
    currency character(3) DEFAULT 'USD'::bpchar NOT NULL,
    credit_limit_minor bigint,
    credit_terms_days integer,
    is_on_credit_hold boolean DEFAULT false NOT NULL,
    credit_hold_reason text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: buyer_profile_id_seq; Type: SEQUENCE; Schema: identity; Owner: -
--

CREATE SEQUENCE identity.buyer_profile_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: buyer_profile_id_seq; Type: SEQUENCE OWNED BY; Schema: identity; Owner: -
--

ALTER SEQUENCE identity.buyer_profile_id_seq OWNED BY identity.buyer_profile.id;


--
-- Name: credit_account; Type: TABLE; Schema: identity; Owner: -
--

CREATE TABLE identity.credit_account (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    buyer_profile_id bigint NOT NULL,
    limit_minor bigint DEFAULT 0 NOT NULL,
    currency character(3) DEFAULT 'USD'::bpchar NOT NULL,
    terms_days integer DEFAULT 30 NOT NULL,
    current_exposure_minor bigint DEFAULT 0 NOT NULL,
    available_minor bigint GENERATED ALWAYS AS ((limit_minor - current_exposure_minor)) STORED,
    is_on_hold boolean DEFAULT false NOT NULL,
    hold_reason text,
    hold_since timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: credit_account_id_seq; Type: SEQUENCE; Schema: identity; Owner: -
--

CREATE SEQUENCE identity.credit_account_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: credit_account_id_seq; Type: SEQUENCE OWNED BY; Schema: identity; Owner: -
--

ALTER SEQUENCE identity.credit_account_id_seq OWNED BY identity.credit_account.id;


--
-- Name: otp_code; Type: TABLE; Schema: identity; Owner: -
--

CREATE TABLE identity.otp_code (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    user_id bigint NOT NULL,
    otp_hash character varying(255) NOT NULL,
    purpose character varying(50) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone,
    attempts integer DEFAULT 0 NOT NULL,
    max_attempts integer DEFAULT 3 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: otp_code_id_seq; Type: SEQUENCE; Schema: identity; Owner: -
--

CREATE SEQUENCE identity.otp_code_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: otp_code_id_seq; Type: SEQUENCE OWNED BY; Schema: identity; Owner: -
--

ALTER SEQUENCE identity.otp_code_id_seq OWNED BY identity.otp_code.id;


--
-- Name: password_reset_token; Type: TABLE; Schema: identity; Owner: -
--

CREATE TABLE identity.password_reset_token (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    user_id bigint NOT NULL,
    token_hash character varying(255) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: password_reset_token_id_seq; Type: SEQUENCE; Schema: identity; Owner: -
--

CREATE SEQUENCE identity.password_reset_token_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: password_reset_token_id_seq; Type: SEQUENCE OWNED BY; Schema: identity; Owner: -
--

ALTER SEQUENCE identity.password_reset_token_id_seq OWNED BY identity.password_reset_token.id;


--
-- Name: permission; Type: TABLE; Schema: identity; Owner: -
--

CREATE TABLE identity.permission (
    id bigint NOT NULL,
    code character varying(100) NOT NULL,
    name character varying(100) NOT NULL,
    description text,
    resource character varying(50) NOT NULL,
    action character varying(50) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: permission_id_seq; Type: SEQUENCE; Schema: identity; Owner: -
--

CREATE SEQUENCE identity.permission_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: permission_id_seq; Type: SEQUENCE OWNED BY; Schema: identity; Owner: -
--

ALTER SEQUENCE identity.permission_id_seq OWNED BY identity.permission.id;


--
-- Name: price_list_assignment; Type: TABLE; Schema: identity; Owner: -
--

CREATE TABLE identity.price_list_assignment (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    buyer_profile_id bigint NOT NULL,
    price_list_id bigint NOT NULL,
    assigned_at timestamp with time zone DEFAULT now() NOT NULL,
    assigned_by bigint,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: price_list_assignment_id_seq; Type: SEQUENCE; Schema: identity; Owner: -
--

CREATE SEQUENCE identity.price_list_assignment_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: price_list_assignment_id_seq; Type: SEQUENCE OWNED BY; Schema: identity; Owner: -
--

ALTER SEQUENCE identity.price_list_assignment_id_seq OWNED BY identity.price_list_assignment.id;


--
-- Name: purchase_scope; Type: TABLE; Schema: identity; Owner: -
--

CREATE TABLE identity.purchase_scope (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    buyer_profile_id bigint NOT NULL,
    handling_class character varying(20) NOT NULL,
    granted_at timestamp with time zone DEFAULT now() NOT NULL,
    granted_by bigint,
    expires_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: purchase_scope_id_seq; Type: SEQUENCE; Schema: identity; Owner: -
--

CREATE SEQUENCE identity.purchase_scope_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: purchase_scope_id_seq; Type: SEQUENCE OWNED BY; Schema: identity; Owner: -
--

ALTER SEQUENCE identity.purchase_scope_id_seq OWNED BY identity.purchase_scope.id;


--
-- Name: refresh_token; Type: TABLE; Schema: identity; Owner: -
--

CREATE TABLE identity.refresh_token (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    user_id bigint NOT NULL,
    token_hash character varying(255) NOT NULL,
    family_id character varying(26) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    revoked_at timestamp with time zone,
    revoked_reason character varying(100),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    replaced_by_id bigint
);


--
-- Name: refresh_token_id_seq; Type: SEQUENCE; Schema: identity; Owner: -
--

CREATE SEQUENCE identity.refresh_token_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: refresh_token_id_seq; Type: SEQUENCE OWNED BY; Schema: identity; Owner: -
--

ALTER SEQUENCE identity.refresh_token_id_seq OWNED BY identity.refresh_token.id;


--
-- Name: role; Type: TABLE; Schema: identity; Owner: -
--

CREATE TABLE identity.role (
    id bigint NOT NULL,
    code character varying(50) NOT NULL,
    name character varying(100) NOT NULL,
    description text,
    is_system boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: role_id_seq; Type: SEQUENCE; Schema: identity; Owner: -
--

CREATE SEQUENCE identity.role_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: role_id_seq; Type: SEQUENCE OWNED BY; Schema: identity; Owner: -
--

ALTER SEQUENCE identity.role_id_seq OWNED BY identity.role.id;


--
-- Name: role_permission; Type: TABLE; Schema: identity; Owner: -
--

CREATE TABLE identity.role_permission (
    role_id bigint NOT NULL,
    permission_id bigint NOT NULL
);


--
-- Name: session; Type: TABLE; Schema: identity; Owner: -
--

CREATE TABLE identity.session (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    user_id bigint NOT NULL,
    refresh_token_id bigint,
    ip_address inet,
    user_agent text,
    expires_at timestamp with time zone NOT NULL,
    revoked_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: session_id_seq; Type: SEQUENCE; Schema: identity; Owner: -
--

CREATE SEQUENCE identity.session_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: session_id_seq; Type: SEQUENCE OWNED BY; Schema: identity; Owner: -
--

ALTER SEQUENCE identity.session_id_seq OWNED BY identity.session.id;


--
-- Name: supplier_profile; Type: TABLE; Schema: identity; Owner: -
--

CREATE TABLE identity.supplier_profile (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    code character varying(26) NOT NULL,
    company_name character varying(255) NOT NULL,
    trading_name character varying(255),
    tax_id character varying(50),
    registration_number character varying(50),
    phone character varying(50),
    email character varying(255),
    website character varying(255),
    address_line1 character varying(255),
    address_line2 character varying(255),
    city character varying(100),
    state_province character varying(100),
    postal_code character varying(20),
    country character(2) DEFAULT 'US'::bpchar NOT NULL,
    contact_person character varying(255),
    contact_phone character varying(50),
    contact_email character varying(255),
    is_approved boolean DEFAULT false NOT NULL,
    approved_at timestamp with time zone,
    approved_by bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: supplier_profile_id_seq; Type: SEQUENCE; Schema: identity; Owner: -
--

CREATE SEQUENCE identity.supplier_profile_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: supplier_profile_id_seq; Type: SEQUENCE OWNED BY; Schema: identity; Owner: -
--

ALTER SEQUENCE identity.supplier_profile_id_seq OWNED BY identity.supplier_profile.id;


--
-- Name: user; Type: TABLE; Schema: identity; Owner: -
--

CREATE TABLE identity."user" (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    email character varying(255) NOT NULL,
    password_hash text NOT NULL,
    user_type identity.user_type DEFAULT 'buyer'::identity.user_type NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    is_verified boolean DEFAULT false NOT NULL,
    last_login_at timestamp with time zone,
    failed_login_attempts integer DEFAULT 0 NOT NULL,
    locked_until timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: user_id_seq; Type: SEQUENCE; Schema: identity; Owner: -
--

CREATE SEQUENCE identity.user_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_id_seq; Type: SEQUENCE OWNED BY; Schema: identity; Owner: -
--

ALTER SEQUENCE identity.user_id_seq OWNED BY identity."user".id;


--
-- Name: user_role; Type: TABLE; Schema: identity; Owner: -
--

CREATE TABLE identity.user_role (
    user_id bigint NOT NULL,
    role_id bigint NOT NULL,
    assigned_at timestamp with time zone DEFAULT now() NOT NULL,
    assigned_by bigint
);


--
-- Name: verification_application; Type: TABLE; Schema: identity; Owner: -
--

CREATE TABLE identity.verification_application (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    buyer_profile_id bigint NOT NULL,
    status identity.verification_status DEFAULT 'pending'::identity.verification_status NOT NULL,
    submitted_at timestamp with time zone DEFAULT now() NOT NULL,
    decided_at timestamp with time zone,
    decided_by bigint,
    decision_reason text,
    rejection_reason text,
    licence_number character varying(100),
    licence_expiry date,
    trading_name character varying(255),
    business_address_line1 character varying(255),
    business_address_line2 character varying(255),
    business_city character varying(100),
    business_state_province character varying(100),
    business_postal_code character varying(20),
    business_country character(2) DEFAULT 'US'::bpchar NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: verification_application_id_seq; Type: SEQUENCE; Schema: identity; Owner: -
--

CREATE SEQUENCE identity.verification_application_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: verification_application_id_seq; Type: SEQUENCE OWNED BY; Schema: identity; Owner: -
--

ALTER SEQUENCE identity.verification_application_id_seq OWNED BY identity.verification_application.id;


--
-- Name: verification_document; Type: TABLE; Schema: identity; Owner: -
--

CREATE TABLE identity.verification_document (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    verification_application_id bigint NOT NULL,
    document_type character varying(50) NOT NULL,
    file_name character varying(255) NOT NULL,
    file_path character varying(500) NOT NULL,
    file_size bigint NOT NULL,
    mime_type character varying(100) NOT NULL,
    uploaded_at timestamp with time zone DEFAULT now() NOT NULL,
    uploaded_by bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: verification_document_id_seq; Type: SEQUENCE; Schema: identity; Owner: -
--

CREATE SEQUENCE identity.verification_document_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: verification_document_id_seq; Type: SEQUENCE OWNED BY; Schema: identity; Owner: -
--

ALTER SEQUENCE identity.verification_document_id_seq OWNED BY identity.verification_document.id;


--
-- Name: lot; Type: TABLE; Schema: inventory; Owner: -
--

CREATE TABLE inventory.lot (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    stock_level_id bigint NOT NULL,
    lot_number character varying(100) NOT NULL,
    initial_quantity integer NOT NULL,
    available_quantity integer NOT NULL,
    reserved_quantity integer DEFAULT 0 NOT NULL,
    status inventory.lot_status DEFAULT 'active'::inventory.lot_status NOT NULL,
    is_quarantined boolean DEFAULT false NOT NULL,
    production_date date,
    expires_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT lot_available_quantity_check CHECK ((available_quantity >= 0)),
    CONSTRAINT lot_initial_quantity_check CHECK ((initial_quantity > 0)),
    CONSTRAINT lot_reserved_quantity_check CHECK ((reserved_quantity >= 0))
);


--
-- Name: lot_id_seq; Type: SEQUENCE; Schema: inventory; Owner: -
--

CREATE SEQUENCE inventory.lot_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: lot_id_seq; Type: SEQUENCE OWNED BY; Schema: inventory; Owner: -
--

ALTER SEQUENCE inventory.lot_id_seq OWNED BY inventory.lot.id;


--
-- Name: quarantine_record; Type: TABLE; Schema: inventory; Owner: -
--

CREATE TABLE inventory.quarantine_record (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    lot_id bigint NOT NULL,
    reason text NOT NULL,
    status inventory.quarantine_status DEFAULT 'quarantined'::inventory.quarantine_status NOT NULL,
    adjusted_quantity integer DEFAULT 0 NOT NULL,
    adjusted_by bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: quarantine_record_id_seq; Type: SEQUENCE; Schema: inventory; Owner: -
--

CREATE SEQUENCE inventory.quarantine_record_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: quarantine_record_id_seq; Type: SEQUENCE OWNED BY; Schema: inventory; Owner: -
--

ALTER SEQUENCE inventory.quarantine_record_id_seq OWNED BY inventory.quarantine_record.id;


--
-- Name: reservation; Type: TABLE; Schema: inventory; Owner: -
--

CREATE TABLE inventory.reservation (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    lot_id bigint NOT NULL,
    order_line_id bigint,
    request_id character varying(100) NOT NULL,
    quantity integer NOT NULL,
    status inventory.reservation_status DEFAULT 'reserved'::inventory.reservation_status NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT reservation_quantity_check CHECK ((quantity > 0))
);


--
-- Name: reservation_id_seq; Type: SEQUENCE; Schema: inventory; Owner: -
--

CREATE SEQUENCE inventory.reservation_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: reservation_id_seq; Type: SEQUENCE OWNED BY; Schema: inventory; Owner: -
--

ALTER SEQUENCE inventory.reservation_id_seq OWNED BY inventory.reservation.id;


--
-- Name: stock_adjustment; Type: TABLE; Schema: inventory; Owner: -
--

CREATE TABLE inventory.stock_adjustment (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    lot_id bigint NOT NULL,
    quantity_delta integer NOT NULL,
    previous_quantity integer NOT NULL,
    new_quantity integer NOT NULL,
    reason_code character varying(100) NOT NULL,
    reason text NOT NULL,
    adjusted_by bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: stock_adjustment_id_seq; Type: SEQUENCE; Schema: inventory; Owner: -
--

CREATE SEQUENCE inventory.stock_adjustment_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: stock_adjustment_id_seq; Type: SEQUENCE OWNED BY; Schema: inventory; Owner: -
--

ALTER SEQUENCE inventory.stock_adjustment_id_seq OWNED BY inventory.stock_adjustment.id;


--
-- Name: stock_level; Type: TABLE; Schema: inventory; Owner: -
--

CREATE TABLE inventory.stock_level (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    product_id bigint NOT NULL,
    supplier_id bigint NOT NULL,
    available_quantity integer DEFAULT 0 NOT NULL,
    reserved_quantity integer DEFAULT 0 NOT NULL,
    total_quantity integer DEFAULT 0 NOT NULL,
    safety_stock integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT stock_level_available_quantity_check CHECK ((available_quantity >= 0)),
    CONSTRAINT stock_level_reserved_quantity_check CHECK ((reserved_quantity >= 0)),
    CONSTRAINT stock_level_safety_stock_check CHECK ((safety_stock >= 0)),
    CONSTRAINT stock_level_total_quantity_check CHECK ((total_quantity >= 0))
);


--
-- Name: stock_level_id_seq; Type: SEQUENCE; Schema: inventory; Owner: -
--

CREATE SEQUENCE inventory.stock_level_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: stock_level_id_seq; Type: SEQUENCE OWNED BY; Schema: inventory; Owner: -
--

ALTER SEQUENCE inventory.stock_level_id_seq OWNED BY inventory.stock_level.id;


--
-- Name: credit_account; Type: TABLE; Schema: payments; Owner: -
--

CREATE TABLE payments.credit_account (
    buyer_id bigint NOT NULL,
    credit_limit_minor bigint DEFAULT 0 NOT NULL,
    terms character varying(64) DEFAULT 'prepaid'::character varying NOT NULL,
    on_hold boolean DEFAULT false NOT NULL,
    hold_reason text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT credit_account_credit_limit_minor_check CHECK ((credit_limit_minor >= 0))
);


--
-- Name: payment_attempt; Type: TABLE; Schema: payments; Owner: -
--

CREATE TABLE payments.payment_attempt (
    id bigint NOT NULL,
    intent_id bigint NOT NULL,
    result character varying(16) NOT NULL,
    gateway_ref character varying(128),
    note text,
    recorded_by bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT payment_attempt_result_check CHECK (((result)::text = ANY ((ARRAY['succeeded'::character varying, 'failed'::character varying])::text[])))
);


--
-- Name: payment_attempt_id_seq; Type: SEQUENCE; Schema: payments; Owner: -
--

CREATE SEQUENCE payments.payment_attempt_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: payment_attempt_id_seq; Type: SEQUENCE OWNED BY; Schema: payments; Owner: -
--

ALTER SEQUENCE payments.payment_attempt_id_seq OWNED BY payments.payment_attempt.id;


--
-- Name: payment_intent; Type: TABLE; Schema: payments; Owner: -
--

CREATE TABLE payments.payment_intent (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    buyer_id bigint NOT NULL,
    order_id bigint NOT NULL,
    order_code character varying(26) NOT NULL,
    payment_method_id bigint,
    amount_minor bigint NOT NULL,
    currency character(3) DEFAULT 'USD'::bpchar NOT NULL,
    status character varying(32) DEFAULT 'requires_action'::character varying NOT NULL,
    idem_key character varying(100) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT payment_intent_amount_minor_check CHECK ((amount_minor >= 0)),
    CONSTRAINT payment_intent_status_check CHECK (((status)::text = ANY ((ARRAY['requires_action'::character varying, 'processing'::character varying, 'succeeded'::character varying, 'failed'::character varying, 'cancelled'::character varying])::text[])))
);


--
-- Name: payment_intent_id_seq; Type: SEQUENCE; Schema: payments; Owner: -
--

CREATE SEQUENCE payments.payment_intent_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: payment_intent_id_seq; Type: SEQUENCE OWNED BY; Schema: payments; Owner: -
--

ALTER SEQUENCE payments.payment_intent_id_seq OWNED BY payments.payment_intent.id;


--
-- Name: payment_method; Type: TABLE; Schema: payments; Owner: -
--

CREATE TABLE payments.payment_method (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    buyer_id bigint NOT NULL,
    method_type character varying(32) NOT NULL,
    display_name character varying(128) NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT payment_method_method_type_check CHECK (((method_type)::text = ANY ((ARRAY['credit_terms'::character varying, 'bank_transfer'::character varying, 'cod'::character varying, 'card'::character varying])::text[])))
);


--
-- Name: payment_method_id_seq; Type: SEQUENCE; Schema: payments; Owner: -
--

CREATE SEQUENCE payments.payment_method_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: payment_method_id_seq; Type: SEQUENCE OWNED BY; Schema: payments; Owner: -
--

ALTER SEQUENCE payments.payment_method_id_seq OWNED BY payments.payment_method.id;


--
-- Name: payment_refund; Type: TABLE; Schema: payments; Owner: -
--

CREATE TABLE payments.payment_refund (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    intent_id bigint NOT NULL,
    amount_minor bigint NOT NULL,
    currency character(3) DEFAULT 'USD'::bpchar NOT NULL,
    reason text NOT NULL,
    status character varying(16) DEFAULT 'pending_approval'::character varying NOT NULL,
    requested_by bigint,
    approved_by bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT payment_refund_amount_minor_check CHECK ((amount_minor > 0)),
    CONSTRAINT payment_refund_reason_check CHECK ((char_length(reason) > 0)),
    CONSTRAINT payment_refund_status_check CHECK (((status)::text = ANY ((ARRAY['pending_approval'::character varying, 'approved'::character varying, 'rejected'::character varying, 'applied'::character varying])::text[])))
);


--
-- Name: payment_refund_id_seq; Type: SEQUENCE; Schema: payments; Owner: -
--

CREATE SEQUENCE payments.payment_refund_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: payment_refund_id_seq; Type: SEQUENCE OWNED BY; Schema: payments; Owner: -
--

ALTER SEQUENCE payments.payment_refund_id_seq OWNED BY payments.payment_refund.id;


--
-- Name: webhook_event; Type: TABLE; Schema: payments; Owner: -
--

CREATE TABLE payments.webhook_event (
    id bigint NOT NULL,
    provider character varying(64) NOT NULL,
    provider_event_id character varying(128) NOT NULL,
    event_type character varying(64) NOT NULL,
    payload_hash character(64) NOT NULL,
    intent_code character varying(26),
    status character varying(16) DEFAULT 'received'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT webhook_event_status_check CHECK (((status)::text = ANY ((ARRAY['received'::character varying, 'applied'::character varying, 'unmatched'::character varying, 'rejected'::character varying])::text[])))
);


--
-- Name: webhook_event_id_seq; Type: SEQUENCE; Schema: payments; Owner: -
--

CREATE SEQUENCE payments.webhook_event_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: webhook_event_id_seq; Type: SEQUENCE OWNED BY; Schema: payments; Owner: -
--

ALTER SEQUENCE payments.webhook_event_id_seq OWNED BY payments.webhook_event.id;


--
-- Name: audit_log; Type: TABLE; Schema: platform; Owner: -
--

CREATE TABLE platform.audit_log (
    id bigint NOT NULL,
    actor_id bigint,
    actor_type character varying(20) NOT NULL,
    action character varying(100) NOT NULL,
    resource_type character varying(100) NOT NULL,
    resource_id bigint,
    resource_code character varying(50),
    metadata jsonb,
    ip_address inet,
    user_agent text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT audit_log_actor_type_check CHECK (((actor_type)::text = ANY ((ARRAY['user'::character varying, 'buyer'::character varying, 'system'::character varying])::text[])))
);


--
-- Name: audit_log_id_seq; Type: SEQUENCE; Schema: platform; Owner: -
--

CREATE SEQUENCE platform.audit_log_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: audit_log_id_seq; Type: SEQUENCE OWNED BY; Schema: platform; Owner: -
--

ALTER SEQUENCE platform.audit_log_id_seq OWNED BY platform.audit_log.id;


--
-- Name: feature_flag; Type: TABLE; Schema: platform; Owner: -
--

CREATE TABLE platform.feature_flag (
    id bigint NOT NULL,
    key character varying(100) NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    enabled boolean DEFAULT false NOT NULL,
    updated_by bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: feature_flag_id_seq; Type: SEQUENCE; Schema: platform; Owner: -
--

CREATE SEQUENCE platform.feature_flag_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: feature_flag_id_seq; Type: SEQUENCE OWNED BY; Schema: platform; Owner: -
--

ALTER SEQUENCE platform.feature_flag_id_seq OWNED BY platform.feature_flag.id;


--
-- Name: health_check; Type: TABLE; Schema: platform; Owner: -
--

CREATE TABLE platform.health_check (
    id bigint NOT NULL,
    checked_at timestamp with time zone DEFAULT now() NOT NULL,
    status text DEFAULT 'ok'::text NOT NULL
);


--
-- Name: health_check_id_seq; Type: SEQUENCE; Schema: platform; Owner: -
--

CREATE SEQUENCE platform.health_check_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: health_check_id_seq; Type: SEQUENCE OWNED BY; Schema: platform; Owner: -
--

ALTER SEQUENCE platform.health_check_id_seq OWNED BY platform.health_check.id;


--
-- Name: integration_traffic; Type: TABLE; Schema: platform; Owner: -
--

CREATE TABLE platform.integration_traffic (
    id bigint NOT NULL,
    direction character varying(10) NOT NULL,
    provider character varying(100) NOT NULL,
    endpoint character varying(500) NOT NULL,
    method character varying(10),
    status_code integer,
    payload_hash character varying(64),
    request_headers jsonb,
    response_headers jsonb,
    error_message text,
    duration_ms integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT integration_traffic_direction_check CHECK (((direction)::text = ANY ((ARRAY['inbound'::character varying, 'outbound'::character varying])::text[])))
);


--
-- Name: integration_traffic_id_seq; Type: SEQUENCE; Schema: platform; Owner: -
--

CREATE SEQUENCE platform.integration_traffic_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: integration_traffic_id_seq; Type: SEQUENCE OWNED BY; Schema: platform; Owner: -
--

ALTER SEQUENCE platform.integration_traffic_id_seq OWNED BY platform.integration_traffic.id;


--
-- Name: media_upload; Type: TABLE; Schema: platform; Owner: -
--

CREATE TABLE platform.media_upload (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    filename character varying(500) NOT NULL,
    content_type character varying(100) NOT NULL,
    size_bytes bigint,
    upload_url character varying(1000) NOT NULL,
    download_url character varying(1000),
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_by bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT media_upload_status_check CHECK (((status)::text = ANY ((ARRAY['pending'::character varying, 'uploaded'::character varying, 'failed'::character varying, 'expired'::character varying])::text[])))
);


--
-- Name: media_upload_id_seq; Type: SEQUENCE; Schema: platform; Owner: -
--

CREATE SEQUENCE platform.media_upload_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: media_upload_id_seq; Type: SEQUENCE OWNED BY; Schema: platform; Owner: -
--

ALTER SEQUENCE platform.media_upload_id_seq OWNED BY platform.media_upload.id;


--
-- Name: notification; Type: TABLE; Schema: platform; Owner: -
--

CREATE TABLE platform.notification (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    template_id bigint,
    recipient_type character varying(20) NOT NULL,
    recipient_id bigint NOT NULL,
    channel character varying(20) NOT NULL,
    recipient_address character varying(500) NOT NULL,
    subject character varying(500),
    body text NOT NULL,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    provider_message_id character varying(255),
    provider_response text,
    sent_at timestamp with time zone,
    delivered_at timestamp with time zone,
    failed_at timestamp with time zone,
    error_message text,
    retry_count integer DEFAULT 0 NOT NULL,
    metadata jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT notification_channel_check CHECK (((channel)::text = ANY ((ARRAY['email'::character varying, 'sms'::character varying])::text[]))),
    CONSTRAINT notification_recipient_type_check CHECK (((recipient_type)::text = ANY ((ARRAY['buyer'::character varying, 'user'::character varying])::text[]))),
    CONSTRAINT notification_status_check CHECK (((status)::text = ANY ((ARRAY['pending'::character varying, 'sent'::character varying, 'delivered'::character varying, 'failed'::character varying, 'bounced'::character varying])::text[])))
);


--
-- Name: notification_event; Type: TABLE; Schema: platform; Owner: -
--

CREATE TABLE platform.notification_event (
    id bigint NOT NULL,
    notification_id bigint NOT NULL,
    event_type character varying(30) NOT NULL,
    provider_event_id character varying(255),
    metadata jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT notification_event_event_type_check CHECK (((event_type)::text = ANY ((ARRAY['created'::character varying, 'sent'::character varying, 'delivered'::character varying, 'failed'::character varying, 'bounced'::character varying, 'retry'::character varying])::text[])))
);


--
-- Name: notification_event_id_seq; Type: SEQUENCE; Schema: platform; Owner: -
--

CREATE SEQUENCE platform.notification_event_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: notification_event_id_seq; Type: SEQUENCE OWNED BY; Schema: platform; Owner: -
--

ALTER SEQUENCE platform.notification_event_id_seq OWNED BY platform.notification_event.id;


--
-- Name: notification_id_seq; Type: SEQUENCE; Schema: platform; Owner: -
--

CREATE SEQUENCE platform.notification_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: notification_id_seq; Type: SEQUENCE OWNED BY; Schema: platform; Owner: -
--

ALTER SEQUENCE platform.notification_id_seq OWNED BY platform.notification.id;


--
-- Name: notification_template; Type: TABLE; Schema: platform; Owner: -
--

CREATE TABLE platform.notification_template (
    id bigint NOT NULL,
    code character varying(50) NOT NULL,
    name character varying(200) NOT NULL,
    channel character varying(20) NOT NULL,
    subject character varying(500),
    body_template text NOT NULL,
    variables jsonb DEFAULT '[]'::jsonb NOT NULL,
    status character varying(20) DEFAULT 'active'::character varying NOT NULL,
    created_by bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT notification_template_channel_check CHECK (((channel)::text = ANY ((ARRAY['email'::character varying, 'sms'::character varying])::text[]))),
    CONSTRAINT notification_template_status_check CHECK (((status)::text = ANY ((ARRAY['active'::character varying, 'inactive'::character varying])::text[])))
);


--
-- Name: notification_template_id_seq; Type: SEQUENCE; Schema: platform; Owner: -
--

CREATE SEQUENCE platform.notification_template_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: notification_template_id_seq; Type: SEQUENCE OWNED BY; Schema: platform; Owner: -
--

ALTER SEQUENCE platform.notification_template_id_seq OWNED BY platform.notification_template.id;


--
-- Name: reference_region; Type: TABLE; Schema: platform; Owner: -
--

CREATE TABLE platform.reference_region (
    id bigint NOT NULL,
    code character varying(20) NOT NULL,
    name character varying(255) NOT NULL,
    country character varying(100) DEFAULT 'VN'::character varying NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: reference_region_id_seq; Type: SEQUENCE; Schema: platform; Owner: -
--

CREATE SEQUENCE platform.reference_region_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: reference_region_id_seq; Type: SEQUENCE OWNED BY; Schema: platform; Owner: -
--

ALTER SEQUENCE platform.reference_region_id_seq OWNED BY platform.reference_region.id;


--
-- Name: price_list; Type: TABLE; Schema: pricing; Owner: -
--

CREATE TABLE pricing.price_list (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    status pricing.price_list_status DEFAULT 'draft'::pricing.price_list_status NOT NULL,
    price_type pricing.price_type DEFAULT 'standard'::pricing.price_type NOT NULL,
    currency character(3) DEFAULT 'USD'::bpchar NOT NULL,
    effective_from timestamp with time zone DEFAULT now() NOT NULL,
    effective_to timestamp with time zone,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: price_list_assignment; Type: TABLE; Schema: pricing; Owner: -
--

CREATE TABLE pricing.price_list_assignment (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    price_list_id bigint NOT NULL,
    buyer_profile_id bigint NOT NULL,
    assigned_by bigint,
    effective_from timestamp with time zone DEFAULT now() NOT NULL,
    effective_to timestamp with time zone,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: price_list_assignment_id_seq; Type: SEQUENCE; Schema: pricing; Owner: -
--

CREATE SEQUENCE pricing.price_list_assignment_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: price_list_assignment_id_seq; Type: SEQUENCE OWNED BY; Schema: pricing; Owner: -
--

ALTER SEQUENCE pricing.price_list_assignment_id_seq OWNED BY pricing.price_list_assignment.id;


--
-- Name: price_list_id_seq; Type: SEQUENCE; Schema: pricing; Owner: -
--

CREATE SEQUENCE pricing.price_list_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: price_list_id_seq; Type: SEQUENCE OWNED BY; Schema: pricing; Owner: -
--

ALTER SEQUENCE pricing.price_list_id_seq OWNED BY pricing.price_list.id;


--
-- Name: price_list_item; Type: TABLE; Schema: pricing; Owner: -
--

CREATE TABLE pricing.price_list_item (
    id bigint NOT NULL,
    price_list_id bigint NOT NULL,
    product_id bigint NOT NULL,
    unit_id bigint NOT NULL,
    currency character(3) DEFAULT 'USD'::bpchar NOT NULL,
    price_minor bigint NOT NULL,
    min_quantity integer DEFAULT 1 NOT NULL,
    max_quantity integer,
    effective_from timestamp with time zone DEFAULT now() NOT NULL,
    effective_to timestamp with time zone,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: price_list_item_id_seq; Type: SEQUENCE; Schema: pricing; Owner: -
--

CREATE SEQUENCE pricing.price_list_item_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: price_list_item_id_seq; Type: SEQUENCE OWNED BY; Schema: pricing; Owner: -
--

ALTER SEQUENCE pricing.price_list_item_id_seq OWNED BY pricing.price_list_item.id;


--
-- Name: quantity_tier; Type: TABLE; Schema: pricing; Owner: -
--

CREATE TABLE pricing.quantity_tier (
    id bigint NOT NULL,
    price_list_item_id bigint NOT NULL,
    min_quantity integer NOT NULL,
    max_quantity integer,
    price_minor bigint NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: quantity_tier_id_seq; Type: SEQUENCE; Schema: pricing; Owner: -
--

CREATE SEQUENCE pricing.quantity_tier_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: quantity_tier_id_seq; Type: SEQUENCE OWNED BY; Schema: pricing; Owner: -
--

ALTER SEQUENCE pricing.quantity_tier_id_seq OWNED BY pricing.quantity_tier.id;


--
-- Name: promotion; Type: TABLE; Schema: promotions; Owner: -
--

CREATE TABLE promotions.promotion (
    id bigint NOT NULL,
    code character varying(64) NOT NULL,
    name character varying(255) NOT NULL,
    kind character varying(16) NOT NULL,
    value_minor bigint NOT NULL,
    currency character(3) DEFAULT 'USD'::bpchar NOT NULL,
    status character varying(16) DEFAULT 'draft'::character varying NOT NULL,
    valid_from timestamp with time zone,
    valid_to timestamp with time zone,
    max_redemptions integer,
    redeemed_count integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT promotion_check CHECK (((valid_to IS NULL) OR (valid_from IS NULL) OR (valid_to >= valid_from))),
    CONSTRAINT promotion_kind_check CHECK (((kind)::text = ANY ((ARRAY['percent'::character varying, 'fixed'::character varying])::text[]))),
    CONSTRAINT promotion_max_redemptions_check CHECK (((max_redemptions IS NULL) OR (max_redemptions > 0))),
    CONSTRAINT promotion_redeemed_count_check CHECK ((redeemed_count >= 0)),
    CONSTRAINT promotion_status_check CHECK (((status)::text = ANY ((ARRAY['draft'::character varying, 'published'::character varying, 'archived'::character varying])::text[]))),
    CONSTRAINT promotion_value_minor_check CHECK ((value_minor > 0))
);


--
-- Name: promotion_id_seq; Type: SEQUENCE; Schema: promotions; Owner: -
--

CREATE SEQUENCE promotions.promotion_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: promotion_id_seq; Type: SEQUENCE OWNED BY; Schema: promotions; Owner: -
--

ALTER SEQUENCE promotions.promotion_id_seq OWNED BY promotions.promotion.id;


--
-- Name: voucher_redemption; Type: TABLE; Schema: promotions; Owner: -
--

CREATE TABLE promotions.voucher_redemption (
    id bigint NOT NULL,
    promotion_id bigint NOT NULL,
    buyer_id bigint NOT NULL,
    order_id bigint NOT NULL,
    amount_minor bigint NOT NULL,
    currency character(3) DEFAULT 'USD'::bpchar NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT voucher_redemption_amount_minor_check CHECK ((amount_minor >= 0))
);


--
-- Name: voucher_redemption_id_seq; Type: SEQUENCE; Schema: promotions; Owner: -
--

CREATE SEQUENCE promotions.voucher_redemption_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: voucher_redemption_id_seq; Type: SEQUENCE OWNED BY; Schema: promotions; Owner: -
--

ALTER SEQUENCE promotions.voucher_redemption_id_seq OWNED BY promotions.voucher_redemption.id;


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


--
-- Name: supplier_contract; Type: TABLE; Schema: suppliers; Owner: -
--

CREATE TABLE suppliers.supplier_contract (
    id bigint NOT NULL,
    supplier_id bigint NOT NULL,
    version integer NOT NULL,
    terms text NOT NULL,
    valid_from timestamp with time zone,
    valid_to timestamp with time zone,
    is_current boolean DEFAULT true NOT NULL,
    created_by bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT supplier_contract_check CHECK (((valid_to IS NULL) OR (valid_from IS NULL) OR (valid_to >= valid_from))),
    CONSTRAINT supplier_contract_terms_check CHECK ((char_length(terms) > 0)),
    CONSTRAINT supplier_contract_version_check CHECK ((version > 0))
);


--
-- Name: supplier_contract_id_seq; Type: SEQUENCE; Schema: suppliers; Owner: -
--

CREATE SEQUENCE suppliers.supplier_contract_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: supplier_contract_id_seq; Type: SEQUENCE OWNED BY; Schema: suppliers; Owner: -
--

ALTER SEQUENCE suppliers.supplier_contract_id_seq OWNED BY suppliers.supplier_contract.id;


--
-- Name: supplier_profile; Type: TABLE; Schema: suppliers; Owner: -
--

CREATE TABLE suppliers.supplier_profile (
    id bigint NOT NULL,
    code character varying(26) NOT NULL,
    company_name character varying(255) NOT NULL,
    contact_name character varying(255),
    contact_email character varying(255),
    contact_phone character varying(64),
    status character varying(16) DEFAULT 'applied'::character varying NOT NULL,
    user_id bigint,
    approved_by bigint,
    approved_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT supplier_profile_status_check CHECK (((status)::text = ANY ((ARRAY['applied'::character varying, 'approved'::character varying, 'rejected'::character varying, 'suspended'::character varying])::text[])))
);


--
-- Name: supplier_profile_id_seq; Type: SEQUENCE; Schema: suppliers; Owner: -
--

CREATE SEQUENCE suppliers.supplier_profile_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: supplier_profile_id_seq; Type: SEQUENCE OWNED BY; Schema: suppliers; Owner: -
--

ALTER SEQUENCE suppliers.supplier_profile_id_seq OWNED BY suppliers.supplier_profile.id;


--
-- Name: budget id; Type: DEFAULT; Schema: ai; Owner: -
--

ALTER TABLE ONLY ai.budget ALTER COLUMN id SET DEFAULT nextval('ai.budget_id_seq'::regclass);


--
-- Name: model_routing id; Type: DEFAULT; Schema: ai; Owner: -
--

ALTER TABLE ONLY ai.model_routing ALTER COLUMN id SET DEFAULT nextval('ai.model_routing_id_seq'::regclass);


--
-- Name: prompt id; Type: DEFAULT; Schema: ai; Owner: -
--

ALTER TABLE ONLY ai.prompt ALTER COLUMN id SET DEFAULT nextval('ai.prompt_id_seq'::regclass);


--
-- Name: prompt_version id; Type: DEFAULT; Schema: ai; Owner: -
--

ALTER TABLE ONLY ai.prompt_version ALTER COLUMN id SET DEFAULT nextval('ai.prompt_version_id_seq'::regclass);


--
-- Name: proposal id; Type: DEFAULT; Schema: ai; Owner: -
--

ALTER TABLE ONLY ai.proposal ALTER COLUMN id SET DEFAULT nextval('ai.proposal_id_seq'::regclass);


--
-- Name: usage_record id; Type: DEFAULT; Schema: ai; Owner: -
--

ALTER TABLE ONLY ai.usage_record ALTER COLUMN id SET DEFAULT nextval('ai.usage_record_id_seq'::regclass);


--
-- Name: attribute id; Type: DEFAULT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.attribute ALTER COLUMN id SET DEFAULT nextval('catalog.attribute_id_seq'::regclass);


--
-- Name: attribute_value id; Type: DEFAULT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.attribute_value ALTER COLUMN id SET DEFAULT nextval('catalog.attribute_value_id_seq'::regclass);


--
-- Name: brand id; Type: DEFAULT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.brand ALTER COLUMN id SET DEFAULT nextval('catalog.brand_id_seq'::regclass);


--
-- Name: category id; Type: DEFAULT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.category ALTER COLUMN id SET DEFAULT nextval('catalog.category_id_seq'::regclass);


--
-- Name: handling_class id; Type: DEFAULT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.handling_class ALTER COLUMN id SET DEFAULT nextval('catalog.handling_class_id_seq'::regclass);


--
-- Name: product id; Type: DEFAULT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product ALTER COLUMN id SET DEFAULT nextval('catalog.product_id_seq'::regclass);


--
-- Name: product_attribute id; Type: DEFAULT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product_attribute ALTER COLUMN id SET DEFAULT nextval('catalog.product_attribute_id_seq'::regclass);


--
-- Name: product_media id; Type: DEFAULT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product_media ALTER COLUMN id SET DEFAULT nextval('catalog.product_media_id_seq'::regclass);


--
-- Name: product_unit id; Type: DEFAULT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product_unit ALTER COLUMN id SET DEFAULT nextval('catalog.product_unit_id_seq'::regclass);


--
-- Name: supplier id; Type: DEFAULT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.supplier ALTER COLUMN id SET DEFAULT nextval('catalog.supplier_id_seq'::regclass);


--
-- Name: unit id; Type: DEFAULT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.unit ALTER COLUMN id SET DEFAULT nextval('catalog.unit_id_seq'::regclass);


--
-- Name: article id; Type: DEFAULT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.article ALTER COLUMN id SET DEFAULT nextval('cms.article_id_seq'::regclass);


--
-- Name: banner id; Type: DEFAULT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.banner ALTER COLUMN id SET DEFAULT nextval('cms.banner_id_seq'::regclass);


--
-- Name: contact_enquiry id; Type: DEFAULT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.contact_enquiry ALTER COLUMN id SET DEFAULT nextval('cms.contact_enquiry_id_seq'::regclass);


--
-- Name: faq id; Type: DEFAULT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.faq ALTER COLUMN id SET DEFAULT nextval('cms.faq_id_seq'::regclass);


--
-- Name: homepage_layout id; Type: DEFAULT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.homepage_layout ALTER COLUMN id SET DEFAULT nextval('cms.homepage_layout_id_seq'::regclass);


--
-- Name: legal_document_version id; Type: DEFAULT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.legal_document_version ALTER COLUMN id SET DEFAULT nextval('cms.legal_document_version_id_seq'::regclass);


--
-- Name: menu_item id; Type: DEFAULT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.menu_item ALTER COLUMN id SET DEFAULT nextval('cms.menu_item_id_seq'::regclass);


--
-- Name: newsletter_subscriber id; Type: DEFAULT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.newsletter_subscriber ALTER COLUMN id SET DEFAULT nextval('cms.newsletter_subscriber_id_seq'::regclass);


--
-- Name: page id; Type: DEFAULT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.page ALTER COLUMN id SET DEFAULT nextval('cms.page_id_seq'::regclass);


--
-- Name: seo_settings id; Type: DEFAULT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.seo_settings ALTER COLUMN id SET DEFAULT nextval('cms.seo_settings_id_seq'::regclass);


--
-- Name: seo_template id; Type: DEFAULT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.seo_template ALTER COLUMN id SET DEFAULT nextval('cms.seo_template_id_seq'::regclass);


--
-- Name: setting id; Type: DEFAULT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.setting ALTER COLUMN id SET DEFAULT nextval('cms.setting_id_seq'::regclass);


--
-- Name: cart id; Type: DEFAULT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.cart ALTER COLUMN id SET DEFAULT nextval('commerce.cart_id_seq'::regclass);


--
-- Name: cart_line id; Type: DEFAULT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.cart_line ALTER COLUMN id SET DEFAULT nextval('commerce.cart_line_id_seq'::regclass);


--
-- Name: checkout_idempotency id; Type: DEFAULT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.checkout_idempotency ALTER COLUMN id SET DEFAULT nextval('commerce.checkout_idempotency_id_seq'::regclass);


--
-- Name: credit_note id; Type: DEFAULT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.credit_note ALTER COLUMN id SET DEFAULT nextval('commerce.credit_note_id_seq'::regclass);


--
-- Name: invoice id; Type: DEFAULT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.invoice ALTER COLUMN id SET DEFAULT nextval('commerce.invoice_id_seq'::regclass);


--
-- Name: order id; Type: DEFAULT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce."order" ALTER COLUMN id SET DEFAULT nextval('commerce.order_id_seq'::regclass);


--
-- Name: order_history id; Type: DEFAULT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.order_history ALTER COLUMN id SET DEFAULT nextval('commerce.order_history_id_seq'::regclass);


--
-- Name: order_line id; Type: DEFAULT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.order_line ALTER COLUMN id SET DEFAULT nextval('commerce.order_line_id_seq'::regclass);


--
-- Name: return_line id; Type: DEFAULT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.return_line ALTER COLUMN id SET DEFAULT nextval('commerce.return_line_id_seq'::regclass);


--
-- Name: return_request id; Type: DEFAULT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.return_request ALTER COLUMN id SET DEFAULT nextval('commerce.return_request_id_seq'::regclass);


--
-- Name: shipment id; Type: DEFAULT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.shipment ALTER COLUMN id SET DEFAULT nextval('commerce.shipment_id_seq'::regclass);


--
-- Name: shipment_line id; Type: DEFAULT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.shipment_line ALTER COLUMN id SET DEFAULT nextval('commerce.shipment_line_id_seq'::regclass);


--
-- Name: lead id; Type: DEFAULT; Schema: crm; Owner: -
--

ALTER TABLE ONLY crm.lead ALTER COLUMN id SET DEFAULT nextval('crm.lead_id_seq'::regclass);


--
-- Name: partner id; Type: DEFAULT; Schema: crm; Owner: -
--

ALTER TABLE ONLY crm.partner ALTER COLUMN id SET DEFAULT nextval('crm.partner_id_seq'::regclass);


--
-- Name: referral_code id; Type: DEFAULT; Schema: crm; Owner: -
--

ALTER TABLE ONLY crm.referral_code ALTER COLUMN id SET DEFAULT nextval('crm.referral_code_id_seq'::regclass);


--
-- Name: address id; Type: DEFAULT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.address ALTER COLUMN id SET DEFAULT nextval('identity.address_id_seq'::regclass);


--
-- Name: buyer_profile id; Type: DEFAULT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.buyer_profile ALTER COLUMN id SET DEFAULT nextval('identity.buyer_profile_id_seq'::regclass);


--
-- Name: credit_account id; Type: DEFAULT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.credit_account ALTER COLUMN id SET DEFAULT nextval('identity.credit_account_id_seq'::regclass);


--
-- Name: otp_code id; Type: DEFAULT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.otp_code ALTER COLUMN id SET DEFAULT nextval('identity.otp_code_id_seq'::regclass);


--
-- Name: password_reset_token id; Type: DEFAULT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.password_reset_token ALTER COLUMN id SET DEFAULT nextval('identity.password_reset_token_id_seq'::regclass);


--
-- Name: permission id; Type: DEFAULT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.permission ALTER COLUMN id SET DEFAULT nextval('identity.permission_id_seq'::regclass);


--
-- Name: price_list_assignment id; Type: DEFAULT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.price_list_assignment ALTER COLUMN id SET DEFAULT nextval('identity.price_list_assignment_id_seq'::regclass);


--
-- Name: purchase_scope id; Type: DEFAULT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.purchase_scope ALTER COLUMN id SET DEFAULT nextval('identity.purchase_scope_id_seq'::regclass);


--
-- Name: refresh_token id; Type: DEFAULT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.refresh_token ALTER COLUMN id SET DEFAULT nextval('identity.refresh_token_id_seq'::regclass);


--
-- Name: role id; Type: DEFAULT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.role ALTER COLUMN id SET DEFAULT nextval('identity.role_id_seq'::regclass);


--
-- Name: session id; Type: DEFAULT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.session ALTER COLUMN id SET DEFAULT nextval('identity.session_id_seq'::regclass);


--
-- Name: supplier_profile id; Type: DEFAULT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.supplier_profile ALTER COLUMN id SET DEFAULT nextval('identity.supplier_profile_id_seq'::regclass);


--
-- Name: user id; Type: DEFAULT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity."user" ALTER COLUMN id SET DEFAULT nextval('identity.user_id_seq'::regclass);


--
-- Name: verification_application id; Type: DEFAULT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.verification_application ALTER COLUMN id SET DEFAULT nextval('identity.verification_application_id_seq'::regclass);


--
-- Name: verification_document id; Type: DEFAULT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.verification_document ALTER COLUMN id SET DEFAULT nextval('identity.verification_document_id_seq'::regclass);


--
-- Name: lot id; Type: DEFAULT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.lot ALTER COLUMN id SET DEFAULT nextval('inventory.lot_id_seq'::regclass);


--
-- Name: quarantine_record id; Type: DEFAULT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.quarantine_record ALTER COLUMN id SET DEFAULT nextval('inventory.quarantine_record_id_seq'::regclass);


--
-- Name: reservation id; Type: DEFAULT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.reservation ALTER COLUMN id SET DEFAULT nextval('inventory.reservation_id_seq'::regclass);


--
-- Name: stock_adjustment id; Type: DEFAULT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.stock_adjustment ALTER COLUMN id SET DEFAULT nextval('inventory.stock_adjustment_id_seq'::regclass);


--
-- Name: stock_level id; Type: DEFAULT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.stock_level ALTER COLUMN id SET DEFAULT nextval('inventory.stock_level_id_seq'::regclass);


--
-- Name: payment_attempt id; Type: DEFAULT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.payment_attempt ALTER COLUMN id SET DEFAULT nextval('payments.payment_attempt_id_seq'::regclass);


--
-- Name: payment_intent id; Type: DEFAULT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.payment_intent ALTER COLUMN id SET DEFAULT nextval('payments.payment_intent_id_seq'::regclass);


--
-- Name: payment_method id; Type: DEFAULT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.payment_method ALTER COLUMN id SET DEFAULT nextval('payments.payment_method_id_seq'::regclass);


--
-- Name: payment_refund id; Type: DEFAULT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.payment_refund ALTER COLUMN id SET DEFAULT nextval('payments.payment_refund_id_seq'::regclass);


--
-- Name: webhook_event id; Type: DEFAULT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.webhook_event ALTER COLUMN id SET DEFAULT nextval('payments.webhook_event_id_seq'::regclass);


--
-- Name: audit_log id; Type: DEFAULT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.audit_log ALTER COLUMN id SET DEFAULT nextval('platform.audit_log_id_seq'::regclass);


--
-- Name: feature_flag id; Type: DEFAULT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.feature_flag ALTER COLUMN id SET DEFAULT nextval('platform.feature_flag_id_seq'::regclass);


--
-- Name: health_check id; Type: DEFAULT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.health_check ALTER COLUMN id SET DEFAULT nextval('platform.health_check_id_seq'::regclass);


--
-- Name: integration_traffic id; Type: DEFAULT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.integration_traffic ALTER COLUMN id SET DEFAULT nextval('platform.integration_traffic_id_seq'::regclass);


--
-- Name: media_upload id; Type: DEFAULT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.media_upload ALTER COLUMN id SET DEFAULT nextval('platform.media_upload_id_seq'::regclass);


--
-- Name: notification id; Type: DEFAULT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.notification ALTER COLUMN id SET DEFAULT nextval('platform.notification_id_seq'::regclass);


--
-- Name: notification_event id; Type: DEFAULT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.notification_event ALTER COLUMN id SET DEFAULT nextval('platform.notification_event_id_seq'::regclass);


--
-- Name: notification_template id; Type: DEFAULT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.notification_template ALTER COLUMN id SET DEFAULT nextval('platform.notification_template_id_seq'::regclass);


--
-- Name: reference_region id; Type: DEFAULT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.reference_region ALTER COLUMN id SET DEFAULT nextval('platform.reference_region_id_seq'::regclass);


--
-- Name: price_list id; Type: DEFAULT; Schema: pricing; Owner: -
--

ALTER TABLE ONLY pricing.price_list ALTER COLUMN id SET DEFAULT nextval('pricing.price_list_id_seq'::regclass);


--
-- Name: price_list_assignment id; Type: DEFAULT; Schema: pricing; Owner: -
--

ALTER TABLE ONLY pricing.price_list_assignment ALTER COLUMN id SET DEFAULT nextval('pricing.price_list_assignment_id_seq'::regclass);


--
-- Name: price_list_item id; Type: DEFAULT; Schema: pricing; Owner: -
--

ALTER TABLE ONLY pricing.price_list_item ALTER COLUMN id SET DEFAULT nextval('pricing.price_list_item_id_seq'::regclass);


--
-- Name: quantity_tier id; Type: DEFAULT; Schema: pricing; Owner: -
--

ALTER TABLE ONLY pricing.quantity_tier ALTER COLUMN id SET DEFAULT nextval('pricing.quantity_tier_id_seq'::regclass);


--
-- Name: promotion id; Type: DEFAULT; Schema: promotions; Owner: -
--

ALTER TABLE ONLY promotions.promotion ALTER COLUMN id SET DEFAULT nextval('promotions.promotion_id_seq'::regclass);


--
-- Name: voucher_redemption id; Type: DEFAULT; Schema: promotions; Owner: -
--

ALTER TABLE ONLY promotions.voucher_redemption ALTER COLUMN id SET DEFAULT nextval('promotions.voucher_redemption_id_seq'::regclass);


--
-- Name: supplier_contract id; Type: DEFAULT; Schema: suppliers; Owner: -
--

ALTER TABLE ONLY suppliers.supplier_contract ALTER COLUMN id SET DEFAULT nextval('suppliers.supplier_contract_id_seq'::regclass);


--
-- Name: supplier_profile id; Type: DEFAULT; Schema: suppliers; Owner: -
--

ALTER TABLE ONLY suppliers.supplier_profile ALTER COLUMN id SET DEFAULT nextval('suppliers.supplier_profile_id_seq'::regclass);


--
-- Name: budget budget_pkey; Type: CONSTRAINT; Schema: ai; Owner: -
--

ALTER TABLE ONLY ai.budget
    ADD CONSTRAINT budget_pkey PRIMARY KEY (id);


--
-- Name: budget budget_scope_key_key; Type: CONSTRAINT; Schema: ai; Owner: -
--

ALTER TABLE ONLY ai.budget
    ADD CONSTRAINT budget_scope_key_key UNIQUE (scope_key);


--
-- Name: model_routing model_routing_feature_key_key; Type: CONSTRAINT; Schema: ai; Owner: -
--

ALTER TABLE ONLY ai.model_routing
    ADD CONSTRAINT model_routing_feature_key_key UNIQUE (feature_key);


--
-- Name: model_routing model_routing_pkey; Type: CONSTRAINT; Schema: ai; Owner: -
--

ALTER TABLE ONLY ai.model_routing
    ADD CONSTRAINT model_routing_pkey PRIMARY KEY (id);


--
-- Name: prompt prompt_key_key; Type: CONSTRAINT; Schema: ai; Owner: -
--

ALTER TABLE ONLY ai.prompt
    ADD CONSTRAINT prompt_key_key UNIQUE (key);


--
-- Name: prompt prompt_pkey; Type: CONSTRAINT; Schema: ai; Owner: -
--

ALTER TABLE ONLY ai.prompt
    ADD CONSTRAINT prompt_pkey PRIMARY KEY (id);


--
-- Name: prompt_version prompt_version_pkey; Type: CONSTRAINT; Schema: ai; Owner: -
--

ALTER TABLE ONLY ai.prompt_version
    ADD CONSTRAINT prompt_version_pkey PRIMARY KEY (id);


--
-- Name: prompt_version prompt_version_prompt_id_version_key; Type: CONSTRAINT; Schema: ai; Owner: -
--

ALTER TABLE ONLY ai.prompt_version
    ADD CONSTRAINT prompt_version_prompt_id_version_key UNIQUE (prompt_id, version);


--
-- Name: proposal proposal_code_key; Type: CONSTRAINT; Schema: ai; Owner: -
--

ALTER TABLE ONLY ai.proposal
    ADD CONSTRAINT proposal_code_key UNIQUE (code);


--
-- Name: proposal proposal_pkey; Type: CONSTRAINT; Schema: ai; Owner: -
--

ALTER TABLE ONLY ai.proposal
    ADD CONSTRAINT proposal_pkey PRIMARY KEY (id);


--
-- Name: usage_record usage_record_pkey; Type: CONSTRAINT; Schema: ai; Owner: -
--

ALTER TABLE ONLY ai.usage_record
    ADD CONSTRAINT usage_record_pkey PRIMARY KEY (id);


--
-- Name: daily_erp_sync_rollup daily_erp_sync_rollup_pkey; Type: CONSTRAINT; Schema: analytics; Owner: -
--

ALTER TABLE ONLY analytics.daily_erp_sync_rollup
    ADD CONSTRAINT daily_erp_sync_rollup_pkey PRIMARY KEY (rollup_date, sync_type, job_status);


--
-- Name: daily_financial_rollup daily_financial_rollup_pkey; Type: CONSTRAINT; Schema: analytics; Owner: -
--

ALTER TABLE ONLY analytics.daily_financial_rollup
    ADD CONSTRAINT daily_financial_rollup_pkey PRIMARY KEY (rollup_date, buyer_code, currency);


--
-- Name: daily_inventory_rollup daily_inventory_rollup_pkey; Type: CONSTRAINT; Schema: analytics; Owner: -
--

ALTER TABLE ONLY analytics.daily_inventory_rollup
    ADD CONSTRAINT daily_inventory_rollup_pkey PRIMARY KEY (rollup_date, product_code, supplier_code, location_code);


--
-- Name: daily_order_rollup daily_order_rollup_pkey; Type: CONSTRAINT; Schema: analytics; Owner: -
--

ALTER TABLE ONLY analytics.daily_order_rollup
    ADD CONSTRAINT daily_order_rollup_pkey PRIMARY KEY (rollup_date, buyer_code, supplier_code, currency, status);


--
-- Name: daily_payment_rollup daily_payment_rollup_pkey; Type: CONSTRAINT; Schema: analytics; Owner: -
--

ALTER TABLE ONLY analytics.daily_payment_rollup
    ADD CONSTRAINT daily_payment_rollup_pkey PRIMARY KEY (rollup_date, buyer_code, payment_method, payment_status, currency);


--
-- Name: daily_promotion_rollup daily_promotion_rollup_pkey; Type: CONSTRAINT; Schema: analytics; Owner: -
--

ALTER TABLE ONLY analytics.daily_promotion_rollup
    ADD CONSTRAINT daily_promotion_rollup_pkey PRIMARY KEY (rollup_date, promotion_code, voucher_code, currency);


--
-- Name: attribute attribute_code_key; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.attribute
    ADD CONSTRAINT attribute_code_key UNIQUE (code);


--
-- Name: attribute attribute_pkey; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.attribute
    ADD CONSTRAINT attribute_pkey PRIMARY KEY (id);


--
-- Name: attribute_value attribute_value_pkey; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.attribute_value
    ADD CONSTRAINT attribute_value_pkey PRIMARY KEY (id);


--
-- Name: brand brand_code_key; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.brand
    ADD CONSTRAINT brand_code_key UNIQUE (code);


--
-- Name: brand brand_pkey; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.brand
    ADD CONSTRAINT brand_pkey PRIMARY KEY (id);


--
-- Name: brand brand_slug_key; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.brand
    ADD CONSTRAINT brand_slug_key UNIQUE (slug);


--
-- Name: category category_code_key; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.category
    ADD CONSTRAINT category_code_key UNIQUE (code);


--
-- Name: category category_pkey; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.category
    ADD CONSTRAINT category_pkey PRIMARY KEY (id);


--
-- Name: category category_slug_key; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.category
    ADD CONSTRAINT category_slug_key UNIQUE (slug);


--
-- Name: handling_class handling_class_code_key; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.handling_class
    ADD CONSTRAINT handling_class_code_key UNIQUE (code);


--
-- Name: handling_class handling_class_pkey; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.handling_class
    ADD CONSTRAINT handling_class_pkey PRIMARY KEY (id);


--
-- Name: product_attribute product_attribute_pkey; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product_attribute
    ADD CONSTRAINT product_attribute_pkey PRIMARY KEY (id);


--
-- Name: product_attribute product_attribute_product_id_attribute_id_attribute_value_i_key; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product_attribute
    ADD CONSTRAINT product_attribute_product_id_attribute_id_attribute_value_i_key UNIQUE (product_id, attribute_id, attribute_value_id);


--
-- Name: product product_code_key; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product
    ADD CONSTRAINT product_code_key UNIQUE (code);


--
-- Name: product_media product_media_code_key; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product_media
    ADD CONSTRAINT product_media_code_key UNIQUE (code);


--
-- Name: product_media product_media_pkey; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product_media
    ADD CONSTRAINT product_media_pkey PRIMARY KEY (id);


--
-- Name: product product_pkey; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product
    ADD CONSTRAINT product_pkey PRIMARY KEY (id);


--
-- Name: product product_slug_key; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product
    ADD CONSTRAINT product_slug_key UNIQUE (slug);


--
-- Name: product_unit product_unit_pkey; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product_unit
    ADD CONSTRAINT product_unit_pkey PRIMARY KEY (id);


--
-- Name: product_unit product_unit_product_id_unit_id_key; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product_unit
    ADD CONSTRAINT product_unit_product_id_unit_id_key UNIQUE (product_id, unit_id);


--
-- Name: supplier supplier_code_key; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.supplier
    ADD CONSTRAINT supplier_code_key UNIQUE (code);


--
-- Name: supplier supplier_pkey; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.supplier
    ADD CONSTRAINT supplier_pkey PRIMARY KEY (id);


--
-- Name: supplier supplier_slug_key; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.supplier
    ADD CONSTRAINT supplier_slug_key UNIQUE (slug);


--
-- Name: unit unit_code_key; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.unit
    ADD CONSTRAINT unit_code_key UNIQUE (code);


--
-- Name: unit unit_pkey; Type: CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.unit
    ADD CONSTRAINT unit_pkey PRIMARY KEY (id);


--
-- Name: article article_code_key; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.article
    ADD CONSTRAINT article_code_key UNIQUE (code);


--
-- Name: article article_pkey; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.article
    ADD CONSTRAINT article_pkey PRIMARY KEY (id);


--
-- Name: article article_slug_key; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.article
    ADD CONSTRAINT article_slug_key UNIQUE (slug);


--
-- Name: banner banner_code_key; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.banner
    ADD CONSTRAINT banner_code_key UNIQUE (code);


--
-- Name: banner banner_pkey; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.banner
    ADD CONSTRAINT banner_pkey PRIMARY KEY (id);


--
-- Name: contact_enquiry contact_enquiry_code_key; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.contact_enquiry
    ADD CONSTRAINT contact_enquiry_code_key UNIQUE (code);


--
-- Name: contact_enquiry contact_enquiry_pkey; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.contact_enquiry
    ADD CONSTRAINT contact_enquiry_pkey PRIMARY KEY (id);


--
-- Name: faq faq_code_key; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.faq
    ADD CONSTRAINT faq_code_key UNIQUE (code);


--
-- Name: faq faq_pkey; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.faq
    ADD CONSTRAINT faq_pkey PRIMARY KEY (id);


--
-- Name: homepage_layout homepage_layout_pkey; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.homepage_layout
    ADD CONSTRAINT homepage_layout_pkey PRIMARY KEY (id);


--
-- Name: legal_document legal_document_pkey; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.legal_document
    ADD CONSTRAINT legal_document_pkey PRIMARY KEY (doc_type);


--
-- Name: legal_document_version legal_document_version_doc_type_version_key; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.legal_document_version
    ADD CONSTRAINT legal_document_version_doc_type_version_key UNIQUE (doc_type, version);


--
-- Name: legal_document_version legal_document_version_pkey; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.legal_document_version
    ADD CONSTRAINT legal_document_version_pkey PRIMARY KEY (id);


--
-- Name: menu_item menu_item_code_key; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.menu_item
    ADD CONSTRAINT menu_item_code_key UNIQUE (code);


--
-- Name: menu_item menu_item_pkey; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.menu_item
    ADD CONSTRAINT menu_item_pkey PRIMARY KEY (id);


--
-- Name: newsletter_subscriber newsletter_subscriber_code_key; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.newsletter_subscriber
    ADD CONSTRAINT newsletter_subscriber_code_key UNIQUE (code);


--
-- Name: newsletter_subscriber newsletter_subscriber_email_key; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.newsletter_subscriber
    ADD CONSTRAINT newsletter_subscriber_email_key UNIQUE (email);


--
-- Name: newsletter_subscriber newsletter_subscriber_pkey; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.newsletter_subscriber
    ADD CONSTRAINT newsletter_subscriber_pkey PRIMARY KEY (id);


--
-- Name: page page_code_key; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.page
    ADD CONSTRAINT page_code_key UNIQUE (code);


--
-- Name: page page_pkey; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.page
    ADD CONSTRAINT page_pkey PRIMARY KEY (id);


--
-- Name: page page_slug_key; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.page
    ADD CONSTRAINT page_slug_key UNIQUE (slug);


--
-- Name: seo_settings seo_settings_key_key; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.seo_settings
    ADD CONSTRAINT seo_settings_key_key UNIQUE (key);


--
-- Name: seo_settings seo_settings_pkey; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.seo_settings
    ADD CONSTRAINT seo_settings_pkey PRIMARY KEY (id);


--
-- Name: seo_template seo_template_key_key; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.seo_template
    ADD CONSTRAINT seo_template_key_key UNIQUE (key);


--
-- Name: seo_template seo_template_pkey; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.seo_template
    ADD CONSTRAINT seo_template_pkey PRIMARY KEY (id);


--
-- Name: setting setting_key_key; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.setting
    ADD CONSTRAINT setting_key_key UNIQUE (key);


--
-- Name: setting setting_pkey; Type: CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.setting
    ADD CONSTRAINT setting_pkey PRIMARY KEY (id);


--
-- Name: cart cart_buyer_id_key; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.cart
    ADD CONSTRAINT cart_buyer_id_key UNIQUE (buyer_id);


--
-- Name: cart cart_code_key; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.cart
    ADD CONSTRAINT cart_code_key UNIQUE (code);


--
-- Name: cart_line cart_line_code_key; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.cart_line
    ADD CONSTRAINT cart_line_code_key UNIQUE (code);


--
-- Name: cart_line cart_line_pkey; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.cart_line
    ADD CONSTRAINT cart_line_pkey PRIMARY KEY (id);


--
-- Name: cart cart_pkey; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.cart
    ADD CONSTRAINT cart_pkey PRIMARY KEY (id);


--
-- Name: checkout_idempotency checkout_idempotency_buyer_id_idem_key_key; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.checkout_idempotency
    ADD CONSTRAINT checkout_idempotency_buyer_id_idem_key_key UNIQUE (buyer_id, idem_key);


--
-- Name: checkout_idempotency checkout_idempotency_pkey; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.checkout_idempotency
    ADD CONSTRAINT checkout_idempotency_pkey PRIMARY KEY (id);


--
-- Name: credit_note credit_note_code_key; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.credit_note
    ADD CONSTRAINT credit_note_code_key UNIQUE (code);


--
-- Name: credit_note credit_note_pkey; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.credit_note
    ADD CONSTRAINT credit_note_pkey PRIMARY KEY (id);


--
-- Name: invoice invoice_code_key; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.invoice
    ADD CONSTRAINT invoice_code_key UNIQUE (code);


--
-- Name: invoice invoice_pkey; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.invoice
    ADD CONSTRAINT invoice_pkey PRIMARY KEY (id);


--
-- Name: order order_code_key; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce."order"
    ADD CONSTRAINT order_code_key UNIQUE (code);


--
-- Name: order_history order_history_pkey; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.order_history
    ADD CONSTRAINT order_history_pkey PRIMARY KEY (id);


--
-- Name: order_line order_line_code_key; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.order_line
    ADD CONSTRAINT order_line_code_key UNIQUE (code);


--
-- Name: order_line order_line_order_id_product_id_key; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.order_line
    ADD CONSTRAINT order_line_order_id_product_id_key UNIQUE (order_id, product_id);


--
-- Name: order_line order_line_pkey; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.order_line
    ADD CONSTRAINT order_line_pkey PRIMARY KEY (id);


--
-- Name: order order_pkey; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce."order"
    ADD CONSTRAINT order_pkey PRIMARY KEY (id);


--
-- Name: return_line return_line_pkey; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.return_line
    ADD CONSTRAINT return_line_pkey PRIMARY KEY (id);


--
-- Name: return_line return_line_return_id_order_line_id_key; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.return_line
    ADD CONSTRAINT return_line_return_id_order_line_id_key UNIQUE (return_id, order_line_id);


--
-- Name: return_request return_request_code_key; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.return_request
    ADD CONSTRAINT return_request_code_key UNIQUE (code);


--
-- Name: return_request return_request_pkey; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.return_request
    ADD CONSTRAINT return_request_pkey PRIMARY KEY (id);


--
-- Name: shipment shipment_code_key; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.shipment
    ADD CONSTRAINT shipment_code_key UNIQUE (code);


--
-- Name: shipment_line shipment_line_pkey; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.shipment_line
    ADD CONSTRAINT shipment_line_pkey PRIMARY KEY (id);


--
-- Name: shipment_line shipment_line_shipment_id_order_line_id_key; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.shipment_line
    ADD CONSTRAINT shipment_line_shipment_id_order_line_id_key UNIQUE (shipment_id, order_line_id);


--
-- Name: shipment shipment_pkey; Type: CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.shipment
    ADD CONSTRAINT shipment_pkey PRIMARY KEY (id);


--
-- Name: lead lead_code_key; Type: CONSTRAINT; Schema: crm; Owner: -
--

ALTER TABLE ONLY crm.lead
    ADD CONSTRAINT lead_code_key UNIQUE (code);


--
-- Name: lead lead_pkey; Type: CONSTRAINT; Schema: crm; Owner: -
--

ALTER TABLE ONLY crm.lead
    ADD CONSTRAINT lead_pkey PRIMARY KEY (id);


--
-- Name: partner partner_code_key; Type: CONSTRAINT; Schema: crm; Owner: -
--

ALTER TABLE ONLY crm.partner
    ADD CONSTRAINT partner_code_key UNIQUE (code);


--
-- Name: partner partner_pkey; Type: CONSTRAINT; Schema: crm; Owner: -
--

ALTER TABLE ONLY crm.partner
    ADD CONSTRAINT partner_pkey PRIMARY KEY (id);


--
-- Name: partner partner_slug_key; Type: CONSTRAINT; Schema: crm; Owner: -
--

ALTER TABLE ONLY crm.partner
    ADD CONSTRAINT partner_slug_key UNIQUE (slug);


--
-- Name: referral_code referral_code_code_key; Type: CONSTRAINT; Schema: crm; Owner: -
--

ALTER TABLE ONLY crm.referral_code
    ADD CONSTRAINT referral_code_code_key UNIQUE (code);


--
-- Name: referral_code referral_code_pkey; Type: CONSTRAINT; Schema: crm; Owner: -
--

ALTER TABLE ONLY crm.referral_code
    ADD CONSTRAINT referral_code_pkey PRIMARY KEY (id);


--
-- Name: address address_code_key; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.address
    ADD CONSTRAINT address_code_key UNIQUE (code);


--
-- Name: address address_pkey; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.address
    ADD CONSTRAINT address_pkey PRIMARY KEY (id);


--
-- Name: buyer_profile buyer_profile_code_key; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.buyer_profile
    ADD CONSTRAINT buyer_profile_code_key UNIQUE (code);


--
-- Name: buyer_profile buyer_profile_pkey; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.buyer_profile
    ADD CONSTRAINT buyer_profile_pkey PRIMARY KEY (id);


--
-- Name: credit_account credit_account_code_key; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.credit_account
    ADD CONSTRAINT credit_account_code_key UNIQUE (code);


--
-- Name: credit_account credit_account_pkey; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.credit_account
    ADD CONSTRAINT credit_account_pkey PRIMARY KEY (id);


--
-- Name: otp_code otp_code_code_key; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.otp_code
    ADD CONSTRAINT otp_code_code_key UNIQUE (code);


--
-- Name: otp_code otp_code_pkey; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.otp_code
    ADD CONSTRAINT otp_code_pkey PRIMARY KEY (id);


--
-- Name: password_reset_token password_reset_token_code_key; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.password_reset_token
    ADD CONSTRAINT password_reset_token_code_key UNIQUE (code);


--
-- Name: password_reset_token password_reset_token_pkey; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.password_reset_token
    ADD CONSTRAINT password_reset_token_pkey PRIMARY KEY (id);


--
-- Name: permission permission_code_key; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.permission
    ADD CONSTRAINT permission_code_key UNIQUE (code);


--
-- Name: permission permission_pkey; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.permission
    ADD CONSTRAINT permission_pkey PRIMARY KEY (id);


--
-- Name: price_list_assignment price_list_assignment_code_key; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.price_list_assignment
    ADD CONSTRAINT price_list_assignment_code_key UNIQUE (code);


--
-- Name: price_list_assignment price_list_assignment_pkey; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.price_list_assignment
    ADD CONSTRAINT price_list_assignment_pkey PRIMARY KEY (id);


--
-- Name: purchase_scope purchase_scope_code_key; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.purchase_scope
    ADD CONSTRAINT purchase_scope_code_key UNIQUE (code);


--
-- Name: purchase_scope purchase_scope_pkey; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.purchase_scope
    ADD CONSTRAINT purchase_scope_pkey PRIMARY KEY (id);


--
-- Name: refresh_token refresh_token_code_key; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.refresh_token
    ADD CONSTRAINT refresh_token_code_key UNIQUE (code);


--
-- Name: refresh_token refresh_token_pkey; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.refresh_token
    ADD CONSTRAINT refresh_token_pkey PRIMARY KEY (id);


--
-- Name: role role_code_key; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.role
    ADD CONSTRAINT role_code_key UNIQUE (code);


--
-- Name: role_permission role_permission_pkey; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.role_permission
    ADD CONSTRAINT role_permission_pkey PRIMARY KEY (role_id, permission_id);


--
-- Name: role role_pkey; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.role
    ADD CONSTRAINT role_pkey PRIMARY KEY (id);


--
-- Name: session session_code_key; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.session
    ADD CONSTRAINT session_code_key UNIQUE (code);


--
-- Name: session session_pkey; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.session
    ADD CONSTRAINT session_pkey PRIMARY KEY (id);


--
-- Name: supplier_profile supplier_profile_code_key; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.supplier_profile
    ADD CONSTRAINT supplier_profile_code_key UNIQUE (code);


--
-- Name: supplier_profile supplier_profile_pkey; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.supplier_profile
    ADD CONSTRAINT supplier_profile_pkey PRIMARY KEY (id);


--
-- Name: user user_code_key; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity."user"
    ADD CONSTRAINT user_code_key UNIQUE (code);


--
-- Name: user user_email_key; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity."user"
    ADD CONSTRAINT user_email_key UNIQUE (email);


--
-- Name: user user_pkey; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity."user"
    ADD CONSTRAINT user_pkey PRIMARY KEY (id);


--
-- Name: user_role user_role_pkey; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.user_role
    ADD CONSTRAINT user_role_pkey PRIMARY KEY (user_id, role_id);


--
-- Name: verification_application verification_application_code_key; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.verification_application
    ADD CONSTRAINT verification_application_code_key UNIQUE (code);


--
-- Name: verification_application verification_application_pkey; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.verification_application
    ADD CONSTRAINT verification_application_pkey PRIMARY KEY (id);


--
-- Name: verification_document verification_document_code_key; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.verification_document
    ADD CONSTRAINT verification_document_code_key UNIQUE (code);


--
-- Name: verification_document verification_document_pkey; Type: CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.verification_document
    ADD CONSTRAINT verification_document_pkey PRIMARY KEY (id);


--
-- Name: lot lot_code_key; Type: CONSTRAINT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.lot
    ADD CONSTRAINT lot_code_key UNIQUE (code);


--
-- Name: lot lot_pkey; Type: CONSTRAINT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.lot
    ADD CONSTRAINT lot_pkey PRIMARY KEY (id);


--
-- Name: quarantine_record quarantine_record_code_key; Type: CONSTRAINT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.quarantine_record
    ADD CONSTRAINT quarantine_record_code_key UNIQUE (code);


--
-- Name: quarantine_record quarantine_record_pkey; Type: CONSTRAINT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.quarantine_record
    ADD CONSTRAINT quarantine_record_pkey PRIMARY KEY (id);


--
-- Name: reservation reservation_code_key; Type: CONSTRAINT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.reservation
    ADD CONSTRAINT reservation_code_key UNIQUE (code);


--
-- Name: reservation reservation_pkey; Type: CONSTRAINT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.reservation
    ADD CONSTRAINT reservation_pkey PRIMARY KEY (id);


--
-- Name: stock_adjustment stock_adjustment_code_key; Type: CONSTRAINT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.stock_adjustment
    ADD CONSTRAINT stock_adjustment_code_key UNIQUE (code);


--
-- Name: stock_adjustment stock_adjustment_pkey; Type: CONSTRAINT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.stock_adjustment
    ADD CONSTRAINT stock_adjustment_pkey PRIMARY KEY (id);


--
-- Name: stock_level stock_level_code_key; Type: CONSTRAINT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.stock_level
    ADD CONSTRAINT stock_level_code_key UNIQUE (code);


--
-- Name: stock_level stock_level_pkey; Type: CONSTRAINT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.stock_level
    ADD CONSTRAINT stock_level_pkey PRIMARY KEY (id);


--
-- Name: stock_level stock_level_product_id_supplier_id_key; Type: CONSTRAINT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.stock_level
    ADD CONSTRAINT stock_level_product_id_supplier_id_key UNIQUE (product_id, supplier_id);


--
-- Name: credit_account credit_account_pkey; Type: CONSTRAINT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.credit_account
    ADD CONSTRAINT credit_account_pkey PRIMARY KEY (buyer_id);


--
-- Name: payment_attempt payment_attempt_pkey; Type: CONSTRAINT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.payment_attempt
    ADD CONSTRAINT payment_attempt_pkey PRIMARY KEY (id);


--
-- Name: payment_intent payment_intent_buyer_id_idem_key_key; Type: CONSTRAINT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.payment_intent
    ADD CONSTRAINT payment_intent_buyer_id_idem_key_key UNIQUE (buyer_id, idem_key);


--
-- Name: payment_intent payment_intent_code_key; Type: CONSTRAINT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.payment_intent
    ADD CONSTRAINT payment_intent_code_key UNIQUE (code);


--
-- Name: payment_intent payment_intent_pkey; Type: CONSTRAINT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.payment_intent
    ADD CONSTRAINT payment_intent_pkey PRIMARY KEY (id);


--
-- Name: payment_method payment_method_code_key; Type: CONSTRAINT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.payment_method
    ADD CONSTRAINT payment_method_code_key UNIQUE (code);


--
-- Name: payment_method payment_method_pkey; Type: CONSTRAINT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.payment_method
    ADD CONSTRAINT payment_method_pkey PRIMARY KEY (id);


--
-- Name: payment_refund payment_refund_code_key; Type: CONSTRAINT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.payment_refund
    ADD CONSTRAINT payment_refund_code_key UNIQUE (code);


--
-- Name: payment_refund payment_refund_pkey; Type: CONSTRAINT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.payment_refund
    ADD CONSTRAINT payment_refund_pkey PRIMARY KEY (id);


--
-- Name: webhook_event webhook_event_pkey; Type: CONSTRAINT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.webhook_event
    ADD CONSTRAINT webhook_event_pkey PRIMARY KEY (id);


--
-- Name: webhook_event webhook_event_provider_provider_event_id_key; Type: CONSTRAINT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.webhook_event
    ADD CONSTRAINT webhook_event_provider_provider_event_id_key UNIQUE (provider, provider_event_id);


--
-- Name: audit_log audit_log_pkey; Type: CONSTRAINT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.audit_log
    ADD CONSTRAINT audit_log_pkey PRIMARY KEY (id);


--
-- Name: feature_flag feature_flag_key_key; Type: CONSTRAINT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.feature_flag
    ADD CONSTRAINT feature_flag_key_key UNIQUE (key);


--
-- Name: feature_flag feature_flag_pkey; Type: CONSTRAINT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.feature_flag
    ADD CONSTRAINT feature_flag_pkey PRIMARY KEY (id);


--
-- Name: health_check health_check_pkey; Type: CONSTRAINT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.health_check
    ADD CONSTRAINT health_check_pkey PRIMARY KEY (id);


--
-- Name: integration_traffic integration_traffic_pkey; Type: CONSTRAINT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.integration_traffic
    ADD CONSTRAINT integration_traffic_pkey PRIMARY KEY (id);


--
-- Name: media_upload media_upload_code_key; Type: CONSTRAINT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.media_upload
    ADD CONSTRAINT media_upload_code_key UNIQUE (code);


--
-- Name: media_upload media_upload_pkey; Type: CONSTRAINT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.media_upload
    ADD CONSTRAINT media_upload_pkey PRIMARY KEY (id);


--
-- Name: notification notification_code_key; Type: CONSTRAINT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.notification
    ADD CONSTRAINT notification_code_key UNIQUE (code);


--
-- Name: notification_event notification_event_pkey; Type: CONSTRAINT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.notification_event
    ADD CONSTRAINT notification_event_pkey PRIMARY KEY (id);


--
-- Name: notification notification_pkey; Type: CONSTRAINT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.notification
    ADD CONSTRAINT notification_pkey PRIMARY KEY (id);


--
-- Name: notification_template notification_template_code_key; Type: CONSTRAINT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.notification_template
    ADD CONSTRAINT notification_template_code_key UNIQUE (code);


--
-- Name: notification_template notification_template_pkey; Type: CONSTRAINT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.notification_template
    ADD CONSTRAINT notification_template_pkey PRIMARY KEY (id);


--
-- Name: reference_region reference_region_code_key; Type: CONSTRAINT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.reference_region
    ADD CONSTRAINT reference_region_code_key UNIQUE (code);


--
-- Name: reference_region reference_region_pkey; Type: CONSTRAINT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.reference_region
    ADD CONSTRAINT reference_region_pkey PRIMARY KEY (id);


--
-- Name: price_list_assignment price_list_assignment_code_key; Type: CONSTRAINT; Schema: pricing; Owner: -
--

ALTER TABLE ONLY pricing.price_list_assignment
    ADD CONSTRAINT price_list_assignment_code_key UNIQUE (code);


--
-- Name: price_list_assignment price_list_assignment_pkey; Type: CONSTRAINT; Schema: pricing; Owner: -
--

ALTER TABLE ONLY pricing.price_list_assignment
    ADD CONSTRAINT price_list_assignment_pkey PRIMARY KEY (id);


--
-- Name: price_list_assignment price_list_assignment_price_list_id_buyer_profile_id_effect_key; Type: CONSTRAINT; Schema: pricing; Owner: -
--

ALTER TABLE ONLY pricing.price_list_assignment
    ADD CONSTRAINT price_list_assignment_price_list_id_buyer_profile_id_effect_key UNIQUE (price_list_id, buyer_profile_id, effective_from);


--
-- Name: price_list price_list_code_key; Type: CONSTRAINT; Schema: pricing; Owner: -
--

ALTER TABLE ONLY pricing.price_list
    ADD CONSTRAINT price_list_code_key UNIQUE (code);


--
-- Name: price_list_item price_list_item_pkey; Type: CONSTRAINT; Schema: pricing; Owner: -
--

ALTER TABLE ONLY pricing.price_list_item
    ADD CONSTRAINT price_list_item_pkey PRIMARY KEY (id);


--
-- Name: price_list_item price_list_item_price_list_id_product_id_unit_id_min_quanti_key; Type: CONSTRAINT; Schema: pricing; Owner: -
--

ALTER TABLE ONLY pricing.price_list_item
    ADD CONSTRAINT price_list_item_price_list_id_product_id_unit_id_min_quanti_key UNIQUE (price_list_id, product_id, unit_id, min_quantity, effective_from);


--
-- Name: price_list price_list_pkey; Type: CONSTRAINT; Schema: pricing; Owner: -
--

ALTER TABLE ONLY pricing.price_list
    ADD CONSTRAINT price_list_pkey PRIMARY KEY (id);


--
-- Name: quantity_tier quantity_tier_pkey; Type: CONSTRAINT; Schema: pricing; Owner: -
--

ALTER TABLE ONLY pricing.quantity_tier
    ADD CONSTRAINT quantity_tier_pkey PRIMARY KEY (id);


--
-- Name: quantity_tier quantity_tier_price_list_item_id_min_quantity_key; Type: CONSTRAINT; Schema: pricing; Owner: -
--

ALTER TABLE ONLY pricing.quantity_tier
    ADD CONSTRAINT quantity_tier_price_list_item_id_min_quantity_key UNIQUE (price_list_item_id, min_quantity);


--
-- Name: promotion promotion_code_key; Type: CONSTRAINT; Schema: promotions; Owner: -
--

ALTER TABLE ONLY promotions.promotion
    ADD CONSTRAINT promotion_code_key UNIQUE (code);


--
-- Name: promotion promotion_pkey; Type: CONSTRAINT; Schema: promotions; Owner: -
--

ALTER TABLE ONLY promotions.promotion
    ADD CONSTRAINT promotion_pkey PRIMARY KEY (id);


--
-- Name: voucher_redemption voucher_redemption_buyer_id_promotion_id_order_id_key; Type: CONSTRAINT; Schema: promotions; Owner: -
--

ALTER TABLE ONLY promotions.voucher_redemption
    ADD CONSTRAINT voucher_redemption_buyer_id_promotion_id_order_id_key UNIQUE (buyer_id, promotion_id, order_id);


--
-- Name: voucher_redemption voucher_redemption_pkey; Type: CONSTRAINT; Schema: promotions; Owner: -
--

ALTER TABLE ONLY promotions.voucher_redemption
    ADD CONSTRAINT voucher_redemption_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: supplier_contract supplier_contract_pkey; Type: CONSTRAINT; Schema: suppliers; Owner: -
--

ALTER TABLE ONLY suppliers.supplier_contract
    ADD CONSTRAINT supplier_contract_pkey PRIMARY KEY (id);


--
-- Name: supplier_contract supplier_contract_supplier_id_version_key; Type: CONSTRAINT; Schema: suppliers; Owner: -
--

ALTER TABLE ONLY suppliers.supplier_contract
    ADD CONSTRAINT supplier_contract_supplier_id_version_key UNIQUE (supplier_id, version);


--
-- Name: supplier_profile supplier_profile_code_key; Type: CONSTRAINT; Schema: suppliers; Owner: -
--

ALTER TABLE ONLY suppliers.supplier_profile
    ADD CONSTRAINT supplier_profile_code_key UNIQUE (code);


--
-- Name: supplier_profile supplier_profile_pkey; Type: CONSTRAINT; Schema: suppliers; Owner: -
--

ALTER TABLE ONLY suppliers.supplier_profile
    ADD CONSTRAINT supplier_profile_pkey PRIMARY KEY (id);


--
-- Name: idx_budget_period; Type: INDEX; Schema: ai; Owner: -
--

CREATE INDEX idx_budget_period ON ai.budget USING btree (period_start, period_end);


--
-- Name: idx_proposal_expires; Type: INDEX; Schema: ai; Owner: -
--

CREATE INDEX idx_proposal_expires ON ai.proposal USING btree (expires_at);


--
-- Name: idx_proposal_feature; Type: INDEX; Schema: ai; Owner: -
--

CREATE INDEX idx_proposal_feature ON ai.proposal USING btree (feature);


--
-- Name: idx_proposal_status; Type: INDEX; Schema: ai; Owner: -
--

CREATE INDEX idx_proposal_status ON ai.proposal USING btree (status);


--
-- Name: idx_proposal_tenant; Type: INDEX; Schema: ai; Owner: -
--

CREATE INDEX idx_proposal_tenant ON ai.proposal USING btree (tenant_id);


--
-- Name: idx_usage_created; Type: INDEX; Schema: ai; Owner: -
--

CREATE INDEX idx_usage_created ON ai.usage_record USING btree (created_at);


--
-- Name: idx_usage_feature; Type: INDEX; Schema: ai; Owner: -
--

CREATE INDEX idx_usage_feature ON ai.usage_record USING btree (feature);


--
-- Name: idx_usage_tenant; Type: INDEX; Schema: ai; Owner: -
--

CREATE INDEX idx_usage_tenant ON ai.usage_record USING btree (tenant_id);


--
-- Name: idx_daily_erp_sync_rollup_date; Type: INDEX; Schema: analytics; Owner: -
--

CREATE INDEX idx_daily_erp_sync_rollup_date ON analytics.daily_erp_sync_rollup USING btree (rollup_date DESC);


--
-- Name: idx_daily_financial_rollup_buyer; Type: INDEX; Schema: analytics; Owner: -
--

CREATE INDEX idx_daily_financial_rollup_buyer ON analytics.daily_financial_rollup USING btree (buyer_code, rollup_date DESC);


--
-- Name: idx_daily_financial_rollup_date; Type: INDEX; Schema: analytics; Owner: -
--

CREATE INDEX idx_daily_financial_rollup_date ON analytics.daily_financial_rollup USING btree (rollup_date DESC);


--
-- Name: idx_daily_inventory_rollup_date; Type: INDEX; Schema: analytics; Owner: -
--

CREATE INDEX idx_daily_inventory_rollup_date ON analytics.daily_inventory_rollup USING btree (rollup_date DESC);


--
-- Name: idx_daily_inventory_rollup_product; Type: INDEX; Schema: analytics; Owner: -
--

CREATE INDEX idx_daily_inventory_rollup_product ON analytics.daily_inventory_rollup USING btree (product_code, rollup_date DESC);


--
-- Name: idx_daily_inventory_rollup_supplier; Type: INDEX; Schema: analytics; Owner: -
--

CREATE INDEX idx_daily_inventory_rollup_supplier ON analytics.daily_inventory_rollup USING btree (supplier_code, rollup_date DESC);


--
-- Name: idx_daily_order_rollup_buyer; Type: INDEX; Schema: analytics; Owner: -
--

CREATE INDEX idx_daily_order_rollup_buyer ON analytics.daily_order_rollup USING btree (buyer_code, rollup_date DESC);


--
-- Name: idx_daily_order_rollup_date; Type: INDEX; Schema: analytics; Owner: -
--

CREATE INDEX idx_daily_order_rollup_date ON analytics.daily_order_rollup USING btree (rollup_date DESC);


--
-- Name: idx_daily_order_rollup_supplier; Type: INDEX; Schema: analytics; Owner: -
--

CREATE INDEX idx_daily_order_rollup_supplier ON analytics.daily_order_rollup USING btree (supplier_code, rollup_date DESC);


--
-- Name: idx_daily_payment_rollup_buyer; Type: INDEX; Schema: analytics; Owner: -
--

CREATE INDEX idx_daily_payment_rollup_buyer ON analytics.daily_payment_rollup USING btree (buyer_code, rollup_date DESC);


--
-- Name: idx_daily_payment_rollup_date; Type: INDEX; Schema: analytics; Owner: -
--

CREATE INDEX idx_daily_payment_rollup_date ON analytics.daily_payment_rollup USING btree (rollup_date DESC);


--
-- Name: idx_daily_promotion_rollup_date; Type: INDEX; Schema: analytics; Owner: -
--

CREATE INDEX idx_daily_promotion_rollup_date ON analytics.daily_promotion_rollup USING btree (rollup_date DESC);


--
-- Name: idx_daily_promotion_rollup_promo; Type: INDEX; Schema: analytics; Owner: -
--

CREATE INDEX idx_daily_promotion_rollup_promo ON analytics.daily_promotion_rollup USING btree (promotion_code, rollup_date DESC);


--
-- Name: idx_catalog_attribute_filterable; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_attribute_filterable ON catalog.attribute USING btree (is_filterable) WHERE ((deleted_at IS NULL) AND (is_filterable = true));


--
-- Name: idx_catalog_attribute_value_attribute; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_attribute_value_attribute ON catalog.attribute_value USING btree (attribute_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_catalog_brand_active; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_brand_active ON catalog.brand USING btree (is_active) WHERE (deleted_at IS NULL);


--
-- Name: idx_catalog_brand_slug; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_brand_slug ON catalog.brand USING btree (slug) WHERE (deleted_at IS NULL);


--
-- Name: idx_catalog_category_active; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_category_active ON catalog.category USING btree (is_active) WHERE (deleted_at IS NULL);


--
-- Name: idx_catalog_category_parent; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_category_parent ON catalog.category USING btree (parent_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_catalog_category_slug; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_category_slug ON catalog.category USING btree (slug) WHERE (deleted_at IS NULL);


--
-- Name: idx_catalog_handling_class_active; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_handling_class_active ON catalog.handling_class USING btree (is_active) WHERE (deleted_at IS NULL);


--
-- Name: idx_catalog_product_active; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_product_active ON catalog.product USING btree (is_active) WHERE (deleted_at IS NULL);


--
-- Name: idx_catalog_product_attribute_attr; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_product_attribute_attr ON catalog.product_attribute USING btree (attribute_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_catalog_product_attribute_product; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_product_attribute_product ON catalog.product_attribute USING btree (product_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_catalog_product_brand; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_product_brand ON catalog.product USING btree (brand_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_catalog_product_category; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_product_category ON catalog.product USING btree (category_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_catalog_product_featured; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_product_featured ON catalog.product USING btree (is_featured) WHERE ((deleted_at IS NULL) AND (is_featured = true));


--
-- Name: idx_catalog_product_handling; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_product_handling ON catalog.product USING btree (handling_class) WHERE (deleted_at IS NULL);


--
-- Name: idx_catalog_product_media_product; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_product_media_product ON catalog.product_media USING btree (product_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_catalog_product_slug; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_product_slug ON catalog.product USING btree (slug) WHERE (deleted_at IS NULL);


--
-- Name: idx_catalog_product_status; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_product_status ON catalog.product USING btree (status) WHERE (deleted_at IS NULL);


--
-- Name: idx_catalog_product_supplier; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_product_supplier ON catalog.product USING btree (supplier_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_catalog_product_unit_product; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_product_unit_product ON catalog.product_unit USING btree (product_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_catalog_supplier_active; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_supplier_active ON catalog.supplier USING btree (is_active) WHERE (deleted_at IS NULL);


--
-- Name: idx_catalog_supplier_slug; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_supplier_slug ON catalog.supplier USING btree (slug) WHERE (deleted_at IS NULL);


--
-- Name: idx_catalog_unit_active; Type: INDEX; Schema: catalog; Owner: -
--

CREATE INDEX idx_catalog_unit_active ON catalog.unit USING btree (is_active) WHERE (deleted_at IS NULL);


--
-- Name: idx_cms_article_slug; Type: INDEX; Schema: cms; Owner: -
--

CREATE INDEX idx_cms_article_slug ON cms.article USING btree (slug) WHERE (deleted_at IS NULL);


--
-- Name: idx_cms_article_status; Type: INDEX; Schema: cms; Owner: -
--

CREATE INDEX idx_cms_article_status ON cms.article USING btree (status) WHERE (deleted_at IS NULL);


--
-- Name: idx_cms_banner_active; Type: INDEX; Schema: cms; Owner: -
--

CREATE INDEX idx_cms_banner_active ON cms.banner USING btree ("position", is_active, sort_order) WHERE (deleted_at IS NULL);


--
-- Name: idx_cms_faq_active; Type: INDEX; Schema: cms; Owner: -
--

CREATE INDEX idx_cms_faq_active ON cms.faq USING btree (is_active, sort_order) WHERE (deleted_at IS NULL);


--
-- Name: idx_cms_faq_category; Type: INDEX; Schema: cms; Owner: -
--

CREATE INDEX idx_cms_faq_category ON cms.faq USING btree (category) WHERE (deleted_at IS NULL);


--
-- Name: idx_cms_menu_location; Type: INDEX; Schema: cms; Owner: -
--

CREATE INDEX idx_cms_menu_location ON cms.menu_item USING btree (location, is_active, sort_order) WHERE (deleted_at IS NULL);


--
-- Name: idx_cms_page_slug; Type: INDEX; Schema: cms; Owner: -
--

CREATE INDEX idx_cms_page_slug ON cms.page USING btree (slug) WHERE (deleted_at IS NULL);


--
-- Name: idx_cms_setting_group; Type: INDEX; Schema: cms; Owner: -
--

CREATE INDEX idx_cms_setting_group ON cms.setting USING btree (group_name);


--
-- Name: idx_contact_enquiry_created_at; Type: INDEX; Schema: cms; Owner: -
--

CREATE INDEX idx_contact_enquiry_created_at ON cms.contact_enquiry USING btree (created_at DESC);


--
-- Name: idx_contact_enquiry_email; Type: INDEX; Schema: cms; Owner: -
--

CREATE INDEX idx_contact_enquiry_email ON cms.contact_enquiry USING btree (email);


--
-- Name: idx_contact_enquiry_source; Type: INDEX; Schema: cms; Owner: -
--

CREATE INDEX idx_contact_enquiry_source ON cms.contact_enquiry USING btree (source);


--
-- Name: idx_contact_enquiry_status; Type: INDEX; Schema: cms; Owner: -
--

CREATE INDEX idx_contact_enquiry_status ON cms.contact_enquiry USING btree (status);


--
-- Name: idx_legal_document_version_doc_type; Type: INDEX; Schema: cms; Owner: -
--

CREATE INDEX idx_legal_document_version_doc_type ON cms.legal_document_version USING btree (doc_type);


--
-- Name: idx_legal_document_version_status; Type: INDEX; Schema: cms; Owner: -
--

CREATE INDEX idx_legal_document_version_status ON cms.legal_document_version USING btree (status);


--
-- Name: idx_legal_document_version_version; Type: INDEX; Schema: cms; Owner: -
--

CREATE INDEX idx_legal_document_version_version ON cms.legal_document_version USING btree (doc_type, version DESC);


--
-- Name: idx_newsletter_subscriber_created_at; Type: INDEX; Schema: cms; Owner: -
--

CREATE INDEX idx_newsletter_subscriber_created_at ON cms.newsletter_subscriber USING btree (created_at DESC);


--
-- Name: idx_newsletter_subscriber_email; Type: INDEX; Schema: cms; Owner: -
--

CREATE INDEX idx_newsletter_subscriber_email ON cms.newsletter_subscriber USING btree (email);


--
-- Name: idx_newsletter_subscriber_status; Type: INDEX; Schema: cms; Owner: -
--

CREATE INDEX idx_newsletter_subscriber_status ON cms.newsletter_subscriber USING btree (status);


--
-- Name: idx_seo_settings_key; Type: INDEX; Schema: cms; Owner: -
--

CREATE INDEX idx_seo_settings_key ON cms.seo_settings USING btree (key);


--
-- Name: idx_seo_template_key; Type: INDEX; Schema: cms; Owner: -
--

CREATE INDEX idx_seo_template_key ON cms.seo_template USING btree (key);


--
-- Name: idx_commerce_cart_buyer; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_cart_buyer ON commerce.cart USING btree (buyer_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_commerce_cart_code; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_cart_code ON commerce.cart USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_commerce_cart_line_cart; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_cart_line_cart ON commerce.cart_line USING btree (cart_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_commerce_cart_line_code; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_cart_line_code ON commerce.cart_line USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_commerce_cart_line_product; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_cart_line_product ON commerce.cart_line USING btree (product_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_commerce_credit_note_batch; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_credit_note_batch ON commerce.credit_note USING btree (batch_code) WHERE (deleted_at IS NULL);


--
-- Name: idx_commerce_credit_note_code; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_credit_note_code ON commerce.credit_note USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_commerce_credit_note_order; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_credit_note_order ON commerce.credit_note USING btree (order_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_commerce_idem_buyer_key; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_idem_buyer_key ON commerce.checkout_idempotency USING btree (buyer_id, idem_key);


--
-- Name: idx_commerce_invoice_code; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_invoice_code ON commerce.invoice USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_commerce_invoice_order; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_invoice_order ON commerce.invoice USING btree (order_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_commerce_order_buyer; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_order_buyer ON commerce."order" USING btree (buyer_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_commerce_order_code; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_order_code ON commerce."order" USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_commerce_order_history_order; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_order_history_order ON commerce.order_history USING btree (order_id);


--
-- Name: idx_commerce_order_line_code; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_order_line_code ON commerce.order_line USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_commerce_order_line_order; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_order_line_order ON commerce.order_line USING btree (order_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_commerce_order_line_product; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_order_line_product ON commerce.order_line USING btree (product_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_commerce_order_placed; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_order_placed ON commerce."order" USING btree (placed_at DESC) WHERE (deleted_at IS NULL);


--
-- Name: idx_commerce_order_supplier; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_order_supplier ON commerce."order" USING btree (supplier_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_commerce_return_buyer; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_return_buyer ON commerce.return_request USING btree (buyer_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_commerce_return_code; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_return_code ON commerce.return_request USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_commerce_return_line_order_line; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_return_line_order_line ON commerce.return_line USING btree (order_line_id);


--
-- Name: idx_commerce_return_line_return; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_return_line_return ON commerce.return_line USING btree (return_id);


--
-- Name: idx_commerce_return_order; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_return_order ON commerce.return_request USING btree (order_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_commerce_shipment_code; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_shipment_code ON commerce.shipment USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_commerce_shipment_line_order_line; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_shipment_line_order_line ON commerce.shipment_line USING btree (order_line_id);


--
-- Name: idx_commerce_shipment_line_shipment; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_shipment_line_shipment ON commerce.shipment_line USING btree (shipment_id);


--
-- Name: idx_commerce_shipment_order; Type: INDEX; Schema: commerce; Owner: -
--

CREATE INDEX idx_commerce_shipment_order ON commerce.shipment USING btree (order_id) WHERE (deleted_at IS NULL);


--
-- Name: uq_commerce_cart_line_cart_product_active; Type: INDEX; Schema: commerce; Owner: -
--

CREATE UNIQUE INDEX uq_commerce_cart_line_cart_product_active ON commerce.cart_line USING btree (cart_id, product_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_crm_lead_code; Type: INDEX; Schema: crm; Owner: -
--

CREATE INDEX idx_crm_lead_code ON crm.lead USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_crm_lead_partner; Type: INDEX; Schema: crm; Owner: -
--

CREATE INDEX idx_crm_lead_partner ON crm.lead USING btree (partner_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_crm_lead_referral; Type: INDEX; Schema: crm; Owner: -
--

CREATE INDEX idx_crm_lead_referral ON crm.lead USING btree (referral_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_crm_lead_status; Type: INDEX; Schema: crm; Owner: -
--

CREATE INDEX idx_crm_lead_status ON crm.lead USING btree (status) WHERE (deleted_at IS NULL);


--
-- Name: idx_crm_partner_code; Type: INDEX; Schema: crm; Owner: -
--

CREATE INDEX idx_crm_partner_code ON crm.partner USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_crm_partner_slug; Type: INDEX; Schema: crm; Owner: -
--

CREATE INDEX idx_crm_partner_slug ON crm.partner USING btree (slug) WHERE (deleted_at IS NULL);


--
-- Name: idx_crm_referral_code; Type: INDEX; Schema: crm; Owner: -
--

CREATE INDEX idx_crm_referral_code ON crm.referral_code USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_crm_referral_partner; Type: INDEX; Schema: crm; Owner: -
--

CREATE INDEX idx_crm_referral_partner ON crm.referral_code USING btree (partner_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_identity_address_code; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_address_code ON identity.address USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_identity_address_owner; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_address_owner ON identity.address USING btree (owner_user_id, owner_type) WHERE (deleted_at IS NULL);


--
-- Name: idx_identity_buyer_profile_code; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_buyer_profile_code ON identity.buyer_profile USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_identity_buyer_profile_user_id; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_buyer_profile_user_id ON identity.buyer_profile USING btree (user_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_identity_credit_account_buyer; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_credit_account_buyer ON identity.credit_account USING btree (buyer_profile_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_identity_otp_user_purpose; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_otp_user_purpose ON identity.otp_code USING btree (user_id, purpose) WHERE (used_at IS NULL);


--
-- Name: idx_identity_password_reset_expires; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_password_reset_expires ON identity.password_reset_token USING btree (expires_at) WHERE (used_at IS NULL);


--
-- Name: idx_identity_password_reset_user; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_password_reset_user ON identity.password_reset_token USING btree (user_id) WHERE (used_at IS NULL);


--
-- Name: idx_identity_price_list_assignment_buyer; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_price_list_assignment_buyer ON identity.price_list_assignment USING btree (buyer_profile_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_identity_purchase_scope_buyer; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_purchase_scope_buyer ON identity.purchase_scope USING btree (buyer_profile_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_identity_purchase_scope_handling; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_purchase_scope_handling ON identity.purchase_scope USING btree (handling_class) WHERE (deleted_at IS NULL);


--
-- Name: idx_identity_refresh_token_expires; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_refresh_token_expires ON identity.refresh_token USING btree (expires_at) WHERE (revoked_at IS NULL);


--
-- Name: idx_identity_refresh_token_family; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_refresh_token_family ON identity.refresh_token USING btree (family_id) WHERE (revoked_at IS NULL);


--
-- Name: idx_identity_refresh_token_user; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_refresh_token_user ON identity.refresh_token USING btree (user_id) WHERE (revoked_at IS NULL);


--
-- Name: idx_identity_session_expires; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_session_expires ON identity.session USING btree (expires_at) WHERE (revoked_at IS NULL);


--
-- Name: idx_identity_session_user; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_session_user ON identity.session USING btree (user_id) WHERE (revoked_at IS NULL);


--
-- Name: idx_identity_supplier_profile_code; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_supplier_profile_code ON identity.supplier_profile USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_identity_supplier_profile_user_id; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_supplier_profile_user_id ON identity.supplier_profile USING btree (user_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_identity_user_code; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_user_code ON identity."user" USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_identity_user_email; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_user_email ON identity."user" USING btree (email) WHERE (deleted_at IS NULL);


--
-- Name: idx_identity_user_type; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_user_type ON identity."user" USING btree (user_type) WHERE (deleted_at IS NULL);


--
-- Name: idx_identity_verification_buyer; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_verification_buyer ON identity.verification_application USING btree (buyer_profile_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_identity_verification_code; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_verification_code ON identity.verification_application USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_identity_verification_doc_app; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_verification_doc_app ON identity.verification_document USING btree (verification_application_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_identity_verification_status; Type: INDEX; Schema: identity; Owner: -
--

CREATE INDEX idx_identity_verification_status ON identity.verification_application USING btree (status) WHERE (deleted_at IS NULL);


--
-- Name: idx_inventory_lot_expiry; Type: INDEX; Schema: inventory; Owner: -
--

CREATE INDEX idx_inventory_lot_expiry ON inventory.lot USING btree (expires_at, id) WHERE ((deleted_at IS NULL) AND (is_quarantined = false) AND (available_quantity > 0) AND (status = 'active'::inventory.lot_status));


--
-- Name: idx_inventory_lot_status; Type: INDEX; Schema: inventory; Owner: -
--

CREATE INDEX idx_inventory_lot_status ON inventory.lot USING btree (status, is_quarantined) WHERE (deleted_at IS NULL);


--
-- Name: idx_inventory_lot_stock_level; Type: INDEX; Schema: inventory; Owner: -
--

CREATE INDEX idx_inventory_lot_stock_level ON inventory.lot USING btree (stock_level_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_inventory_quarantine_lot; Type: INDEX; Schema: inventory; Owner: -
--

CREATE INDEX idx_inventory_quarantine_lot ON inventory.quarantine_record USING btree (lot_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_inventory_reservation_expiry; Type: INDEX; Schema: inventory; Owner: -
--

CREATE INDEX idx_inventory_reservation_expiry ON inventory.reservation USING btree (expires_at) WHERE ((status = 'reserved'::inventory.reservation_status) AND (deleted_at IS NULL));


--
-- Name: idx_inventory_reservation_lot; Type: INDEX; Schema: inventory; Owner: -
--

CREATE INDEX idx_inventory_reservation_lot ON inventory.reservation USING btree (lot_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_inventory_reservation_request_lot; Type: INDEX; Schema: inventory; Owner: -
--

CREATE UNIQUE INDEX idx_inventory_reservation_request_lot ON inventory.reservation USING btree (request_id, lot_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_inventory_stock_adjustment_lot; Type: INDEX; Schema: inventory; Owner: -
--

CREATE INDEX idx_inventory_stock_adjustment_lot ON inventory.stock_adjustment USING btree (lot_id);


--
-- Name: idx_inventory_stock_level_product; Type: INDEX; Schema: inventory; Owner: -
--

CREATE INDEX idx_inventory_stock_level_product ON inventory.stock_level USING btree (product_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_inventory_stock_level_supplier; Type: INDEX; Schema: inventory; Owner: -
--

CREATE INDEX idx_inventory_stock_level_supplier ON inventory.stock_level USING btree (supplier_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_payments_attempt_intent; Type: INDEX; Schema: payments; Owner: -
--

CREATE INDEX idx_payments_attempt_intent ON payments.payment_attempt USING btree (intent_id);


--
-- Name: idx_payments_intent_buyer; Type: INDEX; Schema: payments; Owner: -
--

CREATE INDEX idx_payments_intent_buyer ON payments.payment_intent USING btree (buyer_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_payments_intent_code; Type: INDEX; Schema: payments; Owner: -
--

CREATE INDEX idx_payments_intent_code ON payments.payment_intent USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_payments_intent_order; Type: INDEX; Schema: payments; Owner: -
--

CREATE INDEX idx_payments_intent_order ON payments.payment_intent USING btree (order_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_payments_method_buyer; Type: INDEX; Schema: payments; Owner: -
--

CREATE INDEX idx_payments_method_buyer ON payments.payment_method USING btree (buyer_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_payments_refund_code; Type: INDEX; Schema: payments; Owner: -
--

CREATE INDEX idx_payments_refund_code ON payments.payment_refund USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_payments_refund_intent; Type: INDEX; Schema: payments; Owner: -
--

CREATE INDEX idx_payments_refund_intent ON payments.payment_refund USING btree (intent_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_payments_webhook_provider; Type: INDEX; Schema: payments; Owner: -
--

CREATE INDEX idx_payments_webhook_provider ON payments.webhook_event USING btree (provider, created_at DESC);


--
-- Name: idx_audit_log_action; Type: INDEX; Schema: platform; Owner: -
--

CREATE INDEX idx_audit_log_action ON platform.audit_log USING btree (action);


--
-- Name: idx_audit_log_actor; Type: INDEX; Schema: platform; Owner: -
--

CREATE INDEX idx_audit_log_actor ON platform.audit_log USING btree (actor_type, actor_id);


--
-- Name: idx_audit_log_created; Type: INDEX; Schema: platform; Owner: -
--

CREATE INDEX idx_audit_log_created ON platform.audit_log USING btree (created_at DESC);


--
-- Name: idx_audit_log_resource; Type: INDEX; Schema: platform; Owner: -
--

CREATE INDEX idx_audit_log_resource ON platform.audit_log USING btree (resource_type, resource_id);


--
-- Name: idx_feature_flag_enabled; Type: INDEX; Schema: platform; Owner: -
--

CREATE INDEX idx_feature_flag_enabled ON platform.feature_flag USING btree (enabled) WHERE (enabled = true);


--
-- Name: idx_feature_flag_key; Type: INDEX; Schema: platform; Owner: -
--

CREATE INDEX idx_feature_flag_key ON platform.feature_flag USING btree (key);


--
-- Name: idx_integration_traffic_created; Type: INDEX; Schema: platform; Owner: -
--

CREATE INDEX idx_integration_traffic_created ON platform.integration_traffic USING btree (created_at DESC);


--
-- Name: idx_integration_traffic_direction; Type: INDEX; Schema: platform; Owner: -
--

CREATE INDEX idx_integration_traffic_direction ON platform.integration_traffic USING btree (direction);


--
-- Name: idx_integration_traffic_provider; Type: INDEX; Schema: platform; Owner: -
--

CREATE INDEX idx_integration_traffic_provider ON platform.integration_traffic USING btree (provider);


--
-- Name: idx_media_upload_code; Type: INDEX; Schema: platform; Owner: -
--

CREATE INDEX idx_media_upload_code ON platform.media_upload USING btree (code);


--
-- Name: idx_media_upload_status; Type: INDEX; Schema: platform; Owner: -
--

CREATE INDEX idx_media_upload_status ON platform.media_upload USING btree (status);


--
-- Name: idx_notification_created; Type: INDEX; Schema: platform; Owner: -
--

CREATE INDEX idx_notification_created ON platform.notification USING btree (created_at DESC) WHERE (deleted_at IS NULL);


--
-- Name: idx_notification_event_notification; Type: INDEX; Schema: platform; Owner: -
--

CREATE INDEX idx_notification_event_notification ON platform.notification_event USING btree (notification_id);


--
-- Name: idx_notification_event_type; Type: INDEX; Schema: platform; Owner: -
--

CREATE INDEX idx_notification_event_type ON platform.notification_event USING btree (event_type);


--
-- Name: idx_notification_recipient; Type: INDEX; Schema: platform; Owner: -
--

CREATE INDEX idx_notification_recipient ON platform.notification USING btree (recipient_type, recipient_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_notification_status; Type: INDEX; Schema: platform; Owner: -
--

CREATE INDEX idx_notification_status ON platform.notification USING btree (status) WHERE (deleted_at IS NULL);


--
-- Name: idx_notification_template; Type: INDEX; Schema: platform; Owner: -
--

CREATE INDEX idx_notification_template ON platform.notification USING btree (template_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_notification_template_channel; Type: INDEX; Schema: platform; Owner: -
--

CREATE INDEX idx_notification_template_channel ON platform.notification_template USING btree (channel) WHERE (deleted_at IS NULL);


--
-- Name: idx_notification_template_code; Type: INDEX; Schema: platform; Owner: -
--

CREATE INDEX idx_notification_template_code ON platform.notification_template USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_reference_region_active; Type: INDEX; Schema: platform; Owner: -
--

CREATE INDEX idx_reference_region_active ON platform.reference_region USING btree (is_active, name);


--
-- Name: idx_pricing_price_list_active; Type: INDEX; Schema: pricing; Owner: -
--

CREATE INDEX idx_pricing_price_list_active ON pricing.price_list USING btree (is_active) WHERE (deleted_at IS NULL);


--
-- Name: idx_pricing_price_list_assignment_buyer; Type: INDEX; Schema: pricing; Owner: -
--

CREATE INDEX idx_pricing_price_list_assignment_buyer ON pricing.price_list_assignment USING btree (buyer_profile_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_pricing_price_list_assignment_effective; Type: INDEX; Schema: pricing; Owner: -
--

CREATE INDEX idx_pricing_price_list_assignment_effective ON pricing.price_list_assignment USING btree (effective_from, effective_to) WHERE (deleted_at IS NULL);


--
-- Name: idx_pricing_price_list_assignment_list; Type: INDEX; Schema: pricing; Owner: -
--

CREATE INDEX idx_pricing_price_list_assignment_list ON pricing.price_list_assignment USING btree (price_list_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_pricing_price_list_code; Type: INDEX; Schema: pricing; Owner: -
--

CREATE INDEX idx_pricing_price_list_code ON pricing.price_list USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_pricing_price_list_effective; Type: INDEX; Schema: pricing; Owner: -
--

CREATE INDEX idx_pricing_price_list_effective ON pricing.price_list USING btree (effective_from, effective_to) WHERE (deleted_at IS NULL);


--
-- Name: idx_pricing_price_list_item_effective; Type: INDEX; Schema: pricing; Owner: -
--

CREATE INDEX idx_pricing_price_list_item_effective ON pricing.price_list_item USING btree (effective_from, effective_to) WHERE (deleted_at IS NULL);


--
-- Name: idx_pricing_price_list_item_list; Type: INDEX; Schema: pricing; Owner: -
--

CREATE INDEX idx_pricing_price_list_item_list ON pricing.price_list_item USING btree (price_list_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_pricing_price_list_item_product; Type: INDEX; Schema: pricing; Owner: -
--

CREATE INDEX idx_pricing_price_list_item_product ON pricing.price_list_item USING btree (product_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_pricing_price_list_item_quantity; Type: INDEX; Schema: pricing; Owner: -
--

CREATE INDEX idx_pricing_price_list_item_quantity ON pricing.price_list_item USING btree (min_quantity, max_quantity) WHERE (deleted_at IS NULL);


--
-- Name: idx_pricing_price_list_item_unit; Type: INDEX; Schema: pricing; Owner: -
--

CREATE INDEX idx_pricing_price_list_item_unit ON pricing.price_list_item USING btree (unit_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_pricing_price_list_status; Type: INDEX; Schema: pricing; Owner: -
--

CREATE INDEX idx_pricing_price_list_status ON pricing.price_list USING btree (status) WHERE (deleted_at IS NULL);


--
-- Name: idx_pricing_quantity_tier_item; Type: INDEX; Schema: pricing; Owner: -
--

CREATE INDEX idx_pricing_quantity_tier_item ON pricing.quantity_tier USING btree (price_list_item_id) WHERE (deleted_at IS NULL);


--
-- Name: idx_pricing_quantity_tier_quantity; Type: INDEX; Schema: pricing; Owner: -
--

CREATE INDEX idx_pricing_quantity_tier_quantity ON pricing.quantity_tier USING btree (min_quantity, max_quantity) WHERE (deleted_at IS NULL);


--
-- Name: idx_promotions_promotion_code; Type: INDEX; Schema: promotions; Owner: -
--

CREATE INDEX idx_promotions_promotion_code ON promotions.promotion USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_promotions_promotion_status; Type: INDEX; Schema: promotions; Owner: -
--

CREATE INDEX idx_promotions_promotion_status ON promotions.promotion USING btree (status) WHERE (deleted_at IS NULL);


--
-- Name: idx_promotions_redemption_buyer; Type: INDEX; Schema: promotions; Owner: -
--

CREATE INDEX idx_promotions_redemption_buyer ON promotions.voucher_redemption USING btree (buyer_id);


--
-- Name: idx_promotions_redemption_order; Type: INDEX; Schema: promotions; Owner: -
--

CREATE INDEX idx_promotions_redemption_order ON promotions.voucher_redemption USING btree (order_id);


--
-- Name: idx_promotions_redemption_promo; Type: INDEX; Schema: promotions; Owner: -
--

CREATE INDEX idx_promotions_redemption_promo ON promotions.voucher_redemption USING btree (promotion_id);


--
-- Name: idx_suppliers_contract_current; Type: INDEX; Schema: suppliers; Owner: -
--

CREATE INDEX idx_suppliers_contract_current ON suppliers.supplier_contract USING btree (supplier_id) WHERE is_current;


--
-- Name: idx_suppliers_contract_supplier; Type: INDEX; Schema: suppliers; Owner: -
--

CREATE INDEX idx_suppliers_contract_supplier ON suppliers.supplier_contract USING btree (supplier_id);


--
-- Name: idx_suppliers_profile_code; Type: INDEX; Schema: suppliers; Owner: -
--

CREATE INDEX idx_suppliers_profile_code ON suppliers.supplier_profile USING btree (code) WHERE (deleted_at IS NULL);


--
-- Name: idx_suppliers_profile_status; Type: INDEX; Schema: suppliers; Owner: -
--

CREATE INDEX idx_suppliers_profile_status ON suppliers.supplier_profile USING btree (status) WHERE (deleted_at IS NULL);


--
-- Name: idx_suppliers_profile_user; Type: INDEX; Schema: suppliers; Owner: -
--

CREATE INDEX idx_suppliers_profile_user ON suppliers.supplier_profile USING btree (user_id) WHERE (deleted_at IS NULL);


--
-- Name: attribute update_attribute_updated_at; Type: TRIGGER; Schema: catalog; Owner: -
--

CREATE TRIGGER update_attribute_updated_at BEFORE UPDATE ON catalog.attribute FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();


--
-- Name: attribute_value update_attribute_value_updated_at; Type: TRIGGER; Schema: catalog; Owner: -
--

CREATE TRIGGER update_attribute_value_updated_at BEFORE UPDATE ON catalog.attribute_value FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();


--
-- Name: brand update_brand_updated_at; Type: TRIGGER; Schema: catalog; Owner: -
--

CREATE TRIGGER update_brand_updated_at BEFORE UPDATE ON catalog.brand FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();


--
-- Name: category update_category_updated_at; Type: TRIGGER; Schema: catalog; Owner: -
--

CREATE TRIGGER update_category_updated_at BEFORE UPDATE ON catalog.category FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();


--
-- Name: handling_class update_handling_class_updated_at; Type: TRIGGER; Schema: catalog; Owner: -
--

CREATE TRIGGER update_handling_class_updated_at BEFORE UPDATE ON catalog.handling_class FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();


--
-- Name: product_attribute update_product_attribute_updated_at; Type: TRIGGER; Schema: catalog; Owner: -
--

CREATE TRIGGER update_product_attribute_updated_at BEFORE UPDATE ON catalog.product_attribute FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();


--
-- Name: product_media update_product_media_updated_at; Type: TRIGGER; Schema: catalog; Owner: -
--

CREATE TRIGGER update_product_media_updated_at BEFORE UPDATE ON catalog.product_media FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();


--
-- Name: product_unit update_product_unit_updated_at; Type: TRIGGER; Schema: catalog; Owner: -
--

CREATE TRIGGER update_product_unit_updated_at BEFORE UPDATE ON catalog.product_unit FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();


--
-- Name: product update_product_updated_at; Type: TRIGGER; Schema: catalog; Owner: -
--

CREATE TRIGGER update_product_updated_at BEFORE UPDATE ON catalog.product FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();


--
-- Name: supplier update_supplier_updated_at; Type: TRIGGER; Schema: catalog; Owner: -
--

CREATE TRIGGER update_supplier_updated_at BEFORE UPDATE ON catalog.supplier FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();


--
-- Name: unit update_unit_updated_at; Type: TRIGGER; Schema: catalog; Owner: -
--

CREATE TRIGGER update_unit_updated_at BEFORE UPDATE ON catalog.unit FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();


--
-- Name: cart_line update_cart_line_updated_at; Type: TRIGGER; Schema: commerce; Owner: -
--

CREATE TRIGGER update_cart_line_updated_at BEFORE UPDATE ON commerce.cart_line FOR EACH ROW EXECUTE FUNCTION commerce.update_updated_at_column();


--
-- Name: cart update_cart_updated_at; Type: TRIGGER; Schema: commerce; Owner: -
--

CREATE TRIGGER update_cart_updated_at BEFORE UPDATE ON commerce.cart FOR EACH ROW EXECUTE FUNCTION commerce.update_updated_at_column();


--
-- Name: checkout_idempotency update_checkout_idempotency_updated_at; Type: TRIGGER; Schema: commerce; Owner: -
--

CREATE TRIGGER update_checkout_idempotency_updated_at BEFORE UPDATE ON commerce.checkout_idempotency FOR EACH ROW EXECUTE FUNCTION commerce.update_updated_at_column();


--
-- Name: credit_note update_credit_note_updated_at; Type: TRIGGER; Schema: commerce; Owner: -
--

CREATE TRIGGER update_credit_note_updated_at BEFORE UPDATE ON commerce.credit_note FOR EACH ROW EXECUTE FUNCTION commerce.update_updated_at_column();


--
-- Name: invoice update_invoice_updated_at; Type: TRIGGER; Schema: commerce; Owner: -
--

CREATE TRIGGER update_invoice_updated_at BEFORE UPDATE ON commerce.invoice FOR EACH ROW EXECUTE FUNCTION commerce.update_updated_at_column();


--
-- Name: order_line update_order_line_updated_at; Type: TRIGGER; Schema: commerce; Owner: -
--

CREATE TRIGGER update_order_line_updated_at BEFORE UPDATE ON commerce.order_line FOR EACH ROW EXECUTE FUNCTION commerce.update_updated_at_column();


--
-- Name: order update_order_updated_at; Type: TRIGGER; Schema: commerce; Owner: -
--

CREATE TRIGGER update_order_updated_at BEFORE UPDATE ON commerce."order" FOR EACH ROW EXECUTE FUNCTION commerce.update_updated_at_column();


--
-- Name: return_request update_return_updated_at; Type: TRIGGER; Schema: commerce; Owner: -
--

CREATE TRIGGER update_return_updated_at BEFORE UPDATE ON commerce.return_request FOR EACH ROW EXECUTE FUNCTION commerce.update_updated_at_column();


--
-- Name: shipment update_shipment_updated_at; Type: TRIGGER; Schema: commerce; Owner: -
--

CREATE TRIGGER update_shipment_updated_at BEFORE UPDATE ON commerce.shipment FOR EACH ROW EXECUTE FUNCTION commerce.update_updated_at_column();


--
-- Name: address update_address_updated_at; Type: TRIGGER; Schema: identity; Owner: -
--

CREATE TRIGGER update_address_updated_at BEFORE UPDATE ON identity.address FOR EACH ROW EXECUTE FUNCTION identity.update_updated_at_column();


--
-- Name: buyer_profile update_buyer_profile_updated_at; Type: TRIGGER; Schema: identity; Owner: -
--

CREATE TRIGGER update_buyer_profile_updated_at BEFORE UPDATE ON identity.buyer_profile FOR EACH ROW EXECUTE FUNCTION identity.update_updated_at_column();


--
-- Name: credit_account update_credit_account_updated_at; Type: TRIGGER; Schema: identity; Owner: -
--

CREATE TRIGGER update_credit_account_updated_at BEFORE UPDATE ON identity.credit_account FOR EACH ROW EXECUTE FUNCTION identity.update_updated_at_column();


--
-- Name: permission update_permission_updated_at; Type: TRIGGER; Schema: identity; Owner: -
--

CREATE TRIGGER update_permission_updated_at BEFORE UPDATE ON identity.permission FOR EACH ROW EXECUTE FUNCTION identity.update_updated_at_column();


--
-- Name: role update_role_updated_at; Type: TRIGGER; Schema: identity; Owner: -
--

CREATE TRIGGER update_role_updated_at BEFORE UPDATE ON identity.role FOR EACH ROW EXECUTE FUNCTION identity.update_updated_at_column();


--
-- Name: supplier_profile update_supplier_profile_updated_at; Type: TRIGGER; Schema: identity; Owner: -
--

CREATE TRIGGER update_supplier_profile_updated_at BEFORE UPDATE ON identity.supplier_profile FOR EACH ROW EXECUTE FUNCTION identity.update_updated_at_column();


--
-- Name: user update_user_updated_at; Type: TRIGGER; Schema: identity; Owner: -
--

CREATE TRIGGER update_user_updated_at BEFORE UPDATE ON identity."user" FOR EACH ROW EXECUTE FUNCTION identity.update_updated_at_column();


--
-- Name: verification_application update_verification_application_updated_at; Type: TRIGGER; Schema: identity; Owner: -
--

CREATE TRIGGER update_verification_application_updated_at BEFORE UPDATE ON identity.verification_application FOR EACH ROW EXECUTE FUNCTION identity.update_updated_at_column();


--
-- Name: lot update_lot_updated_at; Type: TRIGGER; Schema: inventory; Owner: -
--

CREATE TRIGGER update_lot_updated_at BEFORE UPDATE ON inventory.lot FOR EACH ROW EXECUTE FUNCTION inventory.update_updated_at_column();


--
-- Name: quarantine_record update_quarantine_record_updated_at; Type: TRIGGER; Schema: inventory; Owner: -
--

CREATE TRIGGER update_quarantine_record_updated_at BEFORE UPDATE ON inventory.quarantine_record FOR EACH ROW EXECUTE FUNCTION inventory.update_updated_at_column();


--
-- Name: reservation update_reservation_updated_at; Type: TRIGGER; Schema: inventory; Owner: -
--

CREATE TRIGGER update_reservation_updated_at BEFORE UPDATE ON inventory.reservation FOR EACH ROW EXECUTE FUNCTION inventory.update_updated_at_column();


--
-- Name: stock_level update_stock_level_updated_at; Type: TRIGGER; Schema: inventory; Owner: -
--

CREATE TRIGGER update_stock_level_updated_at BEFORE UPDATE ON inventory.stock_level FOR EACH ROW EXECUTE FUNCTION inventory.update_updated_at_column();


--
-- Name: price_list_assignment update_price_list_assignment_updated_at; Type: TRIGGER; Schema: pricing; Owner: -
--

CREATE TRIGGER update_price_list_assignment_updated_at BEFORE UPDATE ON pricing.price_list_assignment FOR EACH ROW EXECUTE FUNCTION pricing.update_updated_at_column();


--
-- Name: price_list_item update_price_list_item_updated_at; Type: TRIGGER; Schema: pricing; Owner: -
--

CREATE TRIGGER update_price_list_item_updated_at BEFORE UPDATE ON pricing.price_list_item FOR EACH ROW EXECUTE FUNCTION pricing.update_updated_at_column();


--
-- Name: price_list update_price_list_updated_at; Type: TRIGGER; Schema: pricing; Owner: -
--

CREATE TRIGGER update_price_list_updated_at BEFORE UPDATE ON pricing.price_list FOR EACH ROW EXECUTE FUNCTION pricing.update_updated_at_column();


--
-- Name: quantity_tier update_quantity_tier_updated_at; Type: TRIGGER; Schema: pricing; Owner: -
--

CREATE TRIGGER update_quantity_tier_updated_at BEFORE UPDATE ON pricing.quantity_tier FOR EACH ROW EXECUTE FUNCTION pricing.update_updated_at_column();


--
-- Name: promotion update_promotion_updated_at; Type: TRIGGER; Schema: promotions; Owner: -
--

CREATE TRIGGER update_promotion_updated_at BEFORE UPDATE ON promotions.promotion FOR EACH ROW EXECUTE FUNCTION commerce.update_updated_at_column();


--
-- Name: prompt_version prompt_version_prompt_id_fkey; Type: FK CONSTRAINT; Schema: ai; Owner: -
--

ALTER TABLE ONLY ai.prompt_version
    ADD CONSTRAINT prompt_version_prompt_id_fkey FOREIGN KEY (prompt_id) REFERENCES ai.prompt(id) ON DELETE CASCADE;


--
-- Name: usage_record usage_record_proposal_id_fkey; Type: FK CONSTRAINT; Schema: ai; Owner: -
--

ALTER TABLE ONLY ai.usage_record
    ADD CONSTRAINT usage_record_proposal_id_fkey FOREIGN KEY (proposal_id) REFERENCES ai.proposal(id) ON DELETE SET NULL;


--
-- Name: attribute_value attribute_value_attribute_id_fkey; Type: FK CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.attribute_value
    ADD CONSTRAINT attribute_value_attribute_id_fkey FOREIGN KEY (attribute_id) REFERENCES catalog.attribute(id) ON DELETE CASCADE;


--
-- Name: category category_parent_id_fkey; Type: FK CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.category
    ADD CONSTRAINT category_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES catalog.category(id) ON DELETE SET NULL;


--
-- Name: product_attribute product_attribute_attribute_id_fkey; Type: FK CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product_attribute
    ADD CONSTRAINT product_attribute_attribute_id_fkey FOREIGN KEY (attribute_id) REFERENCES catalog.attribute(id) ON DELETE CASCADE;


--
-- Name: product_attribute product_attribute_attribute_value_id_fkey; Type: FK CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product_attribute
    ADD CONSTRAINT product_attribute_attribute_value_id_fkey FOREIGN KEY (attribute_value_id) REFERENCES catalog.attribute_value(id) ON DELETE SET NULL;


--
-- Name: product_attribute product_attribute_product_id_fkey; Type: FK CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product_attribute
    ADD CONSTRAINT product_attribute_product_id_fkey FOREIGN KEY (product_id) REFERENCES catalog.product(id) ON DELETE CASCADE;


--
-- Name: product product_base_unit_id_fkey; Type: FK CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product
    ADD CONSTRAINT product_base_unit_id_fkey FOREIGN KEY (base_unit_id) REFERENCES catalog.unit(id) ON DELETE RESTRICT;


--
-- Name: product product_brand_id_fkey; Type: FK CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product
    ADD CONSTRAINT product_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES catalog.brand(id) ON DELETE SET NULL;


--
-- Name: product product_category_id_fkey; Type: FK CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product
    ADD CONSTRAINT product_category_id_fkey FOREIGN KEY (category_id) REFERENCES catalog.category(id) ON DELETE RESTRICT;


--
-- Name: product_media product_media_product_id_fkey; Type: FK CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product_media
    ADD CONSTRAINT product_media_product_id_fkey FOREIGN KEY (product_id) REFERENCES catalog.product(id) ON DELETE CASCADE;


--
-- Name: product product_supplier_id_fkey; Type: FK CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product
    ADD CONSTRAINT product_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES catalog.supplier(id) ON DELETE SET NULL;


--
-- Name: product_unit product_unit_product_id_fkey; Type: FK CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product_unit
    ADD CONSTRAINT product_unit_product_id_fkey FOREIGN KEY (product_id) REFERENCES catalog.product(id) ON DELETE CASCADE;


--
-- Name: product_unit product_unit_unit_id_fkey; Type: FK CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.product_unit
    ADD CONSTRAINT product_unit_unit_id_fkey FOREIGN KEY (unit_id) REFERENCES catalog.unit(id) ON DELETE RESTRICT;


--
-- Name: unit unit_base_unit_id_fkey; Type: FK CONSTRAINT; Schema: catalog; Owner: -
--

ALTER TABLE ONLY catalog.unit
    ADD CONSTRAINT unit_base_unit_id_fkey FOREIGN KEY (base_unit_id) REFERENCES catalog.unit(id) ON DELETE SET NULL;


--
-- Name: legal_document_version legal_document_version_doc_type_fkey; Type: FK CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.legal_document_version
    ADD CONSTRAINT legal_document_version_doc_type_fkey FOREIGN KEY (doc_type) REFERENCES cms.legal_document(doc_type);


--
-- Name: menu_item menu_item_parent_id_fkey; Type: FK CONSTRAINT; Schema: cms; Owner: -
--

ALTER TABLE ONLY cms.menu_item
    ADD CONSTRAINT menu_item_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES cms.menu_item(id) ON DELETE CASCADE;


--
-- Name: cart cart_buyer_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.cart
    ADD CONSTRAINT cart_buyer_id_fkey FOREIGN KEY (buyer_id) REFERENCES identity.buyer_profile(id) ON DELETE CASCADE;


--
-- Name: cart_line cart_line_cart_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.cart_line
    ADD CONSTRAINT cart_line_cart_id_fkey FOREIGN KEY (cart_id) REFERENCES commerce.cart(id) ON DELETE CASCADE;


--
-- Name: cart_line cart_line_product_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.cart_line
    ADD CONSTRAINT cart_line_product_id_fkey FOREIGN KEY (product_id) REFERENCES catalog.product(id) ON DELETE RESTRICT;


--
-- Name: cart_line cart_line_supplier_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.cart_line
    ADD CONSTRAINT cart_line_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES catalog.supplier(id) ON DELETE RESTRICT;


--
-- Name: cart_line cart_line_unit_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.cart_line
    ADD CONSTRAINT cart_line_unit_id_fkey FOREIGN KEY (unit_id) REFERENCES catalog.unit(id) ON DELETE RESTRICT;


--
-- Name: checkout_idempotency checkout_idempotency_buyer_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.checkout_idempotency
    ADD CONSTRAINT checkout_idempotency_buyer_id_fkey FOREIGN KEY (buyer_id) REFERENCES identity.buyer_profile(id) ON DELETE CASCADE;


--
-- Name: credit_note credit_note_invoice_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.credit_note
    ADD CONSTRAINT credit_note_invoice_id_fkey FOREIGN KEY (invoice_id) REFERENCES commerce.invoice(id) ON DELETE SET NULL;


--
-- Name: credit_note credit_note_order_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.credit_note
    ADD CONSTRAINT credit_note_order_id_fkey FOREIGN KEY (order_id) REFERENCES commerce."order"(id) ON DELETE RESTRICT;


--
-- Name: credit_note credit_note_return_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.credit_note
    ADD CONSTRAINT credit_note_return_id_fkey FOREIGN KEY (return_id) REFERENCES commerce.return_request(id) ON DELETE SET NULL;


--
-- Name: invoice invoice_order_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.invoice
    ADD CONSTRAINT invoice_order_id_fkey FOREIGN KEY (order_id) REFERENCES commerce."order"(id) ON DELETE RESTRICT;


--
-- Name: order order_buyer_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce."order"
    ADD CONSTRAINT order_buyer_id_fkey FOREIGN KEY (buyer_id) REFERENCES identity.buyer_profile(id) ON DELETE RESTRICT;


--
-- Name: order order_cart_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce."order"
    ADD CONSTRAINT order_cart_id_fkey FOREIGN KEY (cart_id) REFERENCES commerce.cart(id) ON DELETE SET NULL;


--
-- Name: order_history order_history_order_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.order_history
    ADD CONSTRAINT order_history_order_id_fkey FOREIGN KEY (order_id) REFERENCES commerce."order"(id) ON DELETE CASCADE;


--
-- Name: order_line order_line_order_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.order_line
    ADD CONSTRAINT order_line_order_id_fkey FOREIGN KEY (order_id) REFERENCES commerce."order"(id) ON DELETE CASCADE;


--
-- Name: order_line order_line_product_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.order_line
    ADD CONSTRAINT order_line_product_id_fkey FOREIGN KEY (product_id) REFERENCES catalog.product(id) ON DELETE RESTRICT;


--
-- Name: order_line order_line_supplier_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.order_line
    ADD CONSTRAINT order_line_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES catalog.supplier(id) ON DELETE RESTRICT;


--
-- Name: order_line order_line_unit_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.order_line
    ADD CONSTRAINT order_line_unit_id_fkey FOREIGN KEY (unit_id) REFERENCES catalog.unit(id) ON DELETE RESTRICT;


--
-- Name: order order_supplier_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce."order"
    ADD CONSTRAINT order_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES catalog.supplier(id) ON DELETE RESTRICT;


--
-- Name: return_line return_line_order_line_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.return_line
    ADD CONSTRAINT return_line_order_line_id_fkey FOREIGN KEY (order_line_id) REFERENCES commerce.order_line(id) ON DELETE RESTRICT;


--
-- Name: return_line return_line_return_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.return_line
    ADD CONSTRAINT return_line_return_id_fkey FOREIGN KEY (return_id) REFERENCES commerce.return_request(id) ON DELETE CASCADE;


--
-- Name: return_request return_request_buyer_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.return_request
    ADD CONSTRAINT return_request_buyer_id_fkey FOREIGN KEY (buyer_id) REFERENCES identity.buyer_profile(id) ON DELETE RESTRICT;


--
-- Name: return_request return_request_order_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.return_request
    ADD CONSTRAINT return_request_order_id_fkey FOREIGN KEY (order_id) REFERENCES commerce."order"(id) ON DELETE RESTRICT;


--
-- Name: shipment_line shipment_line_order_line_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.shipment_line
    ADD CONSTRAINT shipment_line_order_line_id_fkey FOREIGN KEY (order_line_id) REFERENCES commerce.order_line(id) ON DELETE RESTRICT;


--
-- Name: shipment_line shipment_line_shipment_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.shipment_line
    ADD CONSTRAINT shipment_line_shipment_id_fkey FOREIGN KEY (shipment_id) REFERENCES commerce.shipment(id) ON DELETE CASCADE;


--
-- Name: shipment shipment_order_id_fkey; Type: FK CONSTRAINT; Schema: commerce; Owner: -
--

ALTER TABLE ONLY commerce.shipment
    ADD CONSTRAINT shipment_order_id_fkey FOREIGN KEY (order_id) REFERENCES commerce."order"(id) ON DELETE CASCADE;


--
-- Name: lead lead_buyer_id_fkey; Type: FK CONSTRAINT; Schema: crm; Owner: -
--

ALTER TABLE ONLY crm.lead
    ADD CONSTRAINT lead_buyer_id_fkey FOREIGN KEY (buyer_id) REFERENCES identity.buyer_profile(id) ON DELETE SET NULL;


--
-- Name: lead lead_partner_id_fkey; Type: FK CONSTRAINT; Schema: crm; Owner: -
--

ALTER TABLE ONLY crm.lead
    ADD CONSTRAINT lead_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES crm.partner(id) ON DELETE SET NULL;


--
-- Name: lead lead_referral_id_fkey; Type: FK CONSTRAINT; Schema: crm; Owner: -
--

ALTER TABLE ONLY crm.lead
    ADD CONSTRAINT lead_referral_id_fkey FOREIGN KEY (referral_id) REFERENCES crm.referral_code(id) ON DELETE SET NULL;


--
-- Name: referral_code referral_code_partner_id_fkey; Type: FK CONSTRAINT; Schema: crm; Owner: -
--

ALTER TABLE ONLY crm.referral_code
    ADD CONSTRAINT referral_code_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES crm.partner(id) ON DELETE RESTRICT;


--
-- Name: address address_owner_user_id_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.address
    ADD CONSTRAINT address_owner_user_id_fkey FOREIGN KEY (owner_user_id) REFERENCES identity."user"(id) ON DELETE CASCADE;


--
-- Name: buyer_profile buyer_profile_user_id_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.buyer_profile
    ADD CONSTRAINT buyer_profile_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity."user"(id) ON DELETE CASCADE;


--
-- Name: credit_account credit_account_buyer_profile_id_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.credit_account
    ADD CONSTRAINT credit_account_buyer_profile_id_fkey FOREIGN KEY (buyer_profile_id) REFERENCES identity.buyer_profile(id) ON DELETE CASCADE;


--
-- Name: otp_code otp_code_user_id_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.otp_code
    ADD CONSTRAINT otp_code_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity."user"(id) ON DELETE CASCADE;


--
-- Name: password_reset_token password_reset_token_user_id_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.password_reset_token
    ADD CONSTRAINT password_reset_token_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity."user"(id) ON DELETE CASCADE;


--
-- Name: price_list_assignment price_list_assignment_assigned_by_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.price_list_assignment
    ADD CONSTRAINT price_list_assignment_assigned_by_fkey FOREIGN KEY (assigned_by) REFERENCES identity."user"(id);


--
-- Name: price_list_assignment price_list_assignment_buyer_profile_id_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.price_list_assignment
    ADD CONSTRAINT price_list_assignment_buyer_profile_id_fkey FOREIGN KEY (buyer_profile_id) REFERENCES identity.buyer_profile(id) ON DELETE CASCADE;


--
-- Name: purchase_scope purchase_scope_buyer_profile_id_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.purchase_scope
    ADD CONSTRAINT purchase_scope_buyer_profile_id_fkey FOREIGN KEY (buyer_profile_id) REFERENCES identity.buyer_profile(id) ON DELETE CASCADE;


--
-- Name: purchase_scope purchase_scope_granted_by_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.purchase_scope
    ADD CONSTRAINT purchase_scope_granted_by_fkey FOREIGN KEY (granted_by) REFERENCES identity."user"(id);


--
-- Name: refresh_token refresh_token_replaced_by_id_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.refresh_token
    ADD CONSTRAINT refresh_token_replaced_by_id_fkey FOREIGN KEY (replaced_by_id) REFERENCES identity.refresh_token(id);


--
-- Name: refresh_token refresh_token_user_id_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.refresh_token
    ADD CONSTRAINT refresh_token_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity."user"(id) ON DELETE CASCADE;


--
-- Name: role_permission role_permission_permission_id_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.role_permission
    ADD CONSTRAINT role_permission_permission_id_fkey FOREIGN KEY (permission_id) REFERENCES identity.permission(id) ON DELETE CASCADE;


--
-- Name: role_permission role_permission_role_id_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.role_permission
    ADD CONSTRAINT role_permission_role_id_fkey FOREIGN KEY (role_id) REFERENCES identity.role(id) ON DELETE CASCADE;


--
-- Name: session session_refresh_token_id_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.session
    ADD CONSTRAINT session_refresh_token_id_fkey FOREIGN KEY (refresh_token_id) REFERENCES identity.refresh_token(id) ON DELETE SET NULL;


--
-- Name: session session_user_id_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.session
    ADD CONSTRAINT session_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity."user"(id) ON DELETE CASCADE;


--
-- Name: supplier_profile supplier_profile_approved_by_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.supplier_profile
    ADD CONSTRAINT supplier_profile_approved_by_fkey FOREIGN KEY (approved_by) REFERENCES identity."user"(id);


--
-- Name: supplier_profile supplier_profile_user_id_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.supplier_profile
    ADD CONSTRAINT supplier_profile_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity."user"(id) ON DELETE CASCADE;


--
-- Name: user_role user_role_assigned_by_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.user_role
    ADD CONSTRAINT user_role_assigned_by_fkey FOREIGN KEY (assigned_by) REFERENCES identity."user"(id);


--
-- Name: user_role user_role_role_id_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.user_role
    ADD CONSTRAINT user_role_role_id_fkey FOREIGN KEY (role_id) REFERENCES identity.role(id) ON DELETE CASCADE;


--
-- Name: user_role user_role_user_id_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.user_role
    ADD CONSTRAINT user_role_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity."user"(id) ON DELETE CASCADE;


--
-- Name: verification_application verification_application_buyer_profile_id_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.verification_application
    ADD CONSTRAINT verification_application_buyer_profile_id_fkey FOREIGN KEY (buyer_profile_id) REFERENCES identity.buyer_profile(id) ON DELETE CASCADE;


--
-- Name: verification_application verification_application_decided_by_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.verification_application
    ADD CONSTRAINT verification_application_decided_by_fkey FOREIGN KEY (decided_by) REFERENCES identity."user"(id);


--
-- Name: verification_document verification_document_uploaded_by_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.verification_document
    ADD CONSTRAINT verification_document_uploaded_by_fkey FOREIGN KEY (uploaded_by) REFERENCES identity."user"(id);


--
-- Name: verification_document verification_document_verification_application_id_fkey; Type: FK CONSTRAINT; Schema: identity; Owner: -
--

ALTER TABLE ONLY identity.verification_document
    ADD CONSTRAINT verification_document_verification_application_id_fkey FOREIGN KEY (verification_application_id) REFERENCES identity.verification_application(id) ON DELETE CASCADE;


--
-- Name: lot lot_stock_level_id_fkey; Type: FK CONSTRAINT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.lot
    ADD CONSTRAINT lot_stock_level_id_fkey FOREIGN KEY (stock_level_id) REFERENCES inventory.stock_level(id) ON DELETE CASCADE;


--
-- Name: quarantine_record quarantine_record_adjusted_by_fkey; Type: FK CONSTRAINT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.quarantine_record
    ADD CONSTRAINT quarantine_record_adjusted_by_fkey FOREIGN KEY (adjusted_by) REFERENCES identity."user"(id);


--
-- Name: quarantine_record quarantine_record_lot_id_fkey; Type: FK CONSTRAINT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.quarantine_record
    ADD CONSTRAINT quarantine_record_lot_id_fkey FOREIGN KEY (lot_id) REFERENCES inventory.lot(id) ON DELETE CASCADE;


--
-- Name: reservation reservation_lot_id_fkey; Type: FK CONSTRAINT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.reservation
    ADD CONSTRAINT reservation_lot_id_fkey FOREIGN KEY (lot_id) REFERENCES inventory.lot(id) ON DELETE CASCADE;


--
-- Name: stock_adjustment stock_adjustment_adjusted_by_fkey; Type: FK CONSTRAINT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.stock_adjustment
    ADD CONSTRAINT stock_adjustment_adjusted_by_fkey FOREIGN KEY (adjusted_by) REFERENCES identity."user"(id);


--
-- Name: stock_adjustment stock_adjustment_lot_id_fkey; Type: FK CONSTRAINT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.stock_adjustment
    ADD CONSTRAINT stock_adjustment_lot_id_fkey FOREIGN KEY (lot_id) REFERENCES inventory.lot(id) ON DELETE CASCADE;


--
-- Name: stock_level stock_level_product_id_fkey; Type: FK CONSTRAINT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.stock_level
    ADD CONSTRAINT stock_level_product_id_fkey FOREIGN KEY (product_id) REFERENCES catalog.product(id) ON DELETE RESTRICT;


--
-- Name: stock_level stock_level_supplier_id_fkey; Type: FK CONSTRAINT; Schema: inventory; Owner: -
--

ALTER TABLE ONLY inventory.stock_level
    ADD CONSTRAINT stock_level_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES catalog.supplier(id) ON DELETE RESTRICT;


--
-- Name: credit_account credit_account_buyer_id_fkey; Type: FK CONSTRAINT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.credit_account
    ADD CONSTRAINT credit_account_buyer_id_fkey FOREIGN KEY (buyer_id) REFERENCES identity.buyer_profile(id) ON DELETE CASCADE;


--
-- Name: payment_attempt payment_attempt_intent_id_fkey; Type: FK CONSTRAINT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.payment_attempt
    ADD CONSTRAINT payment_attempt_intent_id_fkey FOREIGN KEY (intent_id) REFERENCES payments.payment_intent(id) ON DELETE CASCADE;


--
-- Name: payment_intent payment_intent_buyer_id_fkey; Type: FK CONSTRAINT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.payment_intent
    ADD CONSTRAINT payment_intent_buyer_id_fkey FOREIGN KEY (buyer_id) REFERENCES identity.buyer_profile(id) ON DELETE RESTRICT;


--
-- Name: payment_intent payment_intent_order_id_fkey; Type: FK CONSTRAINT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.payment_intent
    ADD CONSTRAINT payment_intent_order_id_fkey FOREIGN KEY (order_id) REFERENCES commerce."order"(id) ON DELETE RESTRICT;


--
-- Name: payment_intent payment_intent_payment_method_id_fkey; Type: FK CONSTRAINT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.payment_intent
    ADD CONSTRAINT payment_intent_payment_method_id_fkey FOREIGN KEY (payment_method_id) REFERENCES payments.payment_method(id) ON DELETE SET NULL;


--
-- Name: payment_method payment_method_buyer_id_fkey; Type: FK CONSTRAINT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.payment_method
    ADD CONSTRAINT payment_method_buyer_id_fkey FOREIGN KEY (buyer_id) REFERENCES identity.buyer_profile(id) ON DELETE CASCADE;


--
-- Name: payment_refund payment_refund_intent_id_fkey; Type: FK CONSTRAINT; Schema: payments; Owner: -
--

ALTER TABLE ONLY payments.payment_refund
    ADD CONSTRAINT payment_refund_intent_id_fkey FOREIGN KEY (intent_id) REFERENCES payments.payment_intent(id) ON DELETE RESTRICT;


--
-- Name: notification_event notification_event_notification_id_fkey; Type: FK CONSTRAINT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.notification_event
    ADD CONSTRAINT notification_event_notification_id_fkey FOREIGN KEY (notification_id) REFERENCES platform.notification(id) ON DELETE CASCADE;


--
-- Name: notification notification_template_id_fkey; Type: FK CONSTRAINT; Schema: platform; Owner: -
--

ALTER TABLE ONLY platform.notification
    ADD CONSTRAINT notification_template_id_fkey FOREIGN KEY (template_id) REFERENCES platform.notification_template(id) ON DELETE RESTRICT;


--
-- Name: price_list_assignment price_list_assignment_price_list_id_fkey; Type: FK CONSTRAINT; Schema: pricing; Owner: -
--

ALTER TABLE ONLY pricing.price_list_assignment
    ADD CONSTRAINT price_list_assignment_price_list_id_fkey FOREIGN KEY (price_list_id) REFERENCES pricing.price_list(id) ON DELETE CASCADE;


--
-- Name: price_list_item price_list_item_price_list_id_fkey; Type: FK CONSTRAINT; Schema: pricing; Owner: -
--

ALTER TABLE ONLY pricing.price_list_item
    ADD CONSTRAINT price_list_item_price_list_id_fkey FOREIGN KEY (price_list_id) REFERENCES pricing.price_list(id) ON DELETE CASCADE;


--
-- Name: quantity_tier quantity_tier_price_list_item_id_fkey; Type: FK CONSTRAINT; Schema: pricing; Owner: -
--

ALTER TABLE ONLY pricing.quantity_tier
    ADD CONSTRAINT quantity_tier_price_list_item_id_fkey FOREIGN KEY (price_list_item_id) REFERENCES pricing.price_list_item(id) ON DELETE CASCADE;


--
-- Name: voucher_redemption voucher_redemption_buyer_id_fkey; Type: FK CONSTRAINT; Schema: promotions; Owner: -
--

ALTER TABLE ONLY promotions.voucher_redemption
    ADD CONSTRAINT voucher_redemption_buyer_id_fkey FOREIGN KEY (buyer_id) REFERENCES identity.buyer_profile(id) ON DELETE RESTRICT;


--
-- Name: voucher_redemption voucher_redemption_order_id_fkey; Type: FK CONSTRAINT; Schema: promotions; Owner: -
--

ALTER TABLE ONLY promotions.voucher_redemption
    ADD CONSTRAINT voucher_redemption_order_id_fkey FOREIGN KEY (order_id) REFERENCES commerce."order"(id) ON DELETE RESTRICT;


--
-- Name: voucher_redemption voucher_redemption_promotion_id_fkey; Type: FK CONSTRAINT; Schema: promotions; Owner: -
--

ALTER TABLE ONLY promotions.voucher_redemption
    ADD CONSTRAINT voucher_redemption_promotion_id_fkey FOREIGN KEY (promotion_id) REFERENCES promotions.promotion(id) ON DELETE RESTRICT;


--
-- Name: supplier_contract supplier_contract_supplier_id_fkey; Type: FK CONSTRAINT; Schema: suppliers; Owner: -
--

ALTER TABLE ONLY suppliers.supplier_contract
    ADD CONSTRAINT supplier_contract_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES suppliers.supplier_profile(id) ON DELETE CASCADE;


--
-- Name: supplier_profile supplier_profile_user_id_fkey; Type: FK CONSTRAINT; Schema: suppliers; Owner: -
--

ALTER TABLE ONLY suppliers.supplier_profile
    ADD CONSTRAINT supplier_profile_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity."user"(id) ON DELETE SET NULL;


--
-- PostgreSQL database dump complete
--

\unrestrict qbAC2UrhMHE1zaE9MlIjM4PChl1wePPcdfTXnu2vElWFvdmB0WVQmT2ZVEiYeKK

