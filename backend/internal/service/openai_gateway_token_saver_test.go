package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tokensaver"
	"github.com/gin-gonic/gin"
)

func TestMaybeApplyOpenAITokenSaverDisabledByDefault(t *testing.T) {
	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	body := tokenSaverRequestBody()

	got := svc.maybeApplyOpenAITokenSaver(nil, nil, body)
	if string(got) != string(body) {
		t.Fatalf("expected body unchanged when token saver is disabled")
	}
}

func TestMaybeApplyOpenAITokenSaverHeaderBypass(t *testing.T) {
	c := tokenSaverGinContext()
	c.Request.Header.Set(openAITokenSaverHeader, "off")
	svc := &OpenAIGatewayService{cfg: &config.Config{
		Gateway: config.GatewayConfig{
			TokenSaver: config.GatewayTokenSaverConfig{Enabled: true, MinBytes: 64},
		},
	}}
	body := tokenSaverRequestBody()

	got := svc.maybeApplyOpenAITokenSaver(c, nil, body)
	if string(got) != string(body) {
		t.Fatalf("expected body unchanged when bypass header is set")
	}
}

func TestMaybeApplyOpenAITokenSaverCompressesWhenEnabled(t *testing.T) {
	svc := &OpenAIGatewayService{cfg: &config.Config{
		Gateway: config.GatewayConfig{
			TokenSaver: config.GatewayTokenSaverConfig{Enabled: true, MinBytes: 64},
		},
	}}
	body := tokenSaverRequestBody()

	got := svc.maybeApplyOpenAITokenSaver(nil, &Account{ID: 123}, body)
	if string(got) == string(body) {
		t.Fatalf("expected body to be compressed")
	}
	if !strings.Contains(string(got), "sub2api token saver") {
		t.Fatalf("expected compression marker in body: %s", string(got))
	}
}

func TestMaybeApplyOpenAIOutputStyleHintWhenEnabled(t *testing.T) {
	body := []byte(`{"instructions":"existing","input":[{"type":"message","role":"user","content":"hello"}]}`)

	got := maybeApplyOpenAIOutputStyleHint(body, true, "terse")

	if string(got) == string(body) || !strings.Contains(string(got), openAIOutputStyleMarker) || !strings.Contains(string(got), "terse") {
		t.Fatalf("expected output style hint in body: %s", string(got))
	}
	gotAgain := maybeApplyOpenAIOutputStyleHint(got, true, "terse")
	if string(gotAgain) != string(got) {
		t.Fatalf("expected style hint to be idempotent: %s", string(gotAgain))
	}
}

func TestMaybeApplyOpenAIOutputStyleHintDisabled(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":"hello"}]}`)

	got := maybeApplyOpenAIOutputStyleHint(body, false, "terse")

	if string(got) != string(body) {
		t.Fatalf("expected body unchanged when output style is disabled")
	}
}

func TestOpenAITokenSaverHitStats(t *testing.T) {
	stats := openAITokenSaverHitStats([]tokensaver.Hit{{
		Path:   "$.input[0].output",
		Filter: "file_list",
		Before: 1000,
		After:  250,
	}})
	if len(stats) != 1 {
		t.Fatalf("expected one stat, got %#v", stats)
	}
	if stats[0].Path != "$.input[0].output" || stats[0].Filter != "file_list" ||
		stats[0].Before != 1000 || stats[0].After != 250 || stats[0].BytesSaved != 750 {
		t.Fatalf("unexpected stat: %#v", stats[0])
	}
}

func TestOpenAITokenSaverReferenceScopeUsesAccountModelAndHashedCacheKey(t *testing.T) {
	body := []byte(`{"model":" gpt-5.5 ","prompt_cache_key":" raw-secret-session "}`)

	scope := openAITokenSaverReferenceScope(nil, &Account{ID: 123}, body)

	if !strings.Contains(scope, "account=123") || !strings.Contains(scope, "model=gpt-5.5") {
		t.Fatalf("scope missing account/model: %q", scope)
	}
	if !strings.Contains(scope, "prompt_cache_key=") {
		t.Fatalf("scope missing cache key source: %q", scope)
	}
	if strings.Contains(scope, "raw-secret-session") {
		t.Fatalf("scope leaked raw prompt cache key: %q", scope)
	}
	if !strings.Contains(scope, hashSensitiveValueForLog("raw-secret-session")) {
		t.Fatalf("scope missing cache key hash: %q", scope)
	}
	if got := openAITokenSaverReferenceScope(nil, &Account{ID: 123}, []byte(`{"model":"gpt-5.5"}`)); got != "" {
		t.Fatalf("expected no scope without prompt_cache_key, got %q", got)
	}
}

