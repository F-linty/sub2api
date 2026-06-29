-- CockroachDB variant of 006. CRDB rejects ALTER TABLE inside DO/PL-pgSQL blocks;
-- the original is just an idempotency-guarded ADD COLUMN, which flattens to native DDL.
ALTER TABLE users ADD COLUMN IF NOT EXISTS allowed_groups BIGINT[] DEFAULT NULL;
