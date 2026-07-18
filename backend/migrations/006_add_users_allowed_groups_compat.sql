-- CockroachDB compatibility:
-- CockroachDB does not support ALTER TABLE inside DO/function bodies.
-- On a fresh schema, users exists by 001_init.sql, so keep this as a plain
-- idempotent ALTER. It is a no-op when the column already exists.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS allowed_groups BIGINT[] DEFAULT NULL;
