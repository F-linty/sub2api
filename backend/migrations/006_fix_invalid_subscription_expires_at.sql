-- Fix legacy subscription records with invalid expires_at (year > 2099).
-- CockroachDB compatibility: user_subscriptions is created by 003_subscription.sql,
-- so this can be a plain UPDATE instead of a DO/to_regclass block.
UPDATE user_subscriptions
SET expires_at = TIMESTAMPTZ '2099-12-31 23:59:59+00'
WHERE expires_at > TIMESTAMPTZ '2099-12-31 23:59:59+00';
