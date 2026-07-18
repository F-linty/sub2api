-- CockroachDB compatibility:
-- pg_trgm, gin_trgm_ops, and PostgreSQL GIN trigram indexes are unavailable.
-- Fuzzy search remains functional without these acceleration indexes.
SELECT 1;
