-- CockroachDB does not support PostgreSQL PL/pgSQL triggers. Runtime writes
-- normalize this setting in application code; this migration only backfills
-- existing rows and emits scheduler events for changed spark shadows.
UPDATE accounts
SET extra = jsonb_set(
    COALESCE(extra, '{}'::jsonb),
    '{openai_long_context_billing_enabled}',
    'false'::jsonb,
    true
)
WHERE platform = 'openai'
  AND COALESCE(extra, '{}'::jsonb) ? 'openai_long_context_billing_enabled'
  AND jsonb_typeof(extra->'openai_long_context_billing_enabled') IS DISTINCT FROM 'boolean';

UPDATE accounts
SET extra = jsonb_set(
    COALESCE(extra, '{}'::jsonb),
    '{openai_long_context_billing_enabled}',
    'false'::jsonb,
    true
)
WHERE platform = 'openai'
  AND parent_account_id IS NULL
  AND NOT (COALESCE(extra, '{}'::jsonb) ? 'openai_long_context_billing_enabled');

WITH shadow_values AS (
    SELECT
        shadow.id,
        CASE
            WHEN parent.platform IS DISTINCT FROM 'openai' THEN 'false'::jsonb
            WHEN NOT (COALESCE(parent.extra, '{}'::jsonb) ? 'openai_long_context_billing_enabled') THEN 'false'::jsonb
            WHEN jsonb_typeof(parent.extra->'openai_long_context_billing_enabled') = 'boolean'
                THEN parent.extra->'openai_long_context_billing_enabled'
            ELSE 'false'::jsonb
        END AS effective_value
    FROM accounts AS shadow
    JOIN accounts AS parent ON parent.id = shadow.parent_account_id
    WHERE shadow.platform = 'openai'
      AND shadow.quota_dimension = 'spark'
),
updated_shadows AS (
    UPDATE accounts AS shadow
    SET extra = jsonb_set(
        COALESCE(shadow.extra, '{}'::jsonb),
        '{openai_long_context_billing_enabled}',
        shadow_values.effective_value,
        true
    )
    FROM shadow_values
    WHERE shadow.id = shadow_values.id
      AND shadow.extra->'openai_long_context_billing_enabled'
          IS DISTINCT FROM shadow_values.effective_value
    RETURNING shadow.id
)
INSERT INTO scheduler_outbox (event_type, account_id)
SELECT 'account_changed', id
FROM updated_shadows;
