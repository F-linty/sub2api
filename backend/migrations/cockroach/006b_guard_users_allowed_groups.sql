-- CockroachDB variant of 006b. Fresh installs have not run 014 (drop legacy) yet,
-- so ensuring users.allowed_groups exists matches the PostgreSQL guard's net effect.
ALTER TABLE users ADD COLUMN IF NOT EXISTS allowed_groups BIGINT[] DEFAULT NULL;
