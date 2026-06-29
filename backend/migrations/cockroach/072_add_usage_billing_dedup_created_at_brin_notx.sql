-- CockroachDB variant of 072. CRDB has no BRIN access method, so a default index on
-- created_at backs retention-window cleanup. CONCURRENTLY is supported on CRDB v26+
-- and runs in autocommit. Index name kept identical for parity.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_billing_dedup_created_at_brin
    ON usage_billing_dedup (created_at);
