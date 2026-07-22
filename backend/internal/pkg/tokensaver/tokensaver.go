package tokensaver

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultMinBytes            = 2048
	defaultMaxBytes            = 512 << 10
	markerPrefix               = "[sub2api token saver:"
	referenceMinBytes          = 768
	defaultReferenceKeepRecent = 2
	repeatHistoryTTL           = 2 * time.Hour
	repeatHistoryMaxScopes     = 256
	repeatHistoryMaxPerScope   = 768
)

// Options controls conservative tool-output compression.
type Options struct {
	MinBytes            int
	MaxBytes            int
	ReferenceScope      string
	ReferenceKeepRecent int
}

// Hit describes one compressed JSON field.
type Hit struct {
	Path          string
	Filter        string
	Before        int
	After         int
	ReferenceHash string
	ReferencePath string
}

// Miss describes an eligible tool-output field that was not compressed.
type Miss struct {
	Path   string
	Before int
	Reason string
	Sample string
}

// Result is returned by CompressJSON.
type Result struct {
	Body    []byte
	Changed bool
	Before  int
	After   int
	Hits    []Hit
	Misses  []Miss
}

// CompressJSON rewrites known tool-result fields in OpenAI/Anthropic-shaped JSON.
// It is intentionally fail-open: unsupported shapes simply return the original body.
func CompressJSON(body []byte, opts Options) (Result, error) {
	result := Result{Body: body, Before: len(body), After: len(body)}
	if len(bytes.TrimSpace(body)) == 0 {
		return result, nil
	}
	opts = normalizeOptions(opts)

	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return result, nil
	}

	hits, misses := mutateKnownToolOutputs(root, opts)
	hits = append(hits, dedupeRepeatedToolOutputs(root, opts)...)
	hits = append(hits, referenceRepeatedHistoryOutputs(root, opts)...)
	result.Misses = misses
	if len(hits) == 0 {
		return result, nil
	}

	encoded, err := json.Marshal(root)
	if err != nil || len(encoded) >= len(body) {
		return result, nil
	}
	result.Body = encoded
	result.Changed = true
	result.After = len(encoded)
	result.Hits = hits
	return result, nil
}

func normalizeOptions(opts Options) Options {
	if opts.MinBytes <= 0 {
		opts.MinBytes = defaultMinBytes
	}
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = defaultMaxBytes
	}
	if opts.MaxBytes < opts.MinBytes {
		opts.MaxBytes = opts.MinBytes
	}
	return opts
}

func mutateKnownToolOutputs(root any, opts Options) ([]Hit, []Miss) {
	obj, ok := root.(map[string]any)
	if !ok {
		return nil, nil
	}
	var hits []Hit
	var misses []Miss

	if messages, ok := obj["messages"].([]any); ok {
		for i, raw := range messages {
			msg, ok := raw.(map[string]any)
			if !ok || isErrorToolResult(msg) {
				continue
			}
			if role, _ := msg["role"].(string); role == "tool" {
				fieldHits, fieldMisses := compressStringField(msg, "content", fmt.Sprintf("$.messages[%d].content", i), opts)
				hits = append(hits, fieldHits...)
				misses = append(misses, fieldMisses...)
			}
			contentHits, contentMisses := mutateAnthropicContent(msg["content"], fmt.Sprintf("$.messages[%d].content", i), opts)
			hits = append(hits, contentHits...)
			misses = append(misses, contentMisses...)
		}
	}

	if input, ok := obj["input"].([]any); ok {
		for i, raw := range input {
			item, ok := raw.(map[string]any)
			if !ok || isErrorToolResult(item) {
				continue
			}
			if typ, _ := item["type"].(string); typ == "function_call_output" {
				fieldHits, fieldMisses := compressStringField(item, "output", fmt.Sprintf("$.input[%d].output", i), opts)
				hits = append(hits, fieldHits...)
				misses = append(misses, fieldMisses...)
			}
			contentHits, contentMisses := mutateAnthropicContent(item["content"], fmt.Sprintf("$.input[%d].content", i), opts)
			hits = append(hits, contentHits...)
			misses = append(misses, contentMisses...)
		}
	}

	return hits, misses
}

