# M5 Analytics — Nightly Rollup Jobs, Read Replica & Analytics Tool Connection

**Status:** Specification
**Owner:** Engineering
**Milestone:** M5

## Purpose

Define the technical specification for the M5 Analytics deliverable: nightly rollup jobs that aggregate operational data into analytical tables, a PostgreSQL read replica for analytical query isolation, and a read-only connection for an external analytics tool (e.g., Metabase, Apache Superset, Evidence).

This is infrastructure/data engineering work — not a traditional REST API module. No new HTTP endpoints are added to the API contract.

---

## 1. Nightly Rollup Jobs

### 1.1 Design Principles

- **Idempotent**: Re-running a job for the same date produces identical results
- **Backfillable**: Historical data can be recomputed by running jobs for past dates
- **Incremental where possible**: Only process new/changed data since last successful run
- **Failure isolation**: One rollup failure does not block others
- **Observable**: Structured logging, metrics, and alerting on success/failure

### 1.2 Source Tables (Operational Schema)

Rollups read from the primary database (write replica). Key source tables:

| Domain | Source Tables | Notes |
|--------|--------------|-------|
| Orders | `commerce.order`, `commerce.order_line`, `commerce.order_history` | Core transactional data |
| Payments | `payments.payment_intent`, `payments.payment_refund`, `payments.payment_method` | Payment lifecycle |
| Inventory | `inventory.stock_level`, `inventory.lot`, `inventory.reservation` | Stock movements |
| Catalogue | `catalog.product`, `catalog.category`, `catalog.brand`, `catalog.supplier` | Dimension data |
| Buyers | `identity.buyer`, `identity.buyer_address`, `identity.verification` | Customer dimensions |
| Promotions | `promotions.promotion`, `promotions.voucher`, `promotions.redemption` | Discount attribution |
| Suppliers | `suppliers.supplier`, `suppliers.contract` | Supplier performance |
| Financial | `finance.invoice`, `finance.credit_note`, `finance.credit_account` | AR/AP |
| ERP | `erp.sync_job`, `erp.sync_drift`, `erp.order_dispatch` | Integration health |

### 1.3 Rollup Tables (Analytical Schema)

All rollup tables live in a dedicated `analytics` schema on the **read replica** (populated by jobs running on the primary, then replicated).

#### 1.3.1 Daily Order Rollup — `analytics.daily_order_rollup`

```sql
CREATE TABLE analytics.daily_order_rollup (
    rollup_date        DATE NOT NULL,
    buyer_code         VARCHAR(32) NOT NULL,
    supplier_code      VARCHAR(32) NOT NULL,
    order_count        BIGINT NOT NULL DEFAULT 0,
    line_count         BIGINT NOT NULL DEFAULT 0,
    total_quantity     NUMERIC(18,4) NOT NULL DEFAULT 0,
    revenue_minor      BIGINT NOT NULL DEFAULT 0,
    currency           CHAR(3) NOT NULL,
    status             VARCHAR(32) NOT NULL, -- placed/confirmed/processing/shipped/delivered/cancelled/on_hold
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (rollup_date, buyer_code, supplier_code, currency, status)
);

CREATE INDEX idx_daily_order_rollup_date ON analytics.daily_order_rollup (rollup_date DESC);
CREATE INDEX idx_daily_order_rollup_buyer ON analytics.daily_order_rollup (buyer_code, rollup_date DESC);
CREATE INDEX idx_daily_order_rollup_supplier ON analytics.daily_order_rollup (supplier_code, rollup_date DESC);
```

**Granularity**: One row per (date, buyer, supplier, currency, status)

**Population**: Nightly job aggregates `commerce.order` + `commerce.order_line` for orders placed on `rollup_date`.

#### 1.3.2 Daily Payment Rollup — `analytics.daily_payment_rollup`

```sql
CREATE TABLE analytics.daily_payment_rollup (
    rollup_date        DATE NOT NULL,
    buyer_code         VARCHAR(32) NOT NULL,
    payment_method     VARCHAR(64) NOT NULL,
    payment_status     VARCHAR(32) NOT NULL, -- succeeded/failed/refunded/pending
    transaction_count  BIGINT NOT NULL DEFAULT 0,
    amount_minor       BIGINT NOT NULL DEFAULT 0,
    currency           CHAR(3) NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (rollup_date, buyer_code, payment_method, payment_status, currency)
);
```

