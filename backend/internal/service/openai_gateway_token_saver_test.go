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
