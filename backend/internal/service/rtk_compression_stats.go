package service

import (
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tokensaver"
)

const rtkCompressionRecentLimit = 100

var defaultRTKCompressionRecorder = NewRTKCompressionRecorder()
var defaultCodexPromptCacheRecorder = NewCodexPromptCacheRecorder()
var defaultCodexChainRecorder = NewCodexChainRecorder()

func DefaultRTKCompressionRecorder() *RTKCompressionRecorder {
	return defaultRTKCompressionRecorder
}

func DefaultCodexPromptCacheRecorder() *CodexPromptCacheRecorder {
	return defaultCodexPromptCacheRecorder
}

func DefaultCodexChainRecorder() *CodexChainRecorder {
	return defaultCodexChainRecorder
}

type RTKCompressionFilterStats struct {
	Filter     string `json:"filter"`
	Hits       int64  `json:"hits"`
	Before     int64  `json:"before"`
	After      int64  `json:"after"`
	BytesSaved int64  `json:"bytes_saved"`
}

type RTKCompressionHitStat struct {
	Path          string `json:"path"`
	Filter        string `json:"filter"`
	Before        int    `json:"before"`
	After         int    `json:"after"`
	BytesSaved    int    `json:"bytes_saved"`
	ReferenceHash string `json:"reference_hash,omitempty"`
	ReferencePath string `json:"reference_path,omitempty"`
}

type RTKCompressionMissStats struct {
	Reason string `json:"reason"`
	Count  int64  `json:"count"`
	Bytes  int64  `json:"bytes"`
}

type RTKCompressionMissStat struct {
	Path   string `json:"path"`
	Before int    `json:"before"`
	Reason string `json:"reason"`
	Sample string `json:"sample,omitempty"`
}

type RTKRequestPartStats struct {
	Part  string `json:"part"`
	Count int64  `json:"count"`
	Bytes int64  `json:"bytes"`
}

type RTKCompressionEvent struct {
	At           time.Time                `json:"at"`
	AccountID    int64                    `json:"account_id,omitempty"`
	Before       int                      `json:"before"`
	After        int                      `json:"after"`
	BytesSaved   int                      `json:"bytes_saved"`
	Hits         int                      `json:"hits"`
	Filters      []string                 `json:"filters"`
	HitStats     []RTKCompressionHitStat  `json:"hit_stats"`
	Misses       int                      `json:"misses,omitempty"`
	MissStats    []RTKCompressionMissStat `json:"miss_stats,omitempty"`
	RequestParts []RTKRequestPartStats    `json:"request_parts,omitempty"`
}

type RTKCompressionSnapshot struct {
	Enabled       bool                        `json:"enabled"`
	Strategy      string                      `json:"strategy"`
	MinBytes      int                         `json:"min_bytes"`
	MaxBytes      int                         `json:"max_bytes"`
	OutputStyle   RTKOutputStyleConfig        `json:"output_style"`
	StartedAt     time.Time                   `json:"started_at"`
	UpdatedAt     *time.Time                  `json:"updated_at,omitempty"`
	Requests      int64                       `json:"requests"`
	Hits          int64                       `json:"hits"`
	Misses        int64                       `json:"misses"`
	MissedBytes   int64                       `json:"missed_bytes"`
	Before        int64                       `json:"before"`
	After         int64                       `json:"after"`
	BytesSaved    int64                       `json:"bytes_saved"`
	SaveRatio     float64                     `json:"save_ratio"`
	ByFilter      []RTKCompressionFilterStats `json:"by_filter"`
	ByMissReason  []RTKCompressionMissStats   `json:"by_miss_reason"`
	ByRequestPart []RTKRequestPartStats       `json:"by_request_part"`
	RecentEvents  []RTKCompressionEvent       `json:"recent_events"`
	PromptCache   CodexPromptCacheSnapshot    `json:"prompt_cache"`
	CodexChain    CodexChainSnapshot          `json:"codex_chain"`
}

type RTKOutputStyleConfig struct {
	Enabled bool   `json:"enabled"`
	Level   string `json:"level"`
}

