-- CockroachDB variant of 134. Same SQL as PostgreSQL, run per-statement in autocommit so
-- each ADD COLUMN commits before the UPDATEs that read the new columns (avoids CRDB's
-- single-transaction add-then-reference error).
ALTER TABLE groups ADD COLUMN IF NOT EXISTS allow_image_generation BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE groups ADD COLUMN IF NOT EXISTS image_rate_independent BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE groups ADD COLUMN IF NOT EXISTS image_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0;

UPDATE groups
SET allow_image_generation = true
WHERE platform IN ('openai', 'gemini', 'antigravity');

UPDATE groups
SET image_rate_independent = false,
    image_rate_multiplier = 1.0;

COMMENT ON COLUMN groups.allow_image_generation IS '是否允许该分组使用图片生成能力';
COMMENT ON COLUMN groups.image_rate_independent IS '图片生成是否使用独立倍率；false 表示共享分组有效倍率';
COMMENT ON COLUMN groups.image_rate_multiplier IS '图片生成独立倍率，仅 image_rate_independent=true 时生效';
