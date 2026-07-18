-- CockroachDB compatibility:
-- Avoid DO blocks with schema changes. This guard is only for old schemas;
-- the plain idempotent ALTER is safe before 014_drop_legacy_allowed_groups.sql.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS allowed_groups BIGINT[] DEFAULT NULL;
