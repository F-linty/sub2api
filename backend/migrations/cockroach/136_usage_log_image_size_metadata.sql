-- CockroachDB variant of 136. Flatten the DO-block CHECK guards to plain ADD CONSTRAINT.
-- The PG original uses NOT VALID to skip validating historical rows; on a fresh install
-- usage_logs is empty, so the constraints are added directly (validation is trivial).
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS image_input_size VARCHAR(32);
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS image_output_size VARCHAR(32);
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS image_size_source VARCHAR(16);
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS image_size_breakdown JSONB;

ALTER TABLE usage_logs
    ADD CONSTRAINT usage_logs_image_size_source_check
    CHECK (
        image_size_source IS NULL
        OR image_size_source IN ('output', 'input', 'default', 'legacy')
    );

ALTER TABLE usage_logs
    ADD CONSTRAINT usage_logs_image_billing_size_check
    CHECK (
        image_count <= 0
        OR (
            image_size IS NOT NULL
            AND image_size IN ('1K', '2K', '4K', 'mixed')
        )
    );
