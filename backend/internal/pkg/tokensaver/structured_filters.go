package tokensaver

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	gitLogMaxLines          = 200
	gitStatusMaxFiles       = 20
	shellOutputHeadLines    = 60
	shellOutputTailLines    = 25
	shellOutputHeadBytes    = 1200
	shellOutputTailBytes    = 450
	findPerDirMax           = 10
	findTotalDirMax         = 30
	pathCompactExampleMax   = 80
	treeMaxLines            = 200
	lsExtSummaryTop         = 5
	searchListPerDirMax     = 10
	searchListTotalDirMax   = 30
	readNumberedMinHitRatio = 0.7
	dedupLogMaxLines        = 2000
	smartTruncateHead       = 120
	smartTruncateTail       = 60
	smartTruncateMinLines   = 250
)

var (
	gitLogCommitRE     = regexp.MustCompile(`(?m)^[*|/\\ ]*commit [0-9a-f]{7,40}$`)
	gitStatusLineRE    = regexp.MustCompile(`(?m)^[ MADRCU?!][MADRCU?!] \S`)
	lsRowRE            = regexp.MustCompile(`^[-dlbcps][rwx-]{9}`)
	lsDateRE           = regexp.MustCompile(`\s+(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{1,2}\s+(\d{4}|\d{2}:\d{2})\s+`)
	searchListHeaderRE = regexp.MustCompile(`(?i)^(Result of search in '.+' \(total \d+ files\)|Found \d+ files?|Search results?:)\s*:?$`)
	readNumberedRE     = regexp.MustCompile(`^\s*\d+\s*[|:]\s?`)
)

func compressGitLog(text string) (string, bool) {
	if !gitLogCommitRE.MatchString(firstDetectWindow(text)) {
		return "", false
	}
	lines := splitLines(text)
	if len(lines) <= gitLogMaxLines {
		return "", false
	}
	out := append([]string{}, lines[:gitLogMaxLines]...)
	out = append(out, fmt.Sprintf("... %d more git log lines omitted", len(lines)-gitLogMaxLines))
	return strings.Join(out, "\n"), true
}

func compressGitStatus(text string) (string, bool) {
	head := firstDetectWindow(text)
	if !strings.Contains(head, "On branch ") &&
		!strings.Contains(head, "Changes not ") &&
		!strings.Contains(head, "Changes to be ") &&
		!strings.Contains(head, "Untracked files:") &&
		!gitStatusLineRE.MatchString(head) {
		return "", false
	}

	lines := splitLines(text)
	var headers []string
	var files []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if gitStatusLineRE.MatchString(line) {
			files = append(files, trimmed)
			continue
		}
		if strings.HasPrefix(trimmed, "On branch ") ||
			strings.HasPrefix(trimmed, "Your branch ") ||
			strings.HasPrefix(trimmed, "Changes ") ||
			strings.HasPrefix(trimmed, "Untracked files:") ||
			strings.HasPrefix(trimmed, "nothing to commit") {
			headers = append(headers, trimmed)
		}
	}
	if len(files) == 0 && len(headers) < 2 {
		return "", false
	}
	out := []string{fmt.Sprintf("git_status files=%d total_lines=%d", len(files), len(lines))}
	out = append(out, dedupeLimit(headers, 12)...)
	if len(files) > 0 {
		out = append(out, "files:")
		limit := gitStatusMaxFiles
		if len(files) < limit {
			limit = len(files)
		}
		out = append(out, files[:limit]...)
		if len(files) > limit {
			out = append(out, fmt.Sprintf("... %d more files omitted", len(files)-limit))
		}
	}
	return strings.Join(out, "\n"), true
}

func compressShellOutput(text string) (string, bool) {
	out, _, ok := compressShellOutputNamed(text)
	return out, ok
}

func compressShellOutputNamed(text string) (string, string, bool) {
	return compressShellOutputNamedWithOptions(text, Options{Strategy: StrategyConservative})
}

