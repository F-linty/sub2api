package service

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tokensaver"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

const openAITokenSaverHeader = "X-Sub2API-Token-Saver"

func (s *OpenAIGatewayService) maybeApplyOpenAITokenSaver(c *gin.Context, account *Account, body []byte) []byte {
	if s == nil || s.cfg == nil || !s.cfg.Gateway.TokenSaver.Enabled {
		return body
	}
	if c != nil && isOpenAITokenSaverDisabledByHeader(c.GetHeader(openAITokenSaverHeader)) {
		return body
	}

	body = maybeApplyOpenAIOutputStyleHint(body, s.cfg.Gateway.TokenSaver.OutputStyleEnabled, s.cfg.Gateway.TokenSaver.OutputStyleLevel)

	opts := openAITokenSaverOptions(s.cfg.Gateway.TokenSaver, openAITokenSaverReferenceScope(c, account, body))
	result, err := tokensaver.CompressJSON(body, tokensaver.Options{
		MinBytes:            opts.MinBytes,
		MaxBytes:            opts.MaxBytes,
		Strategy:            opts.Strategy,
		ReferenceScope:      opts.ReferenceScope,
		ReferenceKeepRecent: opts.ReferenceKeepRecent,
	})
	if err != nil {
		return body
	}

	accountID := int64(0)
	if account != nil {
		accountID = account.ID
	}
	if s.rtkCompressionRecorder != nil && (result.Changed || len(result.Misses) > 0) {
		s.rtkCompressionRecorder.Record(accountID, result, analyzeOpenAITokenSaverRequestParts(body))
	}

	if !result.Changed {
		return body
	}

	if s.cfg.Gateway.TokenSaver.LogEnabled {
		fields := []zap.Field{
			zap.Int("bytes_before", result.Before),
			zap.Int("bytes_after", result.After),
			zap.Int("bytes_saved", result.Before-result.After),
			zap.Int("hits", len(result.Hits)),
			zap.Strings("filters", openAITokenSaverFilters(result.Hits)),
			zap.Any("hit_stats", openAITokenSaverHitStats(result.Hits)),
		}
		if account != nil {
			fields = append(fields, zap.Int64("account_id", accountID))
		}
		logger.L().Info("OpenAI token saver compressed tool output", fields...)
	}

	return result.Body
}

func openAITokenSaverOptions(cfg config.GatewayTokenSaverConfig, referenceScope string) tokensaver.Options {
	strategy := strings.ToLower(strings.TrimSpace(cfg.Strategy))
	if strategy == "" {
		strategy = tokensaver.Strategy9Router
	}
	minBytes := cfg.MinBytes
	maxBytes := cfg.MaxBytes
	if strategy == tokensaver.Strategy9Router {
		minBytes = 500
		maxBytes = 10 << 20
	}
	return tokensaver.Options{
		MinBytes:       minBytes,
		MaxBytes:       maxBytes,
		Strategy:       strategy,
		ReferenceScope: referenceScope,
	}
}

const openAIOutputStyleMarker = "[sub2api output style]"

func maybeApplyOpenAIOutputStyleHint(body []byte, enabled bool, level string) []byte {
	if !enabled || len(strings.TrimSpace(string(body))) == 0 {
		return body
	}
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return body
	}
	hint := openAIOutputStyleHint(level)
	if hint == "" {
		return body
	}
	if instructions, ok := root["instructions"].(string); ok {
		if strings.Contains(instructions, openAIOutputStyleMarker) {
			return body
		}
		root["instructions"] = strings.TrimSpace(instructions + "\n\n" + hint)
		encoded, err := json.Marshal(root)
		if err != nil {
			return body
		}
		return encoded
	}
	if messages, ok := root["messages"].([]any); ok {
		for _, raw := range messages {
			msg, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			role, _ := msg["role"].(string)
			if role != "system" && role != "developer" {
				continue
			}
			content, ok := msg["content"].(string)
			if !ok || strings.Contains(content, openAIOutputStyleMarker) {
				return body
			}
			msg["content"] = strings.TrimSpace(content + "\n\n" + hint)
			encoded, err := json.Marshal(root)
			if err != nil {
				return body
			}
			return encoded
		}
		root["messages"] = append([]any{map[string]any{"role": "system", "content": hint}}, messages...)
		encoded, err := json.Marshal(root)
		if err != nil {
			return body
		}
		return encoded
	}
	return body
}

func openAIOutputStyleHint(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "terse":
		return openAIOutputStyleMarker + " Prefer terse final answers. Avoid restating routine tool details unless they change the outcome. Keep code explanations short."
	default:
		return openAIOutputStyleMarker + " Prefer concise final answers. Summarize routine tool output instead of repeating it, while preserving important errors, decisions, and next steps."
	}
}

func analyzeOpenAITokenSaverRequestParts(body []byte) []RTKRequestPartStats {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &top); err != nil {
		return []RTKRequestPartStats{{Part: "body/unparsed", Count: 1, Bytes: int64(len(body))}}
	}
	parts := make(map[string]*RTKRequestPartStats)
	accounted := 0
	add := func(part string, count int64, bytes int) {
		if bytes <= 0 {
			return
		}
		part = strings.TrimSpace(part)
		if part == "" {
			part = "unknown"
		}
		if count <= 0 {
			count = 1
		}
		stats := parts[part]
		if stats == nil {
			stats = &RTKRequestPartStats{Part: part}
			parts[part] = stats
		}
		stats.Count += count
		stats.Bytes += int64(bytes)
	}
	for key, raw := range top {
		accounted += len(raw)
		switch key {
		case "input":
			analyzeOpenAIInputPart(raw, add)
		case "tools":
			add("tools/schema", 1, len(raw))
		case "instructions":
			add("instructions", 1, len(raw))
		case "messages":
			add("messages", 1, len(raw))
		case "reasoning":
			add("reasoning", 1, len(raw))
		case "metadata":
			add("metadata", 1, len(raw))
		case "model", "stream", "store", "prompt_cache_key", "include", "parallel_tool_calls", "service_tier",
			"temperature", "top_p", "max_output_tokens", "max_completion_tokens":
			add("request/options", 1, len(raw))
		default:
			add("other/top_level", 1, len(raw))
		}
	}
	if overhead := len(trimmed) - accounted; overhead > 0 {
		add("json/overhead", 1, overhead)
	}
	result := make([]RTKRequestPartStats, 0, len(parts))
	for _, stats := range parts {
		result = append(result, *stats)
	}
	sortRTKRequestPartStats(result)
	return result
}

