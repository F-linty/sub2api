-- CockroachDB variant of 138. Flatten the DO-block CHECK guards to plain ADD CONSTRAINT;
-- per-statement autocommit lets each ADD COLUMN commit before its CHECK/index.
ALTER TABLE channel_monitors
    ADD COLUMN IF NOT EXISTS api_mode VARCHAR(32) NOT NULL DEFAULT 'chat_completions';

ALTER TABLE channel_monitor_request_templates
    ADD COLUMN IF NOT EXISTS api_mode VARCHAR(32) NOT NULL DEFAULT 'chat_completions';

ALTER TABLE channel_monitors DROP CONSTRAINT IF EXISTS channel_monitors_api_mode_check;
ALTER TABLE channel_monitors
    ADD CONSTRAINT channel_monitors_api_mode_check
    CHECK (api_mode IN ('chat_completions', 'responses'));

ALTER TABLE channel_monitor_request_templates DROP CONSTRAINT IF EXISTS channel_monitor_request_templates_api_mode_check;
ALTER TABLE channel_monitor_request_templates
    ADD CONSTRAINT channel_monitor_request_templates_api_mode_check
    CHECK (api_mode IN ('chat_completions', 'responses'));

CREATE INDEX IF NOT EXISTS idx_channel_monitors_provider_api_mode
    ON channel_monitors (provider, api_mode);

CREATE INDEX IF NOT EXISTS idx_channel_monitor_templates_provider_api_mode
    ON channel_monitor_request_templates (provider, api_mode);
