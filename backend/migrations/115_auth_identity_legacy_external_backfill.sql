-- CockroachDB compatibility:
-- Legacy user_external_identities backfill uses PL/pgSQL helper functions.
-- Fresh installs do not have the legacy table, so this is a no-op.
SELECT 1;