#### 1.3.3 Daily Inventory Rollup — `analytics.daily_inventory_rollup`

```sql
CREATE TABLE analytics.daily_inventory_rollup (
    rollup_date        DATE NOT NULL,
    product_code       VARCHAR(64) NOT NULL,
    supplier_code      VARCHAR(32) NOT NULL,
    location_code      VARCHAR(32),
    stock_on_hand      NUMERIC(18,4) NOT NULL DEFAULT 0,
    stock_reserved     NUMERIC(18,4) NOT NULL DEFAULT 0,
    stock_available    NUMERIC(18,4) NOT NULL DEFAULT 0,
    lots_count         INT NOT NULL DEFAULT 0,
    expiring_soon_count INT NOT NULL DEFAULT 0, -- lots expiring within 30 days
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (rollup_date, product_code, supplier_code, location_code)
);
```

#### 1.3.4 Daily Promotion Rollup — `analytics.daily_promotion_rollup`

```sql
CREATE TABLE analytics.daily_promotion_rollup (
    rollup_date        DATE NOT NULL,
    promotion_code     VARCHAR(64) NOT NULL,
    voucher_code       VARCHAR(64),
    redemptions        BIGINT NOT NULL DEFAULT 0,
    discount_minor     BIGINT NOT NULL DEFAULT 0,
    currency           CHAR(3) NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (rollup_date, promotion_code, voucher_code, currency)
);
```

#### 1.3.5 Daily Financial Rollup — `analytics.daily_financial_rollup`

```sql
CREATE TABLE analytics.daily_financial_rollup (
    rollup_date        DATE NOT NULL,
    buyer_code         VARCHAR(32) NOT NULL,
    currency           CHAR(3) NOT NULL,
    invoiced_minor     BIGINT NOT NULL DEFAULT 0,
    collected_minor    BIGINT NOT NULL DEFAULT 0,
    credited_minor     BIGINT NOT NULL DEFAULT 0,
    refunded_minor     BIGINT NOT NULL DEFAULT 0,
    outstanding_minor  BIGINT NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (rollup_date, buyer_code, currency)
);
```

#### 1.3.6 Daily ERP Sync Rollup — `analytics.daily_erp_sync_rollup`

```sql
CREATE TABLE analytics.daily_erp_sync_rollup (
    rollup_date        DATE NOT NULL,
    sync_type          VARCHAR(32) NOT NULL, -- catalog/stock
    job_status         VARCHAR(32) NOT NULL, -- completed/failed
    total_records      BIGINT NOT NULL DEFAULT 0,
    matched_records    BIGINT NOT NULL DEFAULT 0,
    created_records    BIGINT NOT NULL DEFAULT 0,
    updated_records    BIGINT NOT NULL DEFAULT 0,
    drift_count        BIGINT NOT NULL DEFAULT 0,
    duration_ms        BIGINT NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (rollup_date, sync_type, job_status)
);
```

### 1.4 Job Execution Logic

Each rollup job follows this pattern:

```go
// Pseudocode for each rollup job
func RunDailyRollup(ctx context.Context, date time.Time) error {
    // 1. Determine date window: [date 00:00:00 UTC, date+1 00:00:00 UTC)
    start := date.Truncate(24*time.Hour)
    end := start.Add(24 * time.Hour)

    // 2. Delete existing data for this date (idempotency)
    _, err := db.Exec(ctx, `DELETE FROM analytics.daily_X_rollup WHERE rollup_date = $1`, date)
    if err != nil { return err }

    // 3. Insert aggregated data from operational tables
    _, err = db.Exec(ctx, rollupInsertSQL, start, end)
    return err
}
```

**Scheduling**: Each job runs at 02:00 UTC (after midnight UTC, allowing late-arriving data).

**Backfill**: A CLI command or admin endpoint triggers rollup for a specific date range.

### 1.5 Job Orchestration

Use **pg_cron** extension (runs inside PostgreSQL) for scheduling:

```sql
-- Enable pg_cron
CREATE EXTENSION IF NOT EXISTS pg_cron;

-- Schedule each rollup at 02:00 UTC
SELECT cron.schedule('daily-order-rollup', '0 2 * * *', $$
    SELECT analytics.run_daily_order_rollup(CURRENT_DATE - 1);
$$);

SELECT cron.schedule('daily-payment-rollup', '0 2 * * *', $$
    SELECT analytics.run_daily_payment_rollup(CURRENT_DATE - 1);
$$);
-- ... etc for each rollup
```

**Alternative**: External cron (systemd timer, Kubernetes CronJob) calling a Go binary if pg_cron is not available.

### 1.6 Rollup Functions (PL/pgSQL)

Each rollup is implemented as a PL/pgSQL function for performance (runs close to data):

```sql
CREATE OR REPLACE FUNCTION analytics.run_daily_order_rollup(p_date DATE)
RETURNS VOID LANGUAGE plpgsql AS $$
BEGIN
    DELETE FROM analytics.daily_order_rollup WHERE rollup_date = p_date;

    INSERT INTO analytics.daily_order_rollup (rollup_date, buyer_code, supplier_code, order_count, line_count, total_quantity, revenue_minor, currency, status)
    SELECT
        p_date,
        o.buyer_code,
        ol.supplier_code,
        COUNT(DISTINCT o.id) AS order_count,
        COUNT(ol.id) AS line_count,
        SUM(ol.quantity) AS total_quantity,
        SUM(ol.total_minor) AS revenue_minor,
        ol.currency,
        o.status
    FROM commerce.order o
    JOIN commerce.order_line ol ON ol.order_id = o.id
    WHERE o.placed_at >= p_date AND o.placed_at < p_date + INTERVAL '1 day'
    GROUP BY o.buyer_code, ol.supplier_code, ol.currency, o.status;
END;
$$;
```

---

## 2. Read Replica Setup

### 2.1 Architecture

```
┌─────────────────┐     Streaming Replication     ┌─────────────────┐
│  Primary DB     │ ─────────────────────────────▶ │  Read Replica   │
│  (Read/Write)   │                                │  (Read-Only)    │
│  - Operational  │                                │  - Analytics    │
│    tables       │                                │    schema       │
│  - pg_cron jobs │                                │  - Analytics    │
│                 │                                │    tool conn    │
└─────────────────┘                                └─────────────────┘
```

### 2.2 Configuration

**Primary (postgresql.conf)**:
```conf
wal_level = replica
max_wal_senders = 10
wal_keep_size = 1GB
hot_standby = on
```

**Replica (postgresql.conf)**:
```conf
hot_standby = on
hot_standby_feedback = on
max_standby_streaming_delay = 30s
wal_receiver_status_interval = 10s
```

### 2.3 Connection Pooling

Use **PgBouncer** in transaction pooling mode on the replica:

```ini
[databases]
analytics = host=replica-host port=5432 dbname=atlas user=analytics_ro

[pgbouncer]
pool_mode = transaction
default_pool_size = 20
max_client_conn = 100
```

### 2.4 Failover Considerations

- Replica promotion is manual (not automatic) — analytics downtime is acceptable
- Monitor replication lag: `SELECT pg_last_wal_replay_lsn() = pg_last_wal_receive_lsn();`
- Alert if lag > 60 seconds

---

## 3. Analytics Tool Connection

### 3.1 Database Role

```sql
-- On the REPLICA
CREATE ROLE analytics_ro NOINHERIT LOGIN PASSWORD '...';
GRANT USAGE ON SCHEMA analytics TO analytics_ro;
GRANT SELECT ON ALL TABLES IN SCHEMA analytics TO analytics_ro;
ALTER DEFAULT PRIVILEGES IN SCHEMA analytics GRANT SELECT ON TABLES TO analytics_ro;

-- Limit resource usage
ALTER ROLE analytics_ro SET statement_timeout = '300s'; -- 5 min query timeout
ALTER ROLE analytics_ro SET idle_in_transaction_session_timeout = '60s';
```

### 3.2 Network Access

