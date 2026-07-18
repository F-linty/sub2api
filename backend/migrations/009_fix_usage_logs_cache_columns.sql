-- Ensure usage_logs cache token columns use the underscored names expected by code.
-- Backfill from legacy column names if they exist.

ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS cache_creation_5m_tokens INT NOT NULL DEFAULT 0;

ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS cache_creation_1h_tokens INT NOT NULL DEFAULT 0;

-- CockroachDB compatibility:
-- The legacy non-underscored columns are not present on a fresh Sub2API schema.
-- Conditional backfill from optional columns would require PostgreSQL DO blocks,
-- so it is intentionally skipped for CockroachDB fresh installs.