type repeatHistoryEntry struct {
	hash     string
	path     string
	lastSeen time.Time
}

type repeatHistoryScope struct {
	lastSeen time.Time
	entries  map[string]repeatHistoryEntry
	order    []string
}

type repeatHistoryStore struct {
	mu     sync.Mutex
	scopes map[string]*repeatHistoryScope
}

var defaultRepeatHistoryStore = &repeatHistoryStore{scopes: make(map[string]*repeatHistoryScope)}

type repeatHistoryCandidate struct {
	field     toolStringField
	value     string
	canonical string
	hash      string
	key       string
}

func resetRepeatHistoryForTesting() {
	defaultRepeatHistoryStore.mu.Lock()
	defer defaultRepeatHistoryStore.mu.Unlock()
	defaultRepeatHistoryStore.scopes = make(map[string]*repeatHistoryScope)
}

type toolStringField struct {
	obj  map[string]any
	key  string
	path string
}

func collectKnownToolStringFields(root any) []toolStringField {
	obj, ok := root.(map[string]any)
	if !ok {
		return nil
	}
	var fields []toolStringField
	if messages, ok := obj["messages"].([]any); ok {
		for i, raw := range messages {
			msg, ok := raw.(map[string]any)
			if !ok || isErrorToolResult(msg) {
				continue
			}
			if role, _ := msg["role"].(string); role == "tool" {
				fields = append(fields, toolStringField{obj: msg, key: "content", path: fmt.Sprintf("$.messages[%d].content", i)})
			}
			fields = append(fields, collectAnthropicContentStringFields(msg["content"], fmt.Sprintf("$.messages[%d].content", i))...)
		}
	}
	if input, ok := obj["input"].([]any); ok {
		for i, raw := range input {
			item, ok := raw.(map[string]any)
			if !ok || isErrorToolResult(item) {
				continue
			}
			if typ, _ := item["type"].(string); typ == "function_call_output" {
				fields = append(fields, toolStringField{obj: item, key: "output", path: fmt.Sprintf("$.input[%d].output", i)})
			}
			fields = append(fields, collectAnthropicContentStringFields(item["content"], fmt.Sprintf("$.input[%d].content", i))...)
		}
	}
	return fields
}

func collectAnthropicContentStringFields(raw any, basePath string) []toolStringField {
	parts, ok := raw.([]any)
	if !ok {
		return nil
	}
	var fields []toolStringField
	for i, rawPart := range parts {
		part, ok := rawPart.(map[string]any)
		if !ok || isErrorToolResult(part) {
			continue
		}
		if typ, _ := part["type"].(string); typ != "tool_result" {
			continue
		}
		partPath := fmt.Sprintf("%s[%d]", basePath, i)
		fields = append(fields, toolStringField{obj: part, key: "content", path: partPath + ".content"})
		if nested, ok := part["content"].([]any); ok {
			for j, rawNested := range nested {
				textPart, ok := rawNested.(map[string]any)
				if !ok {
					continue
				}
				if typ, _ := textPart["type"].(string); typ == "" || typ == "text" {
					fields = append(fields, toolStringField{obj: textPart, key: "text", path: fmt.Sprintf("%s.content[%d].text", partPath, j)})
				}
			}
		}
	}
	return fields
}

