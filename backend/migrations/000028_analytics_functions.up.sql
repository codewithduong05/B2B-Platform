-- M5 Analytics: PL/pgSQL rollup functions

-- Helper function to get date window for a given rollup date
CREATE OR REPLACE FUNCTION analytics.date_window(p_date DATE)
RETURNS TABLE (window_start TIMESTAMPTZ, window_end TIMESTAMPTZ) LANGUAGE sql STABLE AS $$
    SELECT p_date::TIMESTAMPTZ AT TIME ZONE 'UTC', (p_date + INTERVAL '1 day')::TIMESTAMPTZ AT TIME ZONE 'UTC';
$$;

-- 1. Daily Order Rollup Function
CREATE OR REPLACE FUNCTION analytics.run_daily_order_rollup(p_date DATE)
RETURNS VOID LANGUAGE plpgsql AS $$
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

-- 2. Daily Payment Rollup Function
CREATE OR REPLACE FUNCTION analytics.run_daily_payment_rollup(p_date DATE)
RETURNS VOID LANGUAGE plpgsql AS $$
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

-- 3. Daily Inventory Rollup Function
CREATE OR REPLACE FUNCTION analytics.run_daily_inventory_rollup(p_date DATE)
RETURNS VOID LANGUAGE plpgsql AS $$
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

-- 4. Daily Promotion Rollup Function
CREATE OR REPLACE FUNCTION analytics.run_daily_promotion_rollup(p_date DATE)
RETURNS VOID LANGUAGE plpgsql AS $$
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

-- 5. Daily Financial Rollup Function
CREATE OR REPLACE FUNCTION analytics.run_daily_financial_rollup(p_date DATE)
RETURNS VOID LANGUAGE plpgsql AS $$
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

-- 6. Daily ERP Sync Rollup Function
CREATE OR REPLACE FUNCTION analytics.run_daily_erp_sync_rollup(p_date DATE)
RETURNS VOID LANGUAGE plpgsql AS $$
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

-- Backfill function: runs all rollups for a date range
CREATE OR REPLACE FUNCTION analytics.backfill_rollups(p_start_date DATE, p_end_date DATE)
RETURNS VOID LANGUAGE plpgsql AS $$
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

-- Grant execute on functions to analytics_ro
GRANT EXECUTE ON FUNCTION analytics.run_daily_order_rollup(DATE) TO analytics_ro;
GRANT EXECUTE ON FUNCTION analytics.run_daily_payment_rollup(DATE) TO analytics_ro;
GRANT EXECUTE ON FUNCTION analytics.run_daily_inventory_rollup(DATE) TO analytics_ro;
GRANT EXECUTE ON FUNCTION analytics.run_daily_promotion_rollup(DATE) TO analytics_ro;
GRANT EXECUTE ON FUNCTION analytics.run_daily_financial_rollup(DATE) TO analytics_ro;
GRANT EXECUTE ON FUNCTION analytics.run_daily_erp_sync_rollup(DATE) TO analytics_ro;
GRANT EXECUTE ON FUNCTION analytics.backfill_rollups(DATE, DATE) TO analytics_ro;
GRANT EXECUTE ON FUNCTION analytics.date_window(DATE) TO analytics_ro;