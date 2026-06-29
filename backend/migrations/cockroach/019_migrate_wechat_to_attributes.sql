-- CockroachDB variant of 019.
--
-- The PostgreSQL original is a +goose file, but this project's runner is not goose-aware,
-- so it executes BOTH the Up and Down sections as one batch. On a fresh install (no users
-- with wechat data) those sections cancel out, with a net effect of: the "wechat" attribute
-- definition is created and then soft-deleted, and users.wechat (added by 005) is dropped
-- then re-added (i.e. still present). CRDB cannot drop-then-reference the column in a single
-- batch, so we reproduce that net end-state directly here.
--
-- Net effect reproduced: create the wechat attribute definition, then soft-delete it.
-- users.wechat is left in place (matches the PG re-add).
INSERT INTO user_attribute_definitions (key, name, description, type, options, required, validation, placeholder, display_order, enabled, created_at, updated_at)
SELECT 'wechat', '微信', '用户微信号', 'text', '[]'::jsonb, false, '{}'::jsonb, '请输入微信号', 0, true, NOW(), NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM user_attribute_definitions WHERE key = 'wechat' AND deleted_at IS NULL
);

UPDATE user_attribute_definitions
SET deleted_at = NOW()
WHERE key = 'wechat' AND deleted_at IS NULL;