func dedupeRepeatedToolOutputs(root any, opts Options) []Hit {
	fields := collectKnownToolStringFields(root)
	if len(fields) < 2 {
		return nil
	}
	minBytes := opts.MinBytes
	if minBytes <= 0 || minBytes > referenceMinBytes {
		minBytes = referenceMinBytes
	}
	type firstSeen struct {
		path string
		hash string
	}
	seen := make(map[string]firstSeen)
	var hits []Hit
	for _, field := range fields {
		value, ok := field.obj[field.key].(string)
		if !ok || len(value) < minBytes || strings.Contains(value, "duplicate reference") {
			continue
		}
		canonical := canonicalReferenceText(value)
		if len(canonical) < minBytes {
			continue
		}
		hash := shortContentHash(canonical)
		if first, ok := seen[canonical]; ok {
			replacement := wrapDuplicateReference(first.hash, len(value), first.path)
			if len(replacement) >= len(value) {
				continue
			}
			field.obj[field.key] = replacement
			hits = append(hits, Hit{
				Path:          field.path,
				Filter:        "duplicate_reference",
				Before:        len(value),
				After:         len(replacement),
				ReferenceHash: first.hash,
				ReferencePath: first.path,
			})
			continue
		}
		seen[canonical] = firstSeen{path: field.path, hash: hash}
	}
	return hits
}

func referenceRepeatedHistoryOutputs(root any, opts Options) []Hit {
	scope := strings.TrimSpace(opts.ReferenceScope)
	if scope == "" {
		return nil
	}
	fields := collectKnownToolStringFields(root)
	if len(fields) == 0 {
		return nil
	}
	minBytes := opts.MinBytes
	if minBytes <= 0 || minBytes > referenceMinBytes {
		minBytes = referenceMinBytes
	}
	keepRecent := opts.ReferenceKeepRecent
	if keepRecent <= 0 {
		keepRecent = defaultReferenceKeepRecent
	}
	candidates := make([]repeatHistoryCandidate, 0, len(fields))
	for _, field := range fields {
		value, ok := field.obj[field.key].(string)
		if !ok || len(value) < minBytes || strings.Contains(value, "duplicate reference") || strings.Contains(value, "repeated historical tool output") {
			continue
		}
		canonical := canonicalReferenceText(value)
		if len(canonical) < minBytes {
			continue
		}
		candidates = append(candidates, repeatHistoryCandidate{
			field:     field,
			value:     value,
			canonical: canonical,
			hash:      shortContentHash(canonical),
			key:       repeatHistoryKey(canonical),
		})
	}
	if len(candidates) == 0 {
		return nil
	}
	replaceBefore := len(candidates) - keepRecent
	if replaceBefore < 0 {
		replaceBefore = 0
	}
	now := time.Now()
	seen := defaultRepeatHistoryStore.lookup(scope, candidates, replaceBefore, now)
	var hits []Hit
	for i := 0; i < replaceBefore; i++ {
		candidate := candidates[i]
		entry, ok := seen[candidate.key]
		if !ok {
			continue
		}
		replacement := wrapRepeatHistoryReference(entry.hash, len(candidate.value), entry.path)
		if len(replacement) >= len(candidate.value) {
			continue
		}
		candidate.field.obj[candidate.field.key] = replacement
		hits = append(hits, Hit{
			Path:          candidate.field.path,
			Filter:        "repeat_history_reference",
			Before:        len(candidate.value),
			After:         len(replacement),
			ReferenceHash: entry.hash,
			ReferencePath: entry.path,
		})
	}
	defaultRepeatHistoryStore.remember(scope, candidates, now)
	return hits
}

func (s *repeatHistoryStore) lookup(scope string, candidates []repeatHistoryCandidate, replaceBefore int, now time.Time) map[string]repeatHistoryEntry {
	if s == nil || replaceBefore <= 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(now)
	scopeState := s.scopes[scope]
	if scopeState == nil {
		return nil
	}
	scopeState.lastSeen = now
	seen := make(map[string]repeatHistoryEntry)
	for i := 0; i < replaceBefore && i < len(candidates); i++ {
		if entry, ok := scopeState.entries[candidates[i].key]; ok {
			entry.lastSeen = now
			scopeState.entries[candidates[i].key] = entry
			seen[candidates[i].key] = entry
		}
	}
	return seen
}

