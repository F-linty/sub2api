-- CockroachDB compatibility:
-- This historical settings migration uses PL/pgSQL and jsonb array iteration.
-- Fresh installs do not have the legacy purchase_subscription_* settings, so
-- keep it as a no-op for CockroachDB.
SELECT 1;
