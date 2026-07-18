-- Add request_type enum for usage_logs while keeping legacy stream/openai_ws_mode compatibility.
ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS request_type SMALLINT NOT NULL DEFAULT 0;

ALTER TABLE usage_logs
    ADD CONSTRAINT usage_logs_request_type_check
    CHECK (request_type IN (0, 1, 2, 3));

CREATE INDEX IF NOT EXISTS idx_usage_logs_request_type_created_at
    ON usage_logs (request_type, created_at);

-- CockroachDB compatibility:
-- Use a single set-based backfill instead of a PL/pgSQL bounded loop.
-- Fresh installs have no historical rows, so this is normally a no-op.
UPDATE usage_logs
SET request_type = CASE
    WHEN openai_ws_mode = TRUE THEN 3
    WHEN stream = TRUE THEN 2
    ELSE 1
END
WHERE request_type = 0;
