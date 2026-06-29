-- CockroachDB variant of 101. Same SQL as PostgreSQL, run per-statement in autocommit
-- so the ADD COLUMN commits before the UPDATEs that read payment_mode (avoids CRDB's
-- single-transaction add-then-reference error). UPDATEs are no-ops on a fresh install.
ALTER TABLE payment_provider_instances ADD COLUMN IF NOT EXISTS payment_mode VARCHAR(20) NOT NULL DEFAULT '';

UPDATE payment_provider_instances
SET payment_mode = 'redirect',
    supported_types = TRIM(BOTH ',' FROM REPLACE(REPLACE(REPLACE(
      supported_types, 'easypay,', ''), ',easypay', ''), 'easypay', ''))
WHERE provider_key = 'easypay' AND supported_types LIKE '%easypay%';

UPDATE payment_provider_instances
SET payment_mode = 'api'
WHERE provider_key = 'easypay' AND payment_mode = '';