func compressShellOutputNamedWithOptions(text string, opts Options) (string, string, bool) {
	opts = normalizeOptions(opts)
	meta, output, ok := splitShellOutput(text)
	if !ok {
		return "", "", false
	}
	output = strings.TrimRight(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	if strings.TrimSpace(output) == "" {
		return strings.Join(append(meta, "Output: <empty>"), "\n"), "shell_output/empty", true
	}

	if compressed, filter, ok := compressShellOutputBodyWithOptions(output, opts); ok {
		out := append([]string{}, meta...)
		out = append(out, fmt.Sprintf("Output: compressed via %s", filter), compressed)
		return strings.Join(out, "\n"), "shell_output/" + filter, true
	}

	lines := splitLines(output)
	if len(lines) > shellOutputHeadLines+shellOutputTailLines {
		head := lines[:shellOutputHeadLines]
		tail := lines[len(lines)-shellOutputTailLines:]
		cut := len(lines) - len(head) - len(tail)
		out := append([]string{}, meta...)
		out = append(out, fmt.Sprintf("Output: kept head=%d tail=%d omitted=%d lines", len(head), len(tail), cut))
		out = append(out, head...)
		out = append(out, fmt.Sprintf("... %d shell output lines truncated", cut))
		out = append(out, tail...)
		return strings.Join(out, "\n"), "shell_output/truncate", true
	}

	if len(output) > shellOutputHeadBytes+shellOutputTailBytes {
		out := append([]string{}, meta...)
		out = append(out, fmt.Sprintf("Output: kept first=%dB last=%dB omitted=%dB", shellOutputHeadBytes, shellOutputTailBytes, len(output)-shellOutputHeadBytes-shellOutputTailBytes))
		out = append(out, output[:shellOutputHeadBytes])
		out = append(out, fmt.Sprintf("... %d shell output bytes truncated", len(output)-shellOutputHeadBytes-shellOutputTailBytes))
		out = append(out, output[len(output)-shellOutputTailBytes:])
		return strings.Join(out, "\n"), "shell_output/truncate", true
	}

	return "", "", false
}

func splitShellOutput(text string) ([]string, string, bool) {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	if !strings.HasPrefix(strings.TrimLeft(normalized, "\n\t "), "Exit code:") {
		return nil, "", false
	}
	idx := strings.Index(normalized, "\nOutput:")
	if idx < 0 {
		return nil, "", false
	}
	metaText := strings.TrimRight(normalized[:idx], "\n")
	output := normalized[idx+len("\nOutput:"):]
	output = strings.TrimPrefix(output, "\n")
	meta := splitLines(metaText)
	if len(meta) == 0 {
		return nil, "", false
	}
	if len(meta) > 6 {
		meta = meta[:6]
	}
	return meta, output, true
}

func compressShellOutputBody(output string) (string, string, bool) {
	return compressShellOutputBodyWithOptions(output, Options{Strategy: StrategyConservative})
}

func compressShellOutputBodyWithOptions(output string, opts Options) (string, string, bool) {
	opts = normalizeOptions(opts)
	candidates := compressionCandidates(opts)
	for _, candidate := range candidates {
		body, ok := candidate.fn(output)
		if ok && len(body) < len(output) {
			return body, candidate.name, true
		}
	}
	return "", "", false
}

func compressFindOutput(text string) (string, bool) {
	lines := nonEmptyLines(text)
	if len(lines) < 3 {
		return "", false
	}
	for _, line := range lines {
		if !isPathLikeOutputLine(line) {
			return "", false
		}
	}
	return compactPathLines("find", lines, findPerDirMax, findTotalDirMax), true
}

func compressTreeOutput(text string) (string, bool) {
	if !strings.Contains(text, "├──") && !strings.Contains(text, "└──") && !strings.Contains(text, "│  ") {
		return "", false
	}
	lines := splitLines(text)
	if len(lines) <= treeMaxLines {
		return "", false
	}
	out := append([]string{}, lines[:treeMaxLines]...)
	out = append(out, fmt.Sprintf("... %d more tree lines omitted", len(lines)-treeMaxLines))
	return strings.Join(out, "\n"), true
}

func compressLSOutput(text string) (string, bool) {
	lines := splitLines(text)
	if len(lines) < 5 {
		return "", false
	}
	hasTotal := false
	lsRows := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "total ") {
			hasTotal = true
		}
		if lsRowRE.MatchString(line) {
			lsRows++
		}
	}
	if !hasTotal && lsRows < 3 {
		return "", false
	}

	var dirs []string
	var files []string
	extCounts := map[string]int{}
	for _, line := range lines {
		parsed, ok := parseLSLine(line)
		if !ok || parsed.name == "." || parsed.name == ".." || isNoiseDir(parsed.name) {
			continue
		}
		if parsed.fileType == 'd' {
			dirs = append(dirs, parsed.name+"/")
			continue
		}
		if parsed.fileType == '-' || parsed.fileType == 'l' {
			files = append(files, fmt.Sprintf("%s  %s", parsed.name, humanSize(parsed.size)))
			ext := strings.ToLower(path.Ext(parsed.name))
			if ext == "" {
				ext = "no_ext"
			}
			extCounts[ext]++
		}
	}
	if len(dirs) == 0 && len(files) == 0 {
		return "", false
	}
	sort.Strings(dirs)
	sort.Strings(files)
	out := append([]string{}, dirs...)
	out = append(out, files...)
	summary := fmt.Sprintf("Summary: %d files, %d dirs", len(files), len(dirs))
	if len(extCounts) > 0 {
		summary += " (" + strings.Join(topCounts(extCounts, lsExtSummaryTop), ",") + ")"
	}
	out = append(out, summary)
	return strings.Join(out, "\n"), true
}