func (s *repeatHistoryStore) remember(scope string, candidates []repeatHistoryCandidate, now time.Time) {
	if s == nil || strings.TrimSpace(scope) == "" || len(candidates) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(now)
	scopeState := s.scopes[scope]
	if scopeState == nil {
		scopeState = &repeatHistoryScope{
			lastSeen: now,
			entries:  make(map[string]repeatHistoryEntry),
		}
		s.scopes[scope] = scopeState
	}
	scopeState.lastSeen = now
	for _, candidate := range candidates {
		entry, exists := scopeState.entries[candidate.key]
		if exists {
			entry.lastSeen = now
			scopeState.entries[candidate.key] = entry
			continue
		}
		scopeState.entries[candidate.key] = repeatHistoryEntry{
			hash:     candidate.hash,
			path:     candidate.field.path,
			lastSeen: now,
		}
		scopeState.order = append(scopeState.order, candidate.key)
	}
	s.evictEntriesLocked(scopeState)
	s.evictScopesLocked()
}

func (s *repeatHistoryStore) cleanupLocked(now time.Time) {
	for scope, scopeState := range s.scopes {
		if now.Sub(scopeState.lastSeen) > repeatHistoryTTL {
			delete(s.scopes, scope)
			continue
		}
		for canonical, entry := range scopeState.entries {
			if now.Sub(entry.lastSeen) > repeatHistoryTTL {
				delete(scopeState.entries, canonical)
			}
		}
		compactRepeatHistoryOrder(scopeState)
		if len(scopeState.entries) == 0 {
			delete(s.scopes, scope)
		}
	}
}

func (s *repeatHistoryStore) evictEntriesLocked(scopeState *repeatHistoryScope) {
	if scopeState == nil {
		return
	}
	for len(scopeState.entries) > repeatHistoryMaxPerScope && len(scopeState.order) > 0 {
		oldest := scopeState.order[0]
		scopeState.order = scopeState.order[1:]
		delete(scopeState.entries, oldest)
	}
	if len(scopeState.order) > repeatHistoryMaxPerScope*2 {
		compactRepeatHistoryOrder(scopeState)
	}
}

func compactRepeatHistoryOrder(scopeState *repeatHistoryScope) {
	if scopeState == nil || len(scopeState.order) == 0 {
		return
	}
	compacted := scopeState.order[:0]
	for _, canonical := range scopeState.order {
		if _, ok := scopeState.entries[canonical]; ok {
			compacted = append(compacted, canonical)
		}
	}
	scopeState.order = compacted
}

func (s *repeatHistoryStore) evictScopesLocked() {
	for len(s.scopes) > repeatHistoryMaxScopes {
		oldestScope := ""
		var oldest time.Time
		for scope, scopeState := range s.scopes {
			if oldestScope == "" || scopeState.lastSeen.Before(oldest) {
				oldestScope = scope
				oldest = scopeState.lastSeen
			}
		}
		if oldestScope == "" {
			return
		}
		delete(s.scopes, oldestScope)
	}
}

func canonicalReferenceText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.TrimRight(value, "\n")
	if !strings.Contains(value, markerPrefix) {
		return value
	}
	lines := splitLines(value)
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, markerPrefix):
			lines[i] = tokenSaverOriginalBytesRE.ReplaceAllString(line, "original <bytes> bytes")
		case strings.HasPrefix(line, "Wall time:"):
			lines[i] = "Wall time: <elapsed>"
		}
	}
	return strings.Join(lines, "\n")
}

func shortContentHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", sum[:8])
}

func repeatHistoryKey(value string) string {
	return fmt.Sprintf("%s:%d", shortContentHash(value), len(value))
}

func wrapDuplicateReference(hash string, before int, firstPath string) string {
	return fmt.Sprintf("%s duplicate reference; hash %s; original %d bytes; first %s]\nSame tool output as %s.", markerPrefix, hash, before, firstPath, firstPath)
}

func wrapRepeatHistoryReference(hash string, before int, firstPath string) string {
	return fmt.Sprintf("%s repeated historical tool output; hash %s; original %d bytes; first %s]\nThis tool output was already observed earlier in this Codex context.", markerPrefix, hash, before, firstPath)
}

