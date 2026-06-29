-- CockroachDB variant of 128. Flatten the DO-block CHECK/FK guards to plain ADD
-- CONSTRAINT (CRDB rejects ALTER TABLE inside DO blocks). Per-statement autocommit
-- lets each ADD COLUMN commit before the constraints/index reference it.
CREATE TABLE IF NOT EXISTS channel_monitor_request_templates (
    id            BIGSERIAL    PRIMARY KEY,
    name          VARCHAR(100) NOT NULL,
    provider      VARCHAR(20)  NOT NULL,
    description   VARCHAR(500) NOT NULL DEFAULT '',
    extra_headers JSONB        NOT NULL DEFAULT '{}'::jsonb,
    body_override_mode VARCHAR(10) NOT NULL DEFAULT 'off',
    body_override JSONB        NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT channel_monitor_request_templates_provider_check
        CHECK (provider IN ('openai', 'anthropic', 'gemini')),
    CONSTRAINT channel_monitor_request_templates_body_mode_check
        CHECK (body_override_mode IN ('off', 'merge', 'replace'))
);

CREATE UNIQUE INDEX IF NOT EXISTS channel_monitor_request_templates_provider_name
    ON channel_monitor_request_templates (provider, name);

ALTER TABLE channel_monitors ADD COLUMN IF NOT EXISTS template_id        BIGINT      NULL;
ALTER TABLE channel_monitors ADD COLUMN IF NOT EXISTS extra_headers      JSONB       NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE channel_monitors ADD COLUMN IF NOT EXISTS body_override_mode VARCHAR(10) NOT NULL DEFAULT 'off';
ALTER TABLE channel_monitors ADD COLUMN IF NOT EXISTS body_override      JSONB       NULL;

ALTER TABLE channel_monitors
    ADD CONSTRAINT channel_monitors_body_mode_check
    CHECK (body_override_mode IN ('off', 'merge', 'replace'));

ALTER TABLE channel_monitors
    ADD CONSTRAINT channel_monitors_template_id_fkey
    FOREIGN KEY (template_id)
    REFERENCES channel_monitor_request_templates (id)
    ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_channel_monitors_template_id
    ON channel_monitors (template_id)
    WHERE template_id IS NOT NULL;
