-- CockroachDB variant of 052. Same SQL as PostgreSQL, but the runner executes these
-- per-statement in autocommit so the ADD COLUMN backfill commits before the UPDATE and
-- CREATE INDEX, avoiding CRDB's "column is being backfilled" async-schema-change error.
ALTER TABLE groups ADD COLUMN IF NOT EXISTS sort_order INT NOT NULL DEFAULT 0;
UPDATE groups SET sort_order = id WHERE sort_order = 0;
CREATE INDEX IF NOT EXISTS idx_groups_sort_order ON groups(sort_order);