func mutateAnthropicContent(raw any, path string, opts Options) ([]Hit, []Miss) {
	parts, ok := raw.([]any)
	if !ok {
		return nil, nil
	}
	var hits []Hit
	var misses []Miss
	for i, rawPart := range parts {
		part, ok := rawPart.(map[string]any)
		if !ok || isErrorToolResult(part) {
			continue
		}
		if typ, _ := part["type"].(string); typ != "tool_result" {
			continue
		}
		partPath := fmt.Sprintf("%s[%d]", path, i)
		fieldHits, fieldMisses := compressStringField(part, "content", partPath+".content", opts)
		hits = append(hits, fieldHits...)
		misses = append(misses, fieldMisses...)
		if nested, ok := part["content"].([]any); ok {
			for j, rawNested := range nested {
				textPart, ok := rawNested.(map[string]any)
				if !ok {
					continue
				}
				if typ, _ := textPart["type"].(string); typ == "" || typ == "text" {
					nestedHits, nestedMisses := compressStringField(textPart, "text", fmt.Sprintf("%s.content[%d].text", partPath, j), opts)
					hits = append(hits, nestedHits...)
					misses = append(misses, nestedMisses...)
				}
			}
		}
	}
	return hits, misses
}

func compressStringField(obj map[string]any, key, path string, opts Options) ([]Hit, []Miss) {
	value, ok := obj[key].(string)
	if !ok {
		return nil, nil
	}
	compressed, filter, reason, ok := compressToolText(value, opts)
	if !ok {
		if miss := newMiss(path, value, opts, reason); miss != nil {
			return nil, []Miss{*miss}
		}
		return nil, nil
	}
	obj[key] = compressed
	return []Hit{{Path: path, Filter: filter, Before: len(value), After: len(compressed)}}, nil
}

func isErrorToolResult(obj map[string]any) bool {
	if isErr, ok := obj["is_error"].(bool); ok && isErr {
		return true
	}
	if status, ok := obj["status"].(string); ok && strings.EqualFold(strings.TrimSpace(status), "error") {
		return true
	}
	return false
}

func compressToolText(text string, opts Options) (string, string, string, bool) {
	size := len(text)
	if size < opts.MinBytes {
		return "", "", "below_min", false
	}
	if size > opts.MaxBytes {
		return "", "", "above_max", false
	}
	if strings.Contains(text, markerPrefix) {
		return "", "", "already_compressed", false
	}
	if body, filter, ok := compressShellOutputNamed(text); ok {
		out := wrapCompressed(filter, size, body)
		if len(out) < size {
			return out, filter, "", true
		}
		return "", "", "not_smaller", false
	}
	matched := false
	candidates := []struct {
		name string
		fn   func(string) (string, bool)
	}{
		{"git_log", compressGitLog},
		{"git_diff", compressGitDiff},
		{"git_status", compressGitStatus},
		{"build_log", compressBuildLog},
		{"grep", compressGrep},
		{"find", compressFindOutput},
		{"tree", compressTreeOutput},
		{"ls", compressLSOutput},
		{"search_list", compressSearchList},
		{"read_numbered", compressReadNumbered},
		{"dedup_log", compressDedupLog},
		{"smart_truncate", compressSmartTruncate},
		{"file_list", compressFileList},
	}
	for _, candidate := range candidates {
		body, ok := candidate.fn(text)
		if !ok {
			continue
		}
		matched = true
		out := wrapCompressed(candidate.name, size, body)
		if len(out) < size {
			return out, candidate.name, "", true
		}
	}
	if matched {
		return "", "", "not_smaller", false
	}
	return "", "", "no_filter", false
}

func newMiss(path, value string, opts Options, reason string) *Miss {
	size := len(value)
	if size < opts.MinBytes || reason == "below_min" || reason == "already_compressed" {
		return nil
	}
	if reason == "" {
		reason = "no_filter"
	}
	return &Miss{
		Path:   path,
		Before: size,
		Reason: reason,
		Sample: sampleToolOutput(value),
	}
}

func sampleToolOutput(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.ReplaceAll(trimmed, "\r\n", "\n")
	lines := strings.Split(trimmed, "\n")
	sample := strings.Join(lines[:min(len(lines), 3)], "\n")
	if len(sample) > 300 {
		sample = sample[:300]
	}
	return sample
}