type RTKCompressionRecorder struct {
	mu        sync.RWMutex
	startedAt time.Time
	updatedAt *time.Time
	requests  int64
	hits      int64
	misses    int64
	missed    int64
	before    int64
	after     int64
	saved     int64
	byFilter  map[string]*RTKCompressionFilterStats
	byMiss    map[string]*RTKCompressionMissStats
	byPart    map[string]*RTKRequestPartStats
	recent    []RTKCompressionEvent
}

type CodexPromptCacheOutcomeStats struct {
	Outcome string `json:"outcome"`
	Count   int64  `json:"count"`
}

type CodexPromptCacheSourceStats struct {
	Source string `json:"source"`
	Count  int64  `json:"count"`
}

type CodexPromptCacheEvent struct {
	At          time.Time `json:"at"`
	AccountID   int64     `json:"account_id,omitempty"`
	Model       string    `json:"model,omitempty"`
	Outcome     string    `json:"outcome"`
	Source      string    `json:"source,omitempty"`
	KeyHash     string    `json:"key_hash,omitempty"`
	AccountType string    `json:"account_type,omitempty"`
	Compact     bool      `json:"compact,omitempty"`
}

type CodexPromptCacheSnapshot struct {
	StartedAt      time.Time                      `json:"started_at"`
	UpdatedAt      *time.Time                     `json:"updated_at,omitempty"`
	Requests       int64                          `json:"requests"`
	Injected       int64                          `json:"injected"`
	AlreadyPresent int64                          `json:"already_present"`
	Skipped        int64                          `json:"skipped"`
	ByOutcome      []CodexPromptCacheOutcomeStats `json:"by_outcome"`
	BySource       []CodexPromptCacheSourceStats  `json:"by_source"`
	RecentEvents   []CodexPromptCacheEvent        `json:"recent_events"`
}

type CodexPromptCacheRecorder struct {
	mu        sync.RWMutex
	startedAt time.Time
	updatedAt *time.Time
	requests  int64
	injected  int64
	present   int64
	skipped   int64
	byOutcome map[string]*CodexPromptCacheOutcomeStats
	bySource  map[string]*CodexPromptCacheSourceStats
	recent    []CodexPromptCacheEvent
}

type CodexChainTransportStats struct {
	Transport string `json:"transport"`
	Count     int64  `json:"count"`
}

type CodexChainEvent struct {
	At                           time.Time `json:"at"`
	AccountID                    int64     `json:"account_id,omitempty"`
	AccountType                  string    `json:"account_type,omitempty"`
	Model                        string    `json:"model,omitempty"`
	Transport                    string    `json:"transport"`
	Compact                      bool      `json:"compact,omitempty"`
	StoreFalse                   bool      `json:"store_false"`
	StreamTrue                   bool      `json:"stream_true"`
	HasPreviousResponseID        bool      `json:"has_previous_response_id"`
	PreviousResponseIDRemoved    bool      `json:"previous_response_id_removed"`
	HasPromptCacheKey            bool      `json:"has_prompt_cache_key"`
	PromptCacheKeyHash           string    `json:"prompt_cache_key_hash,omitempty"`
	HasReasoning                 bool      `json:"has_reasoning"`
	ReasoningEffort              string    `json:"reasoning_effort,omitempty"`
	HasReasoningEncryptedContent bool      `json:"has_reasoning_encrypted_content"`
	HeaderSessionIDHash          string    `json:"header_session_id_hash,omitempty"`
	HeaderConversationIDHash     string    `json:"header_conversation_id_hash,omitempty"`
	UpstreamSessionIDHash        string    `json:"upstream_session_id_hash,omitempty"`
	UpstreamConversationIDHash   string    `json:"upstream_conversation_id_hash,omitempty"`
}

