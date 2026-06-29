-- CockroachDB variant of 112. Same SQL as PostgreSQL, but run per-statement in
-- autocommit so the ADD COLUMN backfill commits before the UPDATE that reads it
-- (avoids CRDB "column is being backfilled").
ALTER TABLE payment_orders ADD COLUMN IF NOT EXISTS provider_key VARCHAR(30);

UPDATE payment_orders
SET provider_key = (
    SELECT provider_key
    FROM payment_provider_instances
    WHERE CAST(id AS TEXT) = payment_orders.provider_instance_id
)
WHERE provider_key IS NULL
  AND provider_instance_id IS NOT NULL;
