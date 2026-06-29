-- CockroachDB variant of 009. The PostgreSQL original backfills the underscored cache
-- columns from legacy non-underscored names inside a DO block. On a fresh install those
-- legacy columns never existed, so only the ADD COLUMNs are needed.
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS cache_creation_5m_tokens INT NOT NULL DEFAULT 0;
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS cache_creation_1h_tokens INT NOT NULL DEFAULT 0;