func wrapCompressed(filter string, before int, body string) string {
	body = strings.TrimSpace(body)
	return fmt.Sprintf("%s compressed %s; original %d bytes]\n%s", markerPrefix, filter, before, body)
}

func compressGitDiff(text string) (string, bool) {
	if !strings.Contains(text, "diff --git ") && !strings.Contains(text, "\n@@ ") {
		return "", false
	}
	lines := splitLines(text)
	var out []string
	files := 0
	hunks := 0
	keptChanges := 0
	const maxChanges = 120
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			files++
			out = append(out, line)
		case strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "@@ "):
			if strings.HasPrefix(line, "@@ ") {
				hunks++
			}
			out = append(out, line)
		case len(line) > 0 && (line[0] == '+' || line[0] == '-') && keptChanges < maxChanges:
			out = append(out, line)
			keptChanges++
		}
	}
	if files == 0 && hunks == 0 {
		return "", false
	}
	return fmt.Sprintf("files=%d hunks=%d kept_changed_lines=%d total_lines=%d\n%s", files, hunks, keptChanges, len(lines), strings.Join(out, "\n")), true
}

var grepLineRE = regexp.MustCompile(`^(.+?):(\d+)(?::\d+)?:`)
var buildSummaryRE = regexp.MustCompile(`^(added|removed|changed|audited|installed)\s+\d+\s+package`)
var tokenSaverOriginalBytesRE = regexp.MustCompile(`original \d+ bytes`)

