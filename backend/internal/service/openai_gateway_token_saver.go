package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tokensaver"
	"github.com/gin-gonic/gin"
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

	result, err := tokensaver.CompressJSON(body, tokensaver.Options{
		MinBytes: s.cfg.Gateway.TokenSaver.MinBytes,
		MaxBytes: s.cfg.Gateway.TokenSaver.MaxBytes,
	})
	if err != nil {
		return body
	}

	accountID := int64(0)
	if account != nil {
		accountID = account.ID
	}
	if s.rtkCompressionRecorder != nil && (result.Changed || len(result.Misses) > 0) {
		s.rtkCompressionRecorder.Record(accountID, result)
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
	Path       string `json:"path"`
	Filter     string `json:"filter"`
	Before     int    `json:"before"`
	After      int    `json:"after"`
	BytesSaved int    `json:"bytes_saved"`
}

func openAITokenSaverHitStats(hits []tokensaver.Hit) []openAITokenSaverHitStat {
	if len(hits) == 0 {
		return nil
	}
	stats := make([]openAITokenSaverHitStat, 0, len(hits))
	for _, hit := range hits {
		stats = append(stats, openAITokenSaverHitStat{
			Path:       hit.Path,
			Filter:     hit.Filter,
			Before:     hit.Before,
			After:      hit.After,
			BytesSaved: hit.Before - hit.After,
		})
	}
	return stats
}
