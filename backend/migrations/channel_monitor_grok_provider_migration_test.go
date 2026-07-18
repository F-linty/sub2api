package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannelMonitorGrokProviderMigration(t *testing.T) {
	content, err := FS.ReadFile("176_channel_monitor_grok_provider.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "channel_monitors_provider_check")
	require.Contains(t, sql, "channel_monitor_request_templates_provider_check")
	require.Contains(t, sql, "CHECK (provider IN ('openai', 'anthropic', 'gemini', 'grok'))")
	require.Contains(t, sql, "DROP CONSTRAINT IF EXISTS")
	require.NotContains(t, sql, "DO $$")
	require.NotContains(t, sql, "pg_get_constraintdef")
}
