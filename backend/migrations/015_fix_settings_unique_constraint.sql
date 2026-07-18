-- 015_fix_settings_unique_constraint.sql
-- 修复 settings 表 key 字段缺失的唯一约束
-- 此约束是 ON CONFLICT ("key") DO UPDATE 语句所必需的

-- CockroachDB compatibility:
-- settings.key is already UNIQUE in 005_schema_parity.sql on fresh installs.
-- Keep an idempotent unique index for older schemas without using pg_constraint
-- or ::regclass inside a DO block.
CREATE UNIQUE INDEX IF NOT EXISTS settings_key_key ON settings (key);