type CodexChainSnapshot struct {
	StartedAt                 time.Time                  `json:"started_at"`
	UpdatedAt                 *time.Time                 `json:"updated_at,omitempty"`
	Requests                  int64                      `json:"requests"`
	WithPreviousResponseID    int64                      `json:"with_previous_response_id"`
	PreviousResponseIDRemoved int64                      `json:"previous_response_id_removed"`
	StoreFalse                int64                      `json:"store_false"`
	ReasoningEncryptedContent int64                      `json:"reasoning_encrypted_content"`
	WithPromptCacheKey        int64                      `json:"with_prompt_cache_key"`
	ByTransport               []CodexChainTransportStats `json:"by_transport"`
	RecentEvents              []CodexChainEvent          `json:"recent_events"`
}

type CodexChainRecorder struct {
	mu                        sync.RWMutex
	startedAt                 time.Time
	updatedAt                 *time.Time
	requests                  int64
	withPreviousResponseID    int64
	previousResponseIDRemoved int64
	storeFalse                int64
	reasoningEncryptedContent int64
	withPromptCacheKey        int64
	byTransport               map[string]*CodexChainTransportStats
	recent                    []CodexChainEvent
}

func NewCodexChainRecorder() *CodexChainRecorder {
	return &CodexChainRecorder{
		startedAt:   time.Now(),
		byTransport: make(map[string]*CodexChainTransportStats),
	}
}

func (r *CodexChainRecorder) Record(event CodexChainEvent) {
	if r == nil {
		return
	}
	now := time.Now()
	event.At = now
	event.AccountType = strings.TrimSpace(event.AccountType)
	event.Model = strings.TrimSpace(event.Model)
	event.Transport = strings.TrimSpace(event.Transport)
	if event.Transport == "" {
		event.Transport = "unknown"
	}
	event.PromptCacheKeyHash = strings.TrimSpace(event.PromptCacheKeyHash)
	event.ReasoningEffort = strings.TrimSpace(event.ReasoningEffort)
	event.HeaderSessionIDHash = strings.TrimSpace(event.HeaderSessionIDHash)
	event.HeaderConversationIDHash = strings.TrimSpace(event.HeaderConversationIDHash)
	event.UpstreamSessionIDHash = strings.TrimSpace(event.UpstreamSessionIDHash)
	event.UpstreamConversationIDHash = strings.TrimSpace(event.UpstreamConversationIDHash)

	r.mu.Lock()
	defer r.mu.Unlock()
	r.updatedAt = &now
	r.requests++
	if event.HasPreviousResponseID {
		r.withPreviousResponseID++
	}
	if event.PreviousResponseIDRemoved {
		r.previousResponseIDRemoved++
	}
	if event.StoreFalse {
		r.storeFalse++
	}
	if event.HasReasoningEncryptedContent {
		r.reasoningEncryptedContent++
	}
	if event.HasPromptCacheKey {
		r.withPromptCacheKey++
	}
	transportStats := r.byTransport[event.Transport]
	if transportStats == nil {
		transportStats = &CodexChainTransportStats{Transport: event.Transport}
		r.byTransport[event.Transport] = transportStats
	}
	transportStats.Count++
	r.recent = append([]CodexChainEvent{event}, r.recent...)
	if len(r.recent) > rtkCompressionRecentLimit {
		r.recent = r.recent[:rtkCompressionRecentLimit]
	}
}

