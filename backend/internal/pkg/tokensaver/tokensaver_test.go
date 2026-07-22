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

func TestCompressJSON9RouterStrategyCompressesSmallToolOutput(t *testing.T) {
	body := fmt.Sprintf(`{"messages":[{"role":"tool","tool_call_id":"c1","content":%q}]}`, smallGrepOutput())

	result, err := CompressJSON([]byte(body), Options{Strategy: Strategy9Router})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed || len(result.Hits) != 1 || result.Hits[0].Filter != "grep" {
		t.Fatalf("expected 9router strategy to compress 500B+ grep output, got changed=%v hits=%#v", result.Changed, result.Hits)
	}
}

func TestCompressJSON9RouterStrategyCompressesResponsesOutputArray(t *testing.T) {
	body := fmt.Sprintf(`{"input":[{"type":"function_call_output","call_id":"c1","output":[{"type":"input_text","text":%q}]}]}`, largeCompilingLog())

	result, err := CompressJSON([]byte(body), Options{Strategy: Strategy9Router, MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed || len(result.Hits) != 1 || result.Hits[0].Path != "$.input[0].output[0].text" || result.Hits[0].Filter != "build-output" {
		t.Fatalf("unexpected hits: %#v changed=%v", result.Hits, result.Changed)
	}
}

func TestCompressJSON9RouterStrategyCompressesToolContentArray(t *testing.T) {
	body := fmt.Sprintf(`{"messages":[{"role":"tool","tool_call_id":"c1","content":[{"type":"text","text":%q}]}]}`, largeCompilingLog())

	result, err := CompressJSON([]byte(body), Options{Strategy: Strategy9Router, MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed || len(result.Hits) != 1 || result.Hits[0].Path != "$.messages[0].content[0].text" || result.Hits[0].Filter != "build-output" {
		t.Fatalf("unexpected hits: %#v changed=%v", result.Hits, result.Changed)
	}
}

func TestCompressJSON9RouterStrategyUsesAggressiveBuildOutput(t *testing.T) {
	body := fmt.Sprintf(`{"messages":[{"role":"tool","content":%q}]}`, largeMixedBuildOutput())

	result, err := CompressJSON([]byte(body), Options{Strategy: Strategy9Router, MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed || len(result.Hits) != 1 || result.Hits[0].Filter != "build-output" {
		t.Fatalf("unexpected result: changed=%v hits=%#v", result.Changed, result.Hits)
	}
	if result.Hits[0].After >= result.Hits[0].Before/5 {
		t.Fatalf("expected 9router build output to shrink aggressively: %#v", result.Hits[0])
	}
	var decoded struct {
		Messages []struct {
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(result.Body, &decoded); err != nil {
		t.Fatalf("compressed body is invalid JSON: %v", err)
	}
	output := decoded.Messages[0].Content
	if strings.Contains(output, "Downloading package_020") || strings.Contains(output, "Compiling crate_020") {
		t.Fatalf("expected verbose build progress to be stripped, got %q", output)
	}
	if !strings.Contains(output, "Compiled ") || !strings.Contains(output, "Downloaded ") || !strings.Contains(output, "npm warn deprecated") {
		t.Fatalf("expected build summary to be preserved, got %q", output)
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

func TestCompressJSONReferencesDuplicateShellOutputWithDifferentWallTime(t *testing.T) {
	outputA := shellOutputWithWallTime(largeCompilingLog(), "0.5 seconds")
	outputB := shellOutputWithWallTime(largeCompilingLog(), "1.2 seconds")
	body := fmt.Sprintf(`{"messages":[{"role":"tool","content":%q},{"role":"tool","content":%q}]}`, outputA, outputB)

	result, err := CompressJSON([]byte(body), Options{MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected duplicated shell output body to be compressed and referenced")
	}
	if len(result.Hits) != 3 {
		t.Fatalf("expected two compression hits plus one reference hit, got %#v", result.Hits)
	}
	if result.Hits[2].Filter != "duplicate_reference" || result.Hits[2].ReferencePath != "$.messages[0].content" {
		t.Fatalf("unexpected reference hit: %#v", result.Hits[2])
	}
}

func TestCompressJSONReferencesRepeatedHistoryAcrossRequests(t *testing.T) {
	resetRepeatHistoryForTesting()
	scope := "account=1|model=gpt-5.5|pc=cache"
	oldOutput := shellOutputWithWallTime(largeCompilingLog(), "0.5 seconds")
	newOutput := shellOutputWithWallTime(largeMediumUnstructuredOutput(), "0.7 seconds")
	firstBody := fmt.Sprintf(`{"input":[{"type":"function_call_output","output":%q}]}`, oldOutput)

	first, err := CompressJSON([]byte(firstBody), Options{MinBytes: 64, ReferenceScope: scope})
	if err != nil {
		t.Fatalf("first CompressJSON returned error: %v", err)
	}
	if !first.Changed || len(first.Hits) != 1 || first.Hits[0].Filter != "shell_output/build_log" {
		t.Fatalf("unexpected first result: changed=%v hits=%#v", first.Changed, first.Hits)
	}

	secondBody := fmt.Sprintf(`{"input":[{"type":"function_call_output","output":%q},{"type":"function_call_output","output":%q}]}`, oldOutput, newOutput)
	second, err := CompressJSON([]byte(secondBody), Options{MinBytes: 64, ReferenceScope: scope, ReferenceKeepRecent: 1})
	if err != nil {
		t.Fatalf("second CompressJSON returned error: %v", err)
	}
	if !second.Changed {
		t.Fatalf("expected repeated historical output to be referenced")
	}
	if len(second.Hits) != 3 {
		t.Fatalf("expected two compression hits plus one history reference, got %#v", second.Hits)
	}
	if second.Hits[2].Filter != "repeat_history_reference" || second.Hits[2].ReferencePath != "$.input[0].output" {
		t.Fatalf("unexpected history reference hit: %#v", second.Hits[2])
	}
	var decoded struct {
		Input []struct {
			Output string `json:"output"`
		} `json:"input"`
	}
	if err := json.Unmarshal(second.Body, &decoded); err != nil {
		t.Fatalf("second compressed body is invalid JSON: %v", err)
	}
	if !strings.Contains(decoded.Input[0].Output, "repeated historical tool output") {
		t.Fatalf("expected historical reference, got %q", decoded.Input[0].Output)
	}
	if strings.Contains(decoded.Input[1].Output, "repeated historical tool output") {
		t.Fatalf("expected newest output to be retained, got %q", decoded.Input[1].Output)
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

func TestCompressJSONGitLog(t *testing.T) {
	body := fmt.Sprintf(`{"input":[{"type":"function_call_output","output":%q}]}`, largeGitLog())

	result, err := CompressJSON([]byte(body), Options{MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed || len(result.Hits) != 1 || result.Hits[0].Filter != "git_log" {
		t.Fatalf("unexpected result: changed=%v hits=%#v", result.Changed, result.Hits)
	}
	if result.Hits[0].After >= result.Hits[0].Before/2 {
		t.Fatalf("expected git log to shrink substantially: %#v", result.Hits[0])
	}
}

func TestCompressJSONFindOutput(t *testing.T) {
	body := fmt.Sprintf(`{"messages":[{"role":"tool","content":%q}]}`, largeFindOutput())

	result, err := CompressJSON([]byte(body), Options{MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed || len(result.Hits) != 1 || result.Hits[0].Filter != "find" {
		t.Fatalf("unexpected result: changed=%v hits=%#v", result.Changed, result.Hits)
	}
}

func TestCompressJSONTreeOutput(t *testing.T) {
	body := fmt.Sprintf(`{"messages":[{"role":"tool","content":%q}]}`, largeTreeOutput())

	result, err := CompressJSON([]byte(body), Options{MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed || len(result.Hits) != 1 || result.Hits[0].Filter != "tree" {
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

func TestCompressJSONSearchList(t *testing.T) {
	body := fmt.Sprintf(`{"messages":[{"role":"tool","content":%q}]}`, largeSearchList())

	result, err := CompressJSON([]byte(body), Options{MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed || len(result.Hits) != 1 || result.Hits[0].Filter != "search_list" {
		t.Fatalf("unexpected result: changed=%v hits=%#v", result.Changed, result.Hits)
	}
}

func TestCompressJSONReadNumbered(t *testing.T) {
	body := fmt.Sprintf(`{"messages":[{"role":"tool","content":%q}]}`, largeReadNumberedOutput())

	result, err := CompressJSON([]byte(body), Options{MinBytes: 64})
	if err != nil {
		t.Fatalf("CompressJSON returned error: %v", err)
	}
	if !result.Changed || len(result.Hits) != 1 || result.Hits[0].Filter != "read_numbered" {
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

func TestCompressJSONShellOutputWrapsStructuredFilters(t *testing.T) {
	cases := []struct {
		name   string
		output string
		filter string
	}{
		{name: "git_log", output: largeGitLog(), filter: "shell_output/git_log"},
		{name: "find", output: largeFindOutput(), filter: "shell_output/find"},
		{name: "tree", output: largeTreeOutput(), filter: "shell_output/tree"},
		{name: "search_list", output: largeSearchList(), filter: "shell_output/search_list"},
		{name: "read_numbered", output: largeReadNumberedOutput(), filter: "shell_output/read_numbered"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := fmt.Sprintf(`{"messages":[{"role":"tool","content":%q}]}`, shellOutput(tc.output))

			result, err := CompressJSON([]byte(body), Options{MinBytes: 64})
			if err != nil {
				t.Fatalf("CompressJSON returned error: %v", err)
			}
			if !result.Changed || len(result.Hits) != 1 || result.Hits[0].Filter != tc.filter {
				t.Fatalf("unexpected result: changed=%v hits=%#v", result.Changed, result.Hits)
			}
		})
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

func smallGrepOutput() string {
	var b strings.Builder
	for i := 0; i < 12; i++ {
		fmt.Fprintf(&b, "src/file.go:%d: compact but repetitive match line %03d\n", i+1, i)
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

func largeMixedBuildOutput() string {
	var b strings.Builder
	for i := 0; i < 60; i++ {
		fmt.Fprintf(&b, "Downloading package_%03d v1.2.3\n", i)
	}
	for i := 0; i < 80; i++ {
		fmt.Fprintf(&b, "   Compiling crate_%03d v0.1.0\n", i)
	}
	for i := 0; i < 12; i++ {
		fmt.Fprintf(&b, "npm warn deprecated package-%03d@1.0.0: legacy package\n", i)
	}
	for i := 0; i < 9; i++ {
		fmt.Fprintf(&b, "npm warn optional warning %03d\n", i)
	}
	b.WriteString("error: failed to compile crate_alpha\n")
	b.WriteString("    Finished release [optimized] target(s) in 3m 12s\n")
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

func largeGitLog() string {
	var b strings.Builder
	for i := 0; i < 90; i++ {
		fmt.Fprintf(&b, "commit %040d\n", i)
		fmt.Fprintf(&b, "Author: Developer %d <dev%d@example.com>\n", i, i)
		fmt.Fprintf(&b, "Date:   Tue Jul 21 12:%02d:00 2026 +0800\n\n", i%60)
		fmt.Fprintf(&b, "    implement token saver change %03d with verbose details\n\n", i)
	}
	return b.String()
}

func largeFindOutput() string {
	var b strings.Builder
	for d := 0; d < 24; d++ {
		for i := 0; i < 16; i++ {
			fmt.Fprintf(&b, "./backend/internal/module_%02d/file_%03d.go\n", d, i)
		}
	}
	return b.String()
}

func largeTreeOutput() string {
	var b strings.Builder
	b.WriteString(".\n")
	for d := 0; d < 40; d++ {
		fmt.Fprintf(&b, "├── module_%02d\n", d)
		for i := 0; i < 8; i++ {
			fmt.Fprintf(&b, "│  ├── file_%03d.go\n", i)
		}
	}
	b.WriteString("└── go.mod\n")
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

func largeSearchList() string {
	var b strings.Builder
	b.WriteString("Result of search in 'backend' (total 260 files):\n")
	for d := 0; d < 26; d++ {
		for i := 0; i < 10; i++ {
			fmt.Fprintf(&b, "backend/internal/search_%02d/result_%03d.ts\n", d, i)
		}
	}
	return b.String()
}

func largeReadNumberedOutput() string {
	var b strings.Builder
	for i := 1; i <= 360; i++ {
		fmt.Fprintf(&b, "%4d| const value%03d = computeVerboseThing(%d)\n", i, i, i)
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
	return shellOutputWithWallTime(output, "0.5 seconds")
}

func shellOutputWithWallTime(output, wallTime string) string {
	return "Exit code: 0\nWall time: " + wallTime + "\nOutput:\n" + output
}