- Replica exposed on internal network only (VPC/subnet)
- Analytics tool connects via private IP or VPC peering
- No public internet access

### 3.3 Connection Parameters

| Parameter | Value |
|-----------|-------|
| Host | replica internal hostname |
| Port | 5432 (or PgBouncer port 6432) |
| Database | atlas |
| User | analytics_ro |
| Password | From secret manager / env var |
| SSL Mode | require |
| Pool Mode | transaction (via PgBouncer) |

### 3.4 Query Governance

- All queries run with `analytics_ro` role (enforces timeouts)
- No DDL/DML possible
- Long-running queries auto-cancelled at 5 minutes
- Idle transactions killed at 60 seconds

---

## 4. Monitoring & Alerting

### 4.1 Rollup Job Metrics

| Metric | Type | Alert Threshold |
|--------|------|-----------------|
| `analytics_rollup_duration_seconds` | Histogram | > 300s (5 min) |
| `analytics_rollup_rows_processed` | Counter | N/A |
| `analytics_rollup_success_total` | Counter | N/A |
| `analytics_rollup_failure_total` | Counter | > 0 (immediate) |

### 4.2 Replication Metrics

| Metric | Type | Alert Threshold |
|--------|------|-----------------|
| `pg_replication_lag_seconds` | Gauge | > 60s |
| `pg_replication_active` | Gauge (0/1) | = 0 (immediate) |

### 4.3 Analytics Tool Health

- Periodic `SELECT 1` from analytics tool connection
- Alert on connection failures

---

## 5. Implementation Plan

### 5.1 Phase 1: Database Objects (Migration)
- Create `analytics` schema
- Create all rollup tables (6 tables)
- Create PL/pgSQL rollup functions (6 functions)
- Create `analytics_ro` role with grants

### 5.2 Phase 2: Job Scheduling
- Enable pg_cron extension
- Schedule 6 nightly jobs at 02:00 UTC
- Add manual backfill function: `analytics.backfill_rollups(start_date, end_date)`

### 5.3 Phase 3: Read Replica
- Provision replica (infrastructure task)
- Configure streaming replication
- Set up PgBouncer on replica
- Verify replication lag < 5s under load

### 5.4 Phase 4: Analytics Tool Onboarding
- Create `analytics_ro` credentials in secret manager
- Document connection parameters for client
- Client configures their tool (Metabase/Superset/Evidence)
- Validate sample queries work

---

## 6. Rollback / Disaster Recovery

- Rollup tables can be dropped and recreated (re-computed from operational data)
- Read replica can be rebuilt from primary base backup + WAL
- Analytics tool connection is read-only — no data loss risk

---

## 7. Open Questions

1. **pg_cron availability**: Confirm target PostgreSQL version supports pg_cron (15+). If not, use external scheduler.
2. **Retention policy**: How long to keep rollup data? (Propose: 3 years rolling)
3. **Analytics tool**: Client to confirm tool choice (affects connection testing)
4. **Timezone**: All rollups use UTC. Confirm with product.
5. **Backfill strategy**: Initial historical backfill (how many months?) — run as one-off after deployment.

---

## 8. Acceptance Criteria

- [ ] 6 rollup tables created in `analytics` schema on replica
- [ ] 6 PL/pgSQL functions created and tested
- [ ] pg_cron jobs scheduled and running nightly at 02:00 UTC
- [ ] Read replica provisioned, replication lag < 5s steady state
- [ ] PgBouncer configured on replica with transaction pooling
- [ ] `analytics_ro` role created with SELECT grants and timeouts
- [ ] Analytics tool successfully connects and runs sample queries
- [ ] Monitoring dashboards show rollup success/failure and replication lag
- [ ] Backfill function tested for 30-day historical range
- [ ] Documentation updated with connection details and rollup table dictionary

---

## 9. Related Documents

- [05 API Contract](05-api-contract.md) — no new endpoints
- [08 Feature Inventory](08-feature-inventory.md) — A10.12 Health dashboard, A11 Reports
- [10 Delivery Plan](10-delivery-plan.md) — M5 Analytics deliverable
- [06 Data and Events](06-data-and-events.md) — event schema for potential future streaming