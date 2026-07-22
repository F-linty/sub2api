package tokensaver

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestCompressJSONResponsesFunctionCallOutput(t *testing.T) {
	body := fmt.Sprintf(`{"input":[{"type":"function_call_output","call_id":"c1","output":%q}]}`, largeGitDiff())

	result, err := CompressJSON([]byte(body), Options{MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected body to be compressed")
	}
	if len(result.Hits) != 1 || result.Hits[0].Path != "$.input[0].output" || result.Hits[0].Filter != "git_diff" {
		t.Fatalf("unexpected hits: %#v", result.Hits)
	}
	var decoded struct {
		Input []struct {
			Output string `json:"output"`
		} `json:"input"`
	}
	if err := json.Unmarshal(result.Body, &decoded); err != nil {
		t.Fatalf("compressed body is invalid JSON: %v", err)
	}
	if !strings.Contains(decoded.Input[0].Output, "sub2api token saver") {
		t.Fatalf("missing compression marker: %q", decoded.Input[0].Output)
	}
}

func TestCompressJSONChatToolMessage(t *testing.T) {
	body := fmt.Sprintf(`{"messages":[{"role":"tool","tool_call_id":"c1","content":%q}]}`, largeGrepOutput())

	result, err := CompressJSON([]byte(body), Options{MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected body to be compressed")
	}
	if len(result.Hits) != 1 || result.Hits[0].Filter != "grep" {
		t.Fatalf("unexpected hits: %#v", result.Hits)
	}
}

func TestCompressJSONAnthropicToolResultStringAndArray(t *testing.T) {
	body := fmt.Sprintf(`{"messages":[{"role":"user","content":[{"type":"tool_result","content":%q},{"type":"tool_result","content":[{"type":"text","text":%q}]}]}]}`, largeBuildLog(), largeFileList())

	result, err := CompressJSON([]byte(body), Options{MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected body to be compressed")
	}
	if len(result.Hits) != 2 {
		t.Fatalf("expected 2 hits, got %#v", result.Hits)
	}
}

func TestCompressJSONSkipsErrorToolResult(t *testing.T) {
	body := fmt.Sprintf(`{"input":[{"type":"function_call_output","status":"error","output":%q}],"messages":[{"role":"user","content":[{"type":"tool_result","is_error":true,"content":%q}]}]}`, largeGitDiff(), largeBuildLog())

	result, err := CompressJSON([]byte(body), Options{MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if result.Changed {
		t.Fatalf("expected error tool results to be left unchanged")
	}
}

func TestCompressJSONLeavesUnsupportedOrInvalidBodyUnchanged(t *testing.T) {
	cases := [][]byte{
		[]byte(`not-json`),
		[]byte(`{"messages":[{"role":"user","content":"hello"}]}`),
		[]byte(`{"input":[{"type":"function_call_output","output":"short"}]}`),
	}
	for _, body := range cases {
		result, err := CompressJSON(body, Options{MinBytes: 64})
		if err != nil {
			t.Fatalf("CompressJSON returned error: %v", err)
		}
		if result.Changed || string(result.Body) != string(body) {
			t.Fatalf("expected unchanged body for %s", string(body))
		}
	}
}

func TestCompressJSONSkipsAlreadyCompressedMarker(t *testing.T) {
	output := "[sub2api token saver: compressed file_list; original 9999 bytes]\nentries=100"
	body := fmt.Sprintf(`{"input":[{"type":"function_call_output","output":%q}]}`, output)

	result, err := CompressJSON([]byte(body), Options{MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if result.Changed {
		t.Fatalf("expected already compressed content to be skipped")
	}
}

func TestCompressJSONReferencesDuplicateCompressedHistory(t *testing.T) {
	output := "[sub2api token saver: compressed shell_output/build_log; original 9999 bytes]\n" + strings.Repeat("compiled package with useful summary\n", 80)
	body := fmt.Sprintf(`{"input":[{"type":"function_call_output","output":%q},{"type":"function_call_output","output":%q}]}`, output, output)

	result, err := CompressJSON([]byte(body), Options{MinBytes: 2048})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected duplicate compressed history to be referenced")
	}
	if len(result.Hits) != 1 || result.Hits[0].Filter != "duplicate_reference" {
		t.Fatalf("unexpected hits: %#v", result.Hits)
	}
	if result.Hits[0].ReferenceHash == "" || result.Hits[0].ReferencePath != "$.input[0].output" {
		t.Fatalf("missing reference metadata: %#v", result.Hits[0])
	}
	var decoded struct {
		Input []struct {
			Output string `json:"output"`
		} `json:"input"`
	}
	if err := json.Unmarshal(result.Body, &decoded); err != nil {
		t.Fatalf("compressed body is invalid JSON: %v", err)
	}
	if !strings.Contains(decoded.Input[1].Output, "duplicate reference") || !strings.Contains(decoded.Input[1].Output, "$.input[0].output") {
		t.Fatalf("expected duplicate reference, got %q", decoded.Input[1].Output)
	}
}

func TestCompressJSONCompressesThenReferencesDuplicateToolOutput(t *testing.T) {
	output := shellOutput(largeCompilingLog())
	body := fmt.Sprintf(`{"messages":[{"role":"tool","content":%q},{"role":"tool","content":%q}]}`, output, output)

	result, err := CompressJSON([]byte(body), Options{MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected duplicated output to be compressed and referenced")
	}
	if len(result.Hits) != 3 {
		t.Fatalf("expected two compression hits plus one reference hit, got %#v", result.Hits)
	}
	if result.Hits[2].Filter != "duplicate_reference" || result.Hits[2].ReferencePath != "$.messages[0].content" {
		t.Fatalf("unexpected reference hit: %#v", result.Hits[2])
	}
}

func TestCompressJSONGitStatus(t *testing.T) {
	body := fmt.Sprintf(`{"input":[{"type":"function_call_output","output":%q}]}`, largeGitStatus())

	result, err := CompressJSON([]byte(body), Options{MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed || len(result.Hits) != 1 || result.Hits[0].Filter != "git_status" {
		t.Fatalf("unexpected result: changed=%v hits=%#v", result.Changed, result.Hits)
	}
}

func TestCompressJSONLSOutput(t *testing.T) {
	body := fmt.Sprintf(`{"messages":[{"role":"tool","content":%q}]}`, largeLSOutput())

	result, err := CompressJSON([]byte(body), Options{MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed || len(result.Hits) != 1 || result.Hits[0].Filter != "ls" {
		t.Fatalf("unexpected result: changed=%v hits=%#v", result.Changed, result.Hits)
	}
}

func TestCompressJSONDedupLogFallback(t *testing.T) {
	body := fmt.Sprintf(`{"messages":[{"role":"tool","content":%q}]}`, largeDuplicateLog())

	result, err := CompressJSON([]byte(body), Options{MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed || len(result.Hits) != 1 || result.Hits[0].Filter != "dedup_log" {
		t.Fatalf("unexpected result: changed=%v hits=%#v", result.Changed, result.Hits)
	}
}

func TestCompressJSONSmartTruncateFallback(t *testing.T) {
	body := fmt.Sprintf(`{"messages":[{"role":"tool","content":%q}]}`, largeUnstructuredOutput())

	result, err := CompressJSON([]byte(body), Options{MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed || len(result.Hits) != 1 || result.Hits[0].Filter != "smart_truncate" {
		t.Fatalf("unexpected result: changed=%v hits=%#v", result.Changed, result.Hits)
	}
}

func TestCompressJSONBuildLogCompilingOnly(t *testing.T) {
	body := fmt.Sprintf(`{"messages":[{"role":"tool","content":%q}]}`, largeCompilingLog())

	result, err := CompressJSON([]byte(body), Options{MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed || len(result.Hits) != 1 || result.Hits[0].Filter != "build_log" {
		t.Fatalf("unexpected result: changed=%v hits=%#v", result.Changed, result.Hits)
	}
}

func TestCompressJSONShellOutputWrapsInnerFilter(t *testing.T) {
	body := fmt.Sprintf(`{"messages":[{"role":"tool","content":%q}]}`, shellOutput(largeCompilingLog()))

	result, err := CompressJSON([]byte(body), Options{MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed || len(result.Hits) != 1 || result.Hits[0].Filter != "shell_output/build_log" {
		t.Fatalf("unexpected result: changed=%v hits=%#v", result.Changed, result.Hits)
	}
	var decoded struct {
		Messages []struct {
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(result.Body, &decoded); err != nil {
		t.Fatalf("compressed body is invalid JSON: %v", err)
	}
	if !strings.Contains(decoded.Messages[0].Content, "Output: compressed via build_log") {
		t.Fatalf("expected inner filter note, got %q", decoded.Messages[0].Content)
	}
}

func TestCompressJSONShellOutputFallbackTruncates(t *testing.T) {
	body := fmt.Sprintf(`{"messages":[{"role":"tool","content":%q}]}`, shellOutput(largeMediumUnstructuredOutput()))

	result, err := CompressJSON([]byte(body), Options{MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed || len(result.Hits) != 1 || result.Hits[0].Filter != "shell_output/truncate" {
		t.Fatalf("unexpected result: changed=%v hits=%#v misses=%#v", result.Changed, result.Hits, result.Misses)
	}
}

func TestCompressJSONRecordsLargeMiss(t *testing.T) {
	body := fmt.Sprintf(`{"messages":[{"role":"tool","content":%q}]}`, largeSingleLineBlob())

	result, err := CompressJSON([]byte(body), Options{MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if result.Changed {
		t.Fatalf("expected uncompressed body")
	}
	if len(result.Misses) != 1 {
		t.Fatalf("expected one miss, got %#v", result.Misses)
	}
	if result.Misses[0].Path != "$.messages[0].content" || result.Misses[0].Reason != "no_filter" {
		t.Fatalf("unexpected miss: %#v", result.Misses[0])
	}
}

func largeGitDiff() string {
	var b strings.Builder
	for f := 0; f < 4; f++ {
		fmt.Fprintf(&b, "diff --git a/file%d.go b/file%d.go\n--- a/file%d.go\n+++ b/file%d.go\n@@ -1,120 +1,120 @@\n", f, f, f, f)
		for i := 0; i < 160; i++ {
			fmt.Fprintf(&b, "-old line %d with a fair amount of removed text\n+new line %d with a fair amount of added text\n", i, i)
		}
	}
	return b.String()
}

func largeGrepOutput() string {
	var b strings.Builder
	for f := 0; f < 10; f++ {
		for i := 0; i < 20; i++ {
			fmt.Fprintf(&b, "backend/internal/file%d.go:%d: matched line with many details here\n", f, i+1)
		}
	}
	return b.String()
}

func largeBuildLog() string {
	var b strings.Builder
	for i := 0; i < 140; i++ {
		if i == 70 {
			b.WriteString("ERROR: package build failed with exit status 1\n")
			continue
		}
		fmt.Fprintf(&b, "compile step %d completed with verbose diagnostic output\n", i)
	}
	return b.String()
}

func largeFileList() string {
	var b strings.Builder
	for i := 0; i < 160; i++ {
		fmt.Fprintf(&b, "backend/internal/pkg/example/file_%03d.go\n", i)
	}
	return b.String()
}

func largeGitStatus() string {
	var b strings.Builder
	b.WriteString("On branch main\nChanges not staged for commit:\n")
	for i := 0; i < 80; i++ {
		fmt.Fprintf(&b, " M backend/internal/pkg/example/file_%03d.go\n", i)
	}
	b.WriteString("Untracked files:\n")
	for i := 0; i < 80; i++ {
		fmt.Fprintf(&b, "?? frontend/src/example/file_%03d.ts\n", i)
	}
	return b.String()
}

func largeLSOutput() string {
	var b strings.Builder
	b.WriteString("total 2000\n")
	for i := 0; i < 120; i++ {
		fmt.Fprintf(&b, "-rw-r--r-- 1 user group %d Jul 18 18:26 file_%03d.go\n", 1024+i, i)
	}
	for i := 0; i < 12; i++ {
		fmt.Fprintf(&b, "drwxr-xr-x 1 user group 4096 Jul 18 18:26 dir_%03d\n", i)
	}
	return b.String()
}

func largeDuplicateLog() string {
	var b strings.Builder
	for i := 0; i < 120; i++ {
		b.WriteString("polling remote cache with no changes\n")
	}
	for i := 0; i < 120; i++ {
		b.WriteString("waiting for worker slot\n")
	}
	return b.String()
}

func largeUnstructuredOutput() string {
	var b strings.Builder
	for i := 0; i < 420; i++ {
		fmt.Fprintf(&b, "verbose diagnostic row %03d with data that does not match a dedicated filter\n", i)
	}
	return b.String()
}

func largeMediumUnstructuredOutput() string {
	var b strings.Builder
	for i := 0; i < 140; i++ {
		fmt.Fprintf(&b, "command row %03d with useful but repetitive terminal data\n", i)
	}
	return b.String()
}

func largeCompilingLog() string {
	var b strings.Builder
	for i := 0; i < 160; i++ {
		fmt.Fprintf(&b, "   Compiling crate_%03d v0.1.0\n", i)
	}
	b.WriteString("    Finished release [optimized] target(s) in 3m 12s\n")
	return b.String()
}

func largeSingleLineBlob() string {
	return strings.Repeat("opaque-token-stream-without-newlines-", 120)
}

func shellOutput(output string) string {
	return "Exit code: 0\nWall time: 0.5 seconds\nOutput:\n" + output
}
