-- CockroachDB compatibility:
-- Legacy external identity safety reporting uses PL/pgSQL helper functions.
-- Fresh installs do not have the legacy table, so this is a no-op.
SELECT 1;
