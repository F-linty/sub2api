-- CockroachDB variant of 061. Flatten the DO-block CHECK-constraint guard to native DDL
-- and drop the historical batched backfill (usage_logs is empty on a fresh install).
-- Per-statement autocommit lets the ADD COLUMN backfill commit before the CHECK/INDEX.
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS request_type SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE usage_logs DROP CONSTRAINT IF EXISTS usage_logs_request_type_check;
ALTER TABLE usage_logs ADD CONSTRAINT usage_logs_request_type_check CHECK (request_type IN (0, 1, 2, 3));
CREATE INDEX IF NOT EXISTS idx_usage_logs_request_type_created_at ON usage_logs (request_type, created_at);
