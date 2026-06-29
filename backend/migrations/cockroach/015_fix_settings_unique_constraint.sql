-- CockroachDB variant of 015. Provide the unique constraint that ON CONFLICT ("key")
-- requires via an idempotent unique index instead of a DO-block guarded ADD CONSTRAINT.
CREATE UNIQUE INDEX IF NOT EXISTS settings_key_key ON settings ("key");