func analyzeOpenAIInputPart(raw json.RawMessage, add func(string, int64, int)) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		add("input/raw", 1, len(raw))
		return
	}
	accounted := 0
	for _, itemRaw := range items {
		accounted += len(itemRaw)
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(itemRaw, &obj); err != nil {
			add("input/other", 1, len(itemRaw))
			continue
		}
		itemType := jsonRawString(obj["type"])
		switch itemType {
		case "function_call_output":
			outputRaw, hasOutput := obj["output"]
			if !hasOutput {
				add("input/tool_output", 1, len(itemRaw))
				continue
			}
			add("input/tool_output", 1, len(outputRaw))
			if metaBytes := len(itemRaw) - len(outputRaw); metaBytes > 0 {
				add("input/tool_output_meta", 1, metaBytes)
			}
		case "function_call":
			add("input/tool_call", 1, len(itemRaw))
		case "message":
			add("input/message", 1, len(itemRaw))
		case "reasoning":
			add("input/reasoning", 1, len(itemRaw))
		default:
			add("input/other", 1, len(itemRaw))
		}
	}
	if overhead := len(raw) - accounted; overhead > 0 {
		add("input/overhead", 1, overhead)
	}
}

func jsonRawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func openAITokenSaverReferenceScope(c *gin.Context, account *Account, body []byte) string {
	if account == nil || account.ID == 0 {
		return ""
	}
	values := gjson.GetManyBytes(body,
		"model",
		"prompt_cache_key",
		"session_id",
		"conversation_id",
		"metadata.user_id",
		"client_metadata.x-codex-window-id",
		"client_metadata.x-codex-installation-id",
	)
	model := strings.TrimSpace(values[0].String())
	source, key := openAITokenSaverScopeKey(c, values[1:])
	if key == "" {
		return ""
	}
	return strings.Join([]string{
		"account=" + strconv.FormatInt(account.ID, 10),
		"model=" + normalizeCodexModel(model),
		source + "=" + hashSensitiveValueForLog(key),
	}, "|")
}

func openAITokenSaverScopeKey(c *gin.Context, bodyValues []gjson.Result) (string, string) {
	if len(bodyValues) > 0 {
		if promptCacheKey := strings.TrimSpace(bodyValues[0].String()); promptCacheKey != "" {
			return "prompt_cache_key", promptCacheKey
		}
	}
	if c != nil {
		if sessionID := strings.TrimSpace(c.GetHeader("session_id")); sessionID != "" {
			return "header_session_id", sessionID
		}
		if conversationID := strings.TrimSpace(c.GetHeader("conversation_id")); conversationID != "" {
			return "header_conversation_id", conversationID
		}
	}
	if len(bodyValues) > 1 {
		if sessionID := strings.TrimSpace(bodyValues[1].String()); sessionID != "" {
			return "body_session_id", sessionID
		}
	}
	if len(bodyValues) > 2 {
		if conversationID := strings.TrimSpace(bodyValues[2].String()); conversationID != "" {
			return "body_conversation_id", conversationID
		}
	}
	if len(bodyValues) > 3 {
		if userID := strings.TrimSpace(bodyValues[3].String()); userID != "" {
			if parsed := ParseMetadataUserID(userID); parsed != nil && strings.TrimSpace(parsed.SessionID) != "" {
				return "metadata_user_session", parsed.SessionID
			}
			return "metadata_user_id", userID
		}
	}
	if len(bodyValues) > 4 {
		if windowID := strings.TrimSpace(bodyValues[4].String()); windowID != "" {
			return "client_window_id", windowID
		}
	}
	if len(bodyValues) > 5 {
		if installationID := strings.TrimSpace(bodyValues[5].String()); installationID != "" {
			return "client_installation_id", installationID
		}
	}
	return "", ""
}

func isOpenAITokenSaverDisabledByHeader(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "off", "false", "0", "disable", "disabled":
		return true
	default:
		return false
	}
}

func openAITokenSaverFilters(hits []tokensaver.Hit) []string {
	if len(hits) == 0 {
		return nil
	}
	filters := make([]string, 0, len(hits))
	for _, hit := range hits {
		filters = append(filters, hit.Filter)
	}
	return filters
}

type openAITokenSaverHitStat struct {
	Path          string `json:"path"`
	Filter        string `json:"filter"`
	Before        int    `json:"before"`
	After         int    `json:"after"`
	BytesSaved    int    `json:"bytes_saved"`
	ReferenceHash string `json:"reference_hash,omitempty"`
	ReferencePath string `json:"reference_path,omitempty"`
}

func openAITokenSaverHitStats(hits []tokensaver.Hit) []openAITokenSaverHitStat {
	if len(hits) == 0 {
		return nil
	}
	stats := make([]openAITokenSaverHitStat, 0, len(hits))
	for _, hit := range hits {
		stats = append(stats, openAITokenSaverHitStat{
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
