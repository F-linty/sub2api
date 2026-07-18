-- CockroachDB compatibility:
-- BRIN indexes are PostgreSQL-specific. Use a normal secondary index.

CREATE INDEX IF NOT EXISTS idx_usage_billing_dedup_created_at_brin
    ON usage_billing_dedup (created_at);