func compressGrep(text string) (string, bool) {
	lines := splitLines(text)
	type fileStats struct {
		count int
		first []string
	}
	files := map[string]*fileStats{}
	matches := 0
	for _, line := range lines {
		m := grepLineRE.FindStringSubmatch(line)
		if len(m) == 0 {
			continue
		}
		matches++
		stats := files[m[1]]
		if stats == nil {
			stats = &fileStats{}
			files[m[1]] = stats
		}
		stats.count++
		if len(stats.first) < 3 {
			stats.first = append(stats.first, line)
		}
	}
	if matches < 20 || len(files) == 0 {
		return "", false
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var out []string
	out = append(out, fmt.Sprintf("matches=%d files=%d total_lines=%d", matches, len(files), len(lines)))
	for i, name := range names {
		if i >= 40 {
			out = append(out, fmt.Sprintf("... %d more files omitted", len(names)-i))
			break
		}
		stats := files[name]
		out = append(out, fmt.Sprintf("%s matches=%d", name, stats.count))
		out = append(out, stats.first...)
	}
	return strings.Join(out, "\n"), true
}

func compressBuildLog(text string) (string, bool) {
	lines := splitLines(text)
	if len(lines) < 30 {
		return "", false
	}
	keywords := []string{
		"error", "err!", "failed", "failure", "panic", "traceback", "exception", "fatal", "warning",
		"fail\t", "--- fail:", "npm err!", "pnpm", "tsc", "typescript", "vite", "dockerfile:", "#0 ",
	}
	var errors []string
	var warnings []string
	var deprecations []string
	var summaries []string
	compilingCount := 0
	downloadingCount := 0
	seen := map[int]bool{}
	keywordCounts := map[string]int{}
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		switch {
		case lower == "":
			continue
		case strings.HasPrefix(lower, "npm warn deprecated"):
			if len(deprecations) < 3 {
				deprecations = append(deprecations, line)
			}
		case strings.HasPrefix(lower, "npm warn") || strings.HasPrefix(lower, "yarn warn") ||
			strings.HasPrefix(lower, "warning") || strings.HasPrefix(lower, "[warning]"):
			if len(warnings) < 8 {
				warnings = append(warnings, line)
			}
		case strings.HasPrefix(lower, "npm err!") || strings.HasPrefix(lower, "npm error") ||
			strings.HasPrefix(lower, "yarn error") || strings.HasPrefix(lower, "error") ||
			strings.HasPrefix(lower, "[error]") || strings.HasPrefix(lower, "build failed") ||
			strings.HasPrefix(lower, "error:"):
			if len(errors) < 120 {
				errors = append(errors, line)
			}
		case strings.HasPrefix(lower, "compiling ") || strings.Contains(lower, " compiling "):
			compilingCount++
		case strings.HasPrefix(lower, "downloading ") || strings.HasPrefix(lower, "fetching "):
			downloadingCount++
		case buildSummaryRE.MatchString(lower) ||
			strings.HasPrefix(lower, "finished ") ||
			strings.HasPrefix(lower, "build success") ||
			strings.HasPrefix(lower, "successfully installed") ||
			strings.HasPrefix(lower, "successfully built") ||
			strings.Contains(lower, "packages are looking for funding"):
			summaries = append(summaries, line)
		}
		for _, keyword := range keywords {
			if strings.Contains(lower, keyword) {
				keywordCounts[keyword]++
				for j := i - 3; j <= i+3; j++ {
					if j >= 0 && j < len(lines) {
						seen[j] = true
					}
				}
			}
		}
	}
	if len(seen) == 0 && len(errors) == 0 && len(warnings) == 0 && len(deprecations) == 0 && compilingCount == 0 && downloadingCount == 0 && len(summaries) == 0 {
		return "", false
	}
	indexes := make([]int, 0, len(seen))
	for idx := range seen {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)
	if len(indexes) > 180 {
		indexes = indexes[:180]
	}
	keyNames := make([]string, 0, len(keywordCounts))
	for key := range keywordCounts {
		keyNames = append(keyNames, key)
	}
	sort.Strings(keyNames)
	var keywordSummary []string
	for _, key := range keyNames {
		keywordSummary = append(keywordSummary, fmt.Sprintf("%s=%d", key, keywordCounts[key]))
	}
	var out []string
	out = append(out, fmt.Sprintf("total_lines=%d highlighted_lines=%d compiling=%d downloading=%d keywords=%s", len(lines), len(indexes), compilingCount, downloadingCount, strings.Join(keywordSummary, ",")))
	if len(deprecations) > 0 {
		out = append(out, "deprecations:")
		out = append(out, deprecations...)
	}
	if compilingCount > 0 {
		out = append(out, fmt.Sprintf("Compiled %d packages", compilingCount))
	}
	if downloadingCount > 0 {
		out = append(out, fmt.Sprintf("Downloaded %d packages", downloadingCount))
	}
	if len(errors) > 0 {
		out = append(out, "errors:")
		out = append(out, dedupeLimit(errors, 120)...)
	}
	if len(warnings) > 0 {
		out = append(out, "warnings:")
		out = append(out, warnings...)
	}
	last := -2
	for _, idx := range indexes {
		if idx != last+1 {
			out = append(out, "...")
		}
		out = append(out, lines[idx])
		last = idx
	}
	if len(summaries) > 0 {
		out = append(out, "summary:")
		out = append(out, dedupeLimit(summaries, 12)...)
	}
	tailStart := len(lines) - 16
	if tailStart < 0 {
		tailStart = 0
	}
	out = append(out, "... tail ...")
	out = append(out, lines[tailStart:]...)
	return strings.Join(out, "\n"), true
}