func compressSearchList(text string) (string, bool) {
	lines := splitLines(text)
	if len(lines) < 3 || !searchListHeaderRE.MatchString(strings.TrimSpace(lines[0])) {
		return "", false
	}
	return compactPathLines("search_list", nonEmptyLines(strings.Join(lines[1:], "\n")), searchListPerDirMax, searchListTotalDirMax), true
}

func compressReadNumbered(text string) (string, bool) {
	lines := splitLines(text)
	if len(lines) < smartTruncateMinLines {
		return "", false
	}
	nonEmpty := 0
	hits := 0
	for i, line := range lines {
		if i >= 100 {
			break
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		nonEmpty++
		if readNumberedRE.MatchString(line) {
			hits++
		}
	}
	if nonEmpty < 5 || float64(hits)/float64(nonEmpty) < readNumberedMinHitRatio {
		return "", false
	}
	return compressSmartTruncate(text)
}

func compressDedupLog(text string) (string, bool) {
	lines := splitLines(text)
	if len(lines) < 5 {
		return "", false
	}
	var out []string
	prev := ""
	runCount := 0
	blankStreak := 0
	changed := false
	flushRun := func() {
		if runCount > 1 {
			out = append(out, fmt.Sprintf("  ... (%d duplicate lines)", runCount-1))
			changed = true
		}
	}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			if blankStreak == 0 {
				out = append(out, line)
			} else {
				changed = true
			}
			blankStreak++
			flushRun()
			prev = ""
			runCount = 0
			continue
		}
		blankStreak = 0
		if line == prev {
			runCount++
			continue
		}
		flushRun()
		out = append(out, line)
		prev = line
		runCount = 1
		if len(out) >= dedupLogMaxLines {
			out = append(out, fmt.Sprintf("... truncated at %d lines", dedupLogMaxLines))
			changed = true
			break
		}
	}
	flushRun()
	if !changed {
		return "", false
	}
	return strings.Join(out, "\n"), true
}

func compressSmartTruncate(text string) (string, bool) {
	lines := splitLines(text)
	if len(lines) < smartTruncateMinLines {
		return "", false
	}
	head := lines[:smartTruncateHead]
	tail := lines[len(lines)-smartTruncateTail:]
	cut := len(lines) - len(head) - len(tail)
	out := append([]string{}, head...)
	out = append(out, fmt.Sprintf("... %d lines truncated", cut))
	out = append(out, tail...)
	return strings.Join(out, "\n"), true
}