func TestOpenAITokenSaverReferenceScopeFallsBackToStableSessionSignals(t *testing.T) {
	c := tokenSaverGinContext()
	c.Request.Header.Set("session_id", " header-session ")
	body := []byte(`{"model":"gpt-5.5","session_id":"body-session","prompt_cache_key":""}`)

	scope := openAITokenSaverReferenceScope(c, &Account{ID: 123}, body)

	if !strings.Contains(scope, "header_session_id=") || strings.Contains(scope, "header-session") {
		t.Fatalf("expected hashed header session scope, got %q", scope)
	}
	if !strings.Contains(scope, hashSensitiveValueForLog("header-session")) {
		t.Fatalf("scope missing header session hash: %q", scope)
	}

	metadata := `{"device_id":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","account_uuid":"","session_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"}`
	body = []byte(fmt.Sprintf(`{"model":"gpt-5.5","metadata":{"user_id":%q}}`, metadata))
	scope = openAITokenSaverReferenceScope(nil, &Account{ID: 123}, body)
	if !strings.Contains(scope, "metadata_user_session=") ||
		!strings.Contains(scope, hashSensitiveValueForLog("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")) {
		t.Fatalf("expected metadata session scope, got %q", scope)
	}
}

func TestOpenAITokenSaverReferenceScopePromptCacheKeyWinsOverFallback(t *testing.T) {
	c := tokenSaverGinContext()
	c.Request.Header.Set("session_id", "header-session")
	body := []byte(`{"model":"gpt-5.5","prompt_cache_key":"cache-key","session_id":"body-session"}`)

	scope := openAITokenSaverReferenceScope(c, &Account{ID: 123}, body)

	if !strings.Contains(scope, "prompt_cache_key=") ||
		!strings.Contains(scope, hashSensitiveValueForLog("cache-key")) ||
		strings.Contains(scope, hashSensitiveValueForLog("header-session")) {
		t.Fatalf("expected prompt cache key to win, got %q", scope)
	}
}

func TestAnalyzeOpenAITokenSaverRequestParts(t *testing.T) {
	body := []byte(`{
		"model": "gpt-5.5",
		"prompt_cache_key": "cache-key",
		"store": false,
		"tools": [{"type":"function","name":"shell"}],
		"instructions": "system text",
		"input": [
			{"type":"message","role":"user","content":"hello"},
			{"type":"function_call","name":"shell","arguments":"{}"},
			{"type":"function_call_output","call_id":"1","output":"long shell output"}
		]
	}`)

	parts := analyzeOpenAITokenSaverRequestParts(body)
	byPart := make(map[string]RTKRequestPartStats)
	for _, part := range parts {
		byPart[part.Part] = part
	}

	for _, name := range []string{
		"input/tool_output",
		"input/tool_output_meta",
		"input/message",
		"input/tool_call",
		"tools/schema",
		"instructions",
		"request/options",
		"json/overhead",
	} {
		if byPart[name].Bytes <= 0 {
			t.Fatalf("expected %s bytes in parts: %#v", name, parts)
		}
	}
	if byPart["input/tool_output"].Count != 1 {
		t.Fatalf("expected one tool output, got %#v", byPart["input/tool_output"])
	}
	if byPart["request/options"].Count != 3 {
		t.Fatalf("expected three request options, got %#v", byPart["request/options"])
	}
}

func tokenSaverGinContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	return c
}

func tokenSaverRequestBody() []byte {
	var b strings.Builder
	for f := 0; f < 3; f++ {
		fmt.Fprintf(&b, "diff --git a/file%d.go b/file%d.go\n--- a/file%d.go\n+++ b/file%d.go\n@@ -1,120 +1,120 @@\n", f, f, f, f)
		for i := 0; i < 160; i++ {
			fmt.Fprintf(&b, "-old line %d with verbose removed text\n+new line %d with verbose added text\n", i, i)
		}
	}
	return []byte(fmt.Sprintf(`{"input":[{"type":"function_call_output","output":%q}]}`, b.String()))
}