func compressFileList(text string) (string, bool) {
	lines := splitLines(text)
	if len(lines) < 100 {
		return "", false
	}
	type dirStats struct {
		count int
		first []string
	}
	dirs := map[string]*dirStats{}
	extCounts := map[string]int{}
	var important []string
	var firstPaths []string
	pathLike := 0
	for _, line := range lines {
		normalized, ok := normalizePathListLine(line)
		if !ok {
			continue
		}
		pathLike++
		if len(firstPaths) < 40 {
			firstPaths = append(firstPaths, normalized)
		}
		dir := path.Dir(normalized)
		if dir == "." {
			dir = "(root)"
		}
		stats := dirs[dir]
		if stats == nil {
			stats = &dirStats{}
			dirs[dir] = stats
		}
		stats.count++
		if len(stats.first) < 2 {
			stats.first = append(stats.first, normalized)
		}
		ext := strings.ToLower(path.Ext(normalized))
		if ext == "" {
			ext = "(none)"
		}
		extCounts[ext]++
		if len(important) < 80 && isImportantPath(normalized) {
			important = append(important, normalized)
		}
	}
	if pathLike < len(lines)/2 {
		return "", false
	}
	dirNames := make([]string, 0, len(dirs))
	for dir := range dirs {
		dirNames = append(dirNames, dir)
	}
	sort.Slice(dirNames, func(i, j int) bool {
		di, dj := dirs[dirNames[i]], dirs[dirNames[j]]
		if di.count == dj.count {
			return dirNames[i] < dirNames[j]
		}
		return di.count > dj.count
	})
	extSummary := topCounts(extCounts, 12)
	out := []string{
		fmt.Sprintf("entries=%d path_like=%d dirs=%d extensions=%s", len(lines), pathLike, len(dirs), strings.Join(extSummary, ",")),
		"top_dirs:",
	}
	for i, dir := range dirNames {
		if i >= 30 {
			out = append(out, fmt.Sprintf("... %d more dirs omitted", len(dirNames)-i))
			break
		}
		stats := dirs[dir]
		out = append(out, fmt.Sprintf("%s entries=%d examples=%s", dir, stats.count, strings.Join(stats.first, " | ")))
	}
	if len(important) > 0 {
		out = append(out, "important_paths:")
		out = append(out, dedupeLimit(important, 80)...)
	}
	out = append(out, "first_paths:")
	out = append(out, firstPaths...)
	out = append(out, fmt.Sprintf("... %d original entries omitted", len(lines)-len(firstPaths)))
	return strings.Join(out, "\n"), true
}

func normalizePathListLine(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimLeft(trimmed, "│├└─-+* \t")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" || strings.HasPrefix(trimmed, "...") {
		return "", false
	}
	fields := strings.Fields(trimmed)
	if len(fields) > 0 {
		last := fields[len(fields)-1]
		if strings.Contains(last, "/") || strings.Contains(last, "\\") || strings.Contains(last, ".") {
			trimmed = last
		}
	}
	trimmed = strings.Trim(trimmed, `"'`)
	trimmed = strings.ReplaceAll(trimmed, "\\", "/")
	for strings.Contains(trimmed, "//") {
		trimmed = strings.ReplaceAll(trimmed, "//", "/")
	}
	if strings.HasPrefix(trimmed, "./") {
		trimmed = strings.TrimPrefix(trimmed, "./")
	}
	if strings.Contains(trimmed, "/") || strings.Contains(path.Base(trimmed), ".") {
		return trimmed, true
	}
	return "", false
}

func isImportantPath(p string) bool {
	base := strings.ToLower(path.Base(p))
	switch base {
	case "go.mod", "go.sum", "package.json", "pnpm-lock.yaml", "package-lock.json", "yarn.lock",
		"dockerfile", "docker-compose.yml", "compose.yml", "makefile", "tsconfig.json", "vite.config.ts",
		"requirements.txt", "pyproject.toml", "cargo.toml", "readme.md":
		return true
	}
	ext := strings.ToLower(path.Ext(p))
	switch ext {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".rs", ".java", ".kt", ".cs", ".sql", ".yaml", ".yml", ".json":
		return true
	default:
		return false
	}
}

func topCounts(counts map[string]int, limit int) []string {
	type pair struct {
		key   string
		count int
	}
	pairs := make([]pair, 0, len(counts))
	for key, count := range counts {
		pairs = append(pairs, pair{key: key, count: count})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count == pairs[j].count {
			return pairs[i].key < pairs[j].key
		}
		return pairs[i].count > pairs[j].count
	})
	if len(pairs) > limit {
		pairs = pairs[:limit]
	}
	out := make([]string, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, fmt.Sprintf("%s=%d", p.key, p.count))
	}
	return out
}

func dedupeLimit(values []string, limit int) []string {
	seen := map[string]bool{}
	out := make([]string, 0, limit)
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func splitLines(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}