func compactPathLines(label string, lines []string, perDirMax, totalDirMax int) string {
	type dirStats struct {
		count int
		first []string
	}
	dirs := map[string]*dirStats{}
	extCounts := map[string]int{}
	for _, line := range lines {
		p := normalizePathForStructuredFilter(line)
		dir := path.Dir(p)
		if dir == "." {
			dir = "(root)"
		}
		stats := dirs[dir]
		if stats == nil {
			stats = &dirStats{}
			dirs[dir] = stats
		}
		stats.count++
		if len(stats.first) < perDirMax {
			stats.first = append(stats.first, p)
		}
		ext := strings.ToLower(path.Ext(p))
		if ext == "" {
			ext = "(none)"
		}
		extCounts[ext]++
	}
	names := make([]string, 0, len(dirs))
	for name := range dirs {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if dirs[names[i]].count == dirs[names[j]].count {
			return names[i] < names[j]
		}
		return dirs[names[i]].count > dirs[names[j]].count
	})
	out := []string{fmt.Sprintf("%s entries=%d dirs=%d extensions=%s", label, len(lines), len(dirs), strings.Join(topCounts(extCounts, 12), ","))}
	examplesKept := 0
	for i, name := range names {
		if i >= totalDirMax {
			out = append(out, fmt.Sprintf("... %d more dirs omitted", len(names)-i))
			break
		}
		stats := dirs[name]
		out = append(out, fmt.Sprintf("%s entries=%d", name, stats.count))
		remaining := pathCompactExampleMax - examplesKept
		if remaining <= 0 {
			continue
		}
		examples := stats.first
		if len(examples) > remaining {
			examples = examples[:remaining]
		}
		out = append(out, examples...)
		examplesKept += len(examples)
	}
	if omitted := len(lines) - examplesKept; omitted > 0 {
		out = append(out, fmt.Sprintf("... %d path entries omitted", omitted))
	}
	return strings.Join(out, "\n")
}

type lsEntry struct {
	fileType byte
	size     int64
	name     string
}

func parseLSLine(line string) (lsEntry, bool) {
	m := lsDateRE.FindStringIndex(line)
	if m == nil {
		return lsEntry{}, false
	}
	name := strings.TrimSpace(line[m[1]:])
	before := strings.Fields(line[:m[0]])
	if len(before) < 4 || len(before[0]) == 0 {
		return lsEntry{}, false
	}
	var size int64
	for i := len(before) - 1; i >= 0; i-- {
		n, err := strconv.ParseInt(before[i], 10, 64)
		if err == nil {
			size = n
			break
		}
	}
	return lsEntry{fileType: before[0][0], size: size, name: name}, true
}

func humanSize(bytes int64) string {
	switch {
	case bytes >= 1_048_576:
		return fmt.Sprintf("%.1fM", float64(bytes)/1_048_576)
	case bytes >= 1024:
		return fmt.Sprintf("%.1fK", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

func firstDetectWindow(text string) string {
	if len(text) <= 1024 {
		return text
	}
	return text[:1024]
}

func nonEmptyLines(text string) []string {
	raw := splitLines(text)
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}

func isPathLikeOutputLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	if strings.Contains(trimmed, ":") && !regexp.MustCompile(`^[A-Za-z]:[\\/]`).MatchString(trimmed) {
		return false
	}
	return strings.HasPrefix(trimmed, ".") || strings.HasPrefix(trimmed, "/") || strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\")
}

func normalizePathForStructuredFilter(line string) string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.Trim(trimmed, `"'`)
	trimmed = strings.ReplaceAll(trimmed, "\\", "/")
	for strings.Contains(trimmed, "//") {
		trimmed = strings.ReplaceAll(trimmed, "//", "/")
	}
	return strings.TrimPrefix(trimmed, "./")
}

func isNoiseDir(name string) bool {
	switch strings.TrimSuffix(name, "/") {
	case "node_modules", ".git", "target", "__pycache__", ".next", "dist", "build", ".cache", ".turbo",
		".vercel", ".pytest_cache", ".mypy_cache", ".tox", ".venv", "venv", "env", "coverage", ".nyc_output",
		".DS_Store", "Thumbs.db", ".idea", ".vscode", ".vs":
		return true
	default:
		return false
	}
}