func (r *CodexChainRecorder) Snapshot() CodexChainSnapshot {
	if r == nil {
		return CodexChainSnapshot{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	byTransport := make([]CodexChainTransportStats, 0, len(r.byTransport))
	for _, stats := range r.byTransport {
		byTransport = append(byTransport, *stats)
	}
	sortCodexChainTransportStats(byTransport)
	recent := append([]CodexChainEvent(nil), r.recent...)
	return CodexChainSnapshot{
		StartedAt:                 r.startedAt,
		UpdatedAt:                 r.updatedAt,
		Requests:                  r.requests,
		WithPreviousResponseID:    r.withPreviousResponseID,
		PreviousResponseIDRemoved: r.previousResponseIDRemoved,
		StoreFalse:                r.storeFalse,
		ReasoningEncryptedContent: r.reasoningEncryptedContent,
		WithPromptCacheKey:        r.withPromptCacheKey,
		ByTransport:               byTransport,
		RecentEvents:              recent,
	}
}

func NewCodexPromptCacheRecorder() *CodexPromptCacheRecorder {
	return &CodexPromptCacheRecorder{
		startedAt: time.Now(),
		byOutcome: make(map[string]*CodexPromptCacheOutcomeStats),
		bySource:  make(map[string]*CodexPromptCacheSourceStats),
	}
}

func (r *CodexPromptCacheRecorder) Record(event CodexPromptCacheEvent) {
	if r == nil || strings.TrimSpace(event.Outcome) == "" {
		return
	}
	now := time.Now()
	event.At = now
	event.Outcome = strings.TrimSpace(event.Outcome)
	event.Source = strings.TrimSpace(event.Source)
	event.Model = strings.TrimSpace(event.Model)
	event.KeyHash = strings.TrimSpace(event.KeyHash)

	r.mu.Lock()
	defer r.mu.Unlock()
	r.updatedAt = &now
	r.requests++
	switch event.Outcome {
	case "injected":
		r.injected++
	case "already_present":
		r.present++
	default:
		r.skipped++
	}
	outcomeStats := r.byOutcome[event.Outcome]
	if outcomeStats == nil {
		outcomeStats = &CodexPromptCacheOutcomeStats{Outcome: event.Outcome}
		r.byOutcome[event.Outcome] = outcomeStats
	}
	outcomeStats.Count++
	if event.Source != "" {
		sourceStats := r.bySource[event.Source]
		if sourceStats == nil {
			sourceStats = &CodexPromptCacheSourceStats{Source: event.Source}
			r.bySource[event.Source] = sourceStats
		}
		sourceStats.Count++
	}
	r.recent = append([]CodexPromptCacheEvent{event}, r.recent...)
	if len(r.recent) > rtkCompressionRecentLimit {
		r.recent = r.recent[:rtkCompressionRecentLimit]
	}
}

func (r *CodexPromptCacheRecorder) Snapshot() CodexPromptCacheSnapshot {
	if r == nil {
		return CodexPromptCacheSnapshot{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	byOutcome := make([]CodexPromptCacheOutcomeStats, 0, len(r.byOutcome))
	for _, stats := range r.byOutcome {
		byOutcome = append(byOutcome, *stats)
	}
	sortCodexPromptCacheOutcomeStats(byOutcome)
	bySource := make([]CodexPromptCacheSourceStats, 0, len(r.bySource))
	for _, stats := range r.bySource {
		bySource = append(bySource, *stats)
	}
	sortCodexPromptCacheSourceStats(bySource)
	recent := append([]CodexPromptCacheEvent(nil), r.recent...)
	return CodexPromptCacheSnapshot{
		StartedAt:      r.startedAt,
		UpdatedAt:      r.updatedAt,
		Requests:       r.requests,
		Injected:       r.injected,
		AlreadyPresent: r.present,
		Skipped:        r.skipped,
		ByOutcome:      byOutcome,
		BySource:       bySource,
		RecentEvents:   recent,
	}
}

func NewRTKCompressionRecorder() *RTKCompressionRecorder {
	return &RTKCompressionRecorder{
		startedAt: time.Now(),
		byFilter:  make(map[string]*RTKCompressionFilterStats),
		byMiss:    make(map[string]*RTKCompressionMissStats),
		byPart:    make(map[string]*RTKRequestPartStats),
	}
}

func (r *RTKCompressionRecorder) Record(accountID int64, result tokensaver.Result, requestParts ...[]RTKRequestPartStats) {
	if r == nil || (!result.Changed && len(result.Misses) == 0) {
		return
	}
	now := time.Now()
	parts := flattenRTKRequestParts(requestParts)
	event := RTKCompressionEvent{
		At:           now,
		AccountID:    accountID,
		Before:       result.Before,
		After:        result.After,
		BytesSaved:   result.Before - result.After,
		Hits:         len(result.Hits),
		Filters:      rtkCompressionFilters(result.Hits),
		HitStats:     rtkCompressionHitStats(result.Hits),
		Misses:       len(result.Misses),
		MissStats:    rtkCompressionMissStats(result.Misses),
		RequestParts: parts,
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.updatedAt = &now
	if result.Changed {
		r.requests++
		r.hits += int64(len(result.Hits))
		r.before += int64(result.Before)
		r.after += int64(result.After)
		r.saved += int64(result.Before - result.After)
	}
	for _, hit := range result.Hits {
		stats := r.byFilter[hit.Filter]
		if stats == nil {
			stats = &RTKCompressionFilterStats{Filter: hit.Filter}
			r.byFilter[hit.Filter] = stats
		}
		stats.Hits++
		stats.Before += int64(hit.Before)
		stats.After += int64(hit.After)
		stats.BytesSaved += int64(hit.Before - hit.After)
	}
	for _, miss := range result.Misses {
		reason := miss.Reason
		if reason == "" {
			reason = "unknown"
		}
		stats := r.byMiss[reason]
		if stats == nil {
			stats = &RTKCompressionMissStats{Reason: reason}
			r.byMiss[reason] = stats
		}
		stats.Count++
		stats.Bytes += int64(miss.Before)
		r.misses++
		r.missed += int64(miss.Before)
	}
	for _, part := range parts {
		name := strings.TrimSpace(part.Part)
		if name == "" {
			name = "unknown"
		}
		stats := r.byPart[name]
		if stats == nil {
			stats = &RTKRequestPartStats{Part: name}
			r.byPart[name] = stats
		}
		stats.Count += part.Count
		stats.Bytes += part.Bytes
	}
	r.recent = append([]RTKCompressionEvent{event}, r.recent...)
	if len(r.recent) > rtkCompressionRecentLimit {
		r.recent = r.recent[:rtkCompressionRecentLimit]
	}
}

func (r *RTKCompressionRecorder) Snapshot(enabled bool, strategy string, minBytes, maxBytes int, outputStyle RTKOutputStyleConfig) RTKCompressionSnapshot {
	if r == nil {
		return RTKCompressionSnapshot{Enabled: enabled, Strategy: strategy, MinBytes: minBytes, MaxBytes: maxBytes, OutputStyle: outputStyle}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	byFilter := make([]RTKCompressionFilterStats, 0, len(r.byFilter))
	for _, stats := range r.byFilter {
		byFilter = append(byFilter, *stats)
	}
	sortRTKCompressionFilterStats(byFilter)
	byMiss := make([]RTKCompressionMissStats, 0, len(r.byMiss))
	for _, stats := range r.byMiss {
		byMiss = append(byMiss, *stats)
	}
	sortRTKCompressionMissStats(byMiss)
	byPart := make([]RTKRequestPartStats, 0, len(r.byPart))
	for _, stats := range r.byPart {
		byPart = append(byPart, *stats)
	}
	sortRTKRequestPartStats(byPart)
	recent := append([]RTKCompressionEvent(nil), r.recent...)
	ratio := 0.0
	if r.before > 0 {
		ratio = float64(r.saved) / float64(r.before)
	}
	return RTKCompressionSnapshot{
		Enabled:       enabled,
		Strategy:      strategy,
		MinBytes:      minBytes,
		MaxBytes:      maxBytes,
		OutputStyle:   outputStyle,
		StartedAt:     r.startedAt,
		UpdatedAt:     r.updatedAt,
		Requests:      r.requests,
		Hits:          r.hits,
		Misses:        r.misses,
		MissedBytes:   r.missed,
		Before:        r.before,
		After:         r.after,
		BytesSaved:    r.saved,
		SaveRatio:     ratio,
		ByFilter:      byFilter,
		ByMissReason:  byMiss,
		ByRequestPart: byPart,
		RecentEvents:  recent,
		PromptCache:   DefaultCodexPromptCacheRecorder().Snapshot(),
		CodexChain:    DefaultCodexChainRecorder().Snapshot(),
	}
}

func flattenRTKRequestParts(groups [][]RTKRequestPartStats) []RTKRequestPartStats {
	if len(groups) == 0 {
		return nil
	}
	var parts []RTKRequestPartStats
	for _, group := range groups {
		for _, part := range group {
			if strings.TrimSpace(part.Part) == "" || part.Bytes <= 0 {
				continue
			}
			if part.Count <= 0 {
				part.Count = 1
			}
			parts = append(parts, part)
		}
	}
	sortRTKRequestPartStats(parts)
	return parts
}

func rtkCompressionFilters(hits []tokensaver.Hit) []string {
	if len(hits) == 0 {
		return nil
	}
	filters := make([]string, 0, len(hits))
	for _, hit := range hits {
		filters = append(filters, hit.Filter)
	}
	return filters
}

func rtkCompressionHitStats(hits []tokensaver.Hit) []RTKCompressionHitStat {
	if len(hits) == 0 {
		return nil
	}
	stats := make([]RTKCompressionHitStat, 0, len(hits))
	for _, hit := range hits {
		stats = append(stats, RTKCompressionHitStat{
			Path:          hit.Path,
			Filter:        hit.Filter,
			Before:        hit.Before,
			After:         hit.After,
			BytesSaved:    hit.Before - hit.After,
			ReferenceHash: hit.ReferenceHash,
			ReferencePath: hit.ReferencePath,
		})
	}
	return stats
}

func rtkCompressionMissStats(misses []tokensaver.Miss) []RTKCompressionMissStat {
	if len(misses) == 0 {
		return nil
	}
	stats := make([]RTKCompressionMissStat, 0, len(misses))
	for _, miss := range misses {
		stats = append(stats, RTKCompressionMissStat{
			Path:   miss.Path,
			Before: miss.Before,
			Reason: miss.Reason,
			Sample: miss.Sample,
		})
	}
	return stats
}

func sortRTKCompressionFilterStats(stats []RTKCompressionFilterStats) {
	for i := 1; i < len(stats); i++ {
		current := stats[i]
		j := i - 1
		for j >= 0 && (stats[j].BytesSaved < current.BytesSaved ||
			(stats[j].BytesSaved == current.BytesSaved && stats[j].Filter > current.Filter)) {
			stats[j+1] = stats[j]
			j--
		}
		stats[j+1] = current
	}
}

func sortRTKCompressionMissStats(stats []RTKCompressionMissStats) {
	for i := 1; i < len(stats); i++ {
		current := stats[i]
		j := i - 1
		for j >= 0 && (stats[j].Bytes < current.Bytes ||
			(stats[j].Bytes == current.Bytes && stats[j].Reason > current.Reason)) {
			stats[j+1] = stats[j]
			j--
		}
		stats[j+1] = current
	}
}

func sortRTKRequestPartStats(stats []RTKRequestPartStats) {
	for i := 1; i < len(stats); i++ {
		current := stats[i]
		j := i - 1
		for j >= 0 && (stats[j].Bytes < current.Bytes ||
			(stats[j].Bytes == current.Bytes && stats[j].Part > current.Part)) {
			stats[j+1] = stats[j]
			j--
		}
		stats[j+1] = current
	}
}

func sortCodexPromptCacheOutcomeStats(stats []CodexPromptCacheOutcomeStats) {
	for i := 1; i < len(stats); i++ {
		current := stats[i]
		j := i - 1
		for j >= 0 && (stats[j].Count < current.Count ||
			(stats[j].Count == current.Count && stats[j].Outcome > current.Outcome)) {
			stats[j+1] = stats[j]
			j--
		}
		stats[j+1] = current
	}
}

func sortCodexPromptCacheSourceStats(stats []CodexPromptCacheSourceStats) {
	for i := 1; i < len(stats); i++ {
		current := stats[i]
		j := i - 1
		for j >= 0 && (stats[j].Count < current.Count ||
			(stats[j].Count == current.Count && stats[j].Source > current.Source)) {
			stats[j+1] = stats[j]
			j--
		}
		stats[j+1] = current
	}
}

func sortCodexChainTransportStats(stats []CodexChainTransportStats) {
	for i := 1; i < len(stats); i++ {
		current := stats[i]
		j := i - 1
		for j >= 0 && (stats[j].Count < current.Count ||
			(stats[j].Count == current.Count && stats[j].Transport > current.Transport)) {
			stats[j+1] = stats[j]
			j--
		}
		stats[j+1] = current
	}
}
