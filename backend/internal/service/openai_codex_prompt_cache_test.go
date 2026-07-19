package service

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResolveCodexOAuthPromptCacheKeyUsesExistingBodyKey(t *testing.T) {
	reqBody := map[string]any{"prompt_cache_key": " client-key "}

	got := resolveCodexOAuthPromptCacheKey(nil, &Account{ID: 10}, reqBody, "gpt-5.3-codex")

	require.Equal(t, "client-key", got.Key)
	require.Equal(t, "body", got.Source)
}

func TestResolveCodexOAuthPromptCacheKeyDerivesFromHeaderSessionID(t *testing.T) {
	c := codexPromptCacheTestContext(map[string]string{"session_id": "sess-1"})
	account := &Account{ID: 10}
	reqBody := map[string]any{"model": "gpt-5.3-codex", "input": []any{"hello"}}

	got1 := resolveCodexOAuthPromptCacheKey(c, account, reqBody, "gpt-5.3-codex")
	got2 := resolveCodexOAuthPromptCacheKey(c, account, reqBody, "gpt-5.3-codex")

	require.NotEmpty(t, got1.Key)
	require.Equal(t, got1.Key, got2.Key)
	require.Equal(t, "header_session_id", got1.Source)
	require.Contains(t, got1.Key, codexAutoPromptCacheKeyPrefix)
}

func TestResolveCodexOAuthPromptCacheKeyBodySeedIsStableAndIsolated(t *testing.T) {
	reqBody := map[string]any{
		"model":        "gpt-5.3-codex",
		"instructions": "be useful",
		"tools": []any{
			map[string]any{"type": "function", "name": "a"},
		},
		"input": []any{
			map[string]any{"type": "message", "role": "developer", "content": "system"},
			map[string]any{"type": "message", "role": "user", "content": "first user"},
			map[string]any{"type": "message", "role": "user", "content": "later user"},
		},
	}
	account := &Account{ID: 10}

	got1 := resolveCodexOAuthPromptCacheKey(nil, account, reqBody, "gpt-5.3-codex")
	got2 := resolveCodexOAuthPromptCacheKey(nil, account, reqBody, "gpt-5.3-codex")
	otherAccount := resolveCodexOAuthPromptCacheKey(nil, &Account{ID: 11}, reqBody, "gpt-5.3-codex")

	require.NotEmpty(t, got1.Key)
	require.Equal(t, "derived_body", got1.Source)
	require.Equal(t, got1.Key, got2.Key)
	require.NotEqual(t, got1.Key, otherAccount.Key)
}

func codexPromptCacheTestContext(headers map[string]string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	req, _ := http.NewRequest(http.MethodPost, "/v1/responses", nil)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	c.Request = req
	return c
}
