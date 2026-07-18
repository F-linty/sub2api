package service

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Ops settings models stored in DB `settings` table (JSON blobs).

type OpsEmailNotificationConfig struct {
	Alert  OpsEmailAlertConfig  `json:"alert"`
	Report OpsEmailReportConfig `json:"report"`
}

type OpsEmailAlertConfig struct {
	Enabled               bool     `json:"enabled"`
	Recipients            []string `json:"recipients"`
	MinSeverity           string   `json:"min_severity"`
	RateLimitPerHour      int      `json:"rate_limit_per_hour"`
	BatchingWindowSeconds int      `json:"batching_window_seconds"`
	IncludeResolvedAlerts bool     `json:"include_resolved_alerts"`
}

type OpsEmailReportConfig struct {
	Enabled                         bool     `json:"enabled"`
	Recipients                      []string `json:"recipients"`
	DailySummaryEnabled             bool     `json:"daily_summary_enabled"`
	DailySummarySchedule            string   `json:"daily_summary_schedule"`
	WeeklySummaryEnabled            bool     `json:"weekly_summary_enabled"`
	WeeklySummarySchedule           string   `json:"weekly_summary_schedule"`
	ErrorDigestEnabled              bool     `json:"error_digest_enabled"`
	ErrorDigestSchedule             string   `json:"error_digest_schedule"`
	ErrorDigestMinCount             int      `json:"error_digest_min_count"`
	AccountHealthEnabled            bool     `json:"account_health_enabled"`
	AccountHealthSchedule           string   `json:"account_health_schedule"`
	AccountHealthErrorRateThreshold float64  `json:"account_health_error_rate_threshold"`
}

// OpsEmailNotificationConfigUpdateRequest allows partial updates, while the
// frontend can still send the full config shape.
type OpsEmailNotificationConfigUpdateRequest struct {
	Alert  *OpsEmailAlertConfig  `json:"alert"`
	Report *OpsEmailReportConfig `json:"report"`
}

type OpsDistributedLockSettings struct {
	Enabled    bool   `json:"enabled"`
	Key        string `json:"key"`
	TTLSeconds int    `json:"ttl_seconds"`
}

type OpsAlertSilenceEntry struct {
	RuleID     *int64   `json:"rule_id,omitempty"`
	Severities []string `json:"severities,omitempty"`

	UntilRFC3339 string `json:"until_rfc3339"`
	Reason       string `json:"reason"`
}

func (e OpsAlertSilenceEntry) MarshalJSON() ([]byte, error) {
	type payload struct {
		RuleID       *string  `json:"rule_id,omitempty"`
		Severities   []string `json:"severities,omitempty"`
		UntilRFC3339 string   `json:"until_rfc3339"`
		Reason       string   `json:"reason"`
	}
	var ruleID *string
	if e.RuleID != nil {
		value := strconv.FormatInt(*e.RuleID, 10)
		ruleID = &value
	}
	return json.Marshal(payload{
		RuleID:       ruleID,
		Severities:   e.Severities,
		UntilRFC3339: e.UntilRFC3339,
		Reason:       e.Reason,
	})
}

func (e *OpsAlertSilenceEntry) UnmarshalJSON(raw []byte) error {
	type payload struct {
		RuleID       json.RawMessage `json:"rule_id"`
		Severities   []string        `json:"severities,omitempty"`
		UntilRFC3339 string          `json:"until_rfc3339"`
		Reason       string          `json:"reason"`
	}
	var p payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}

	e.RuleID = nil
	if len(p.RuleID) > 0 && strings.TrimSpace(string(p.RuleID)) != "null" {
		value := strings.TrimSpace(string(p.RuleID))
		if unquoted, err := strconv.Unquote(value); err == nil {
			value = unquoted
		}
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		e.RuleID = &parsed
	}
	e.Severities = p.Severities
	e.UntilRFC3339 = p.UntilRFC3339
	e.Reason = p.Reason
	return nil
}

type OpsAlertSilencingSettings struct {
	Enabled bool `json:"enabled"`

	GlobalUntilRFC3339 string `json:"global_until_rfc3339"`
	GlobalReason       string `json:"global_reason"`

	Entries []OpsAlertSilenceEntry `json:"entries,omitempty"`
}

type OpsMetricThresholds struct {
	SLAPercentMin               *float64 `json:"sla_percent_min,omitempty"`                 // SLA低于此值变红
	TTFTp99MsMax                *float64 `json:"ttft_p99_ms_max,omitempty"`                 // TTFT P99高于此值变红
	RequestErrorRatePercentMax  *float64 `json:"request_error_rate_percent_max,omitempty"`  // 请求错误率高于此值变红
	UpstreamErrorRatePercentMax *float64 `json:"upstream_error_rate_percent_max,omitempty"` // 上游错误率高于此值变红
}

type OpsRuntimeLogConfig struct {
	Level           string         `json:"level"`
	EnableSampling  bool           `json:"enable_sampling"`
	SamplingInitial int            `json:"sampling_initial"`
	SamplingNext    int            `json:"sampling_thereafter"`
	Caller          bool           `json:"caller"`
	StacktraceLevel string         `json:"stacktrace_level"`
	RetentionDays   int            `json:"retention_days"`
	Source          string         `json:"source,omitempty"`
	UpdatedAt       string         `json:"updated_at,omitempty"`
	UpdatedByUserID int64          `json:"updated_by_user_id,omitempty"`
	Extra           map[string]any `json:"extra,omitempty"`
}

type OpsAlertRuntimeSettings struct {
	EvaluationIntervalSeconds int `json:"evaluation_interval_seconds"`

	DistributedLock OpsDistributedLockSettings `json:"distributed_lock"`
	Silencing       OpsAlertSilencingSettings  `json:"silencing"`
	Thresholds      OpsMetricThresholds        `json:"thresholds"` // 指标阈值配置
}

// OpsAdvancedSettings stores advanced ops configuration (data retention, aggregation).
type OpsAdvancedSettings struct {
	DataRetention                   OpsDataRetentionSettings               `json:"data_retention"`
	Aggregation                     OpsAggregationSettings                 `json:"aggregation"`
	OpenAIAccountQuotaAutoPause     OpsOpenAIAccountQuotaAutoPauseSettings `json:"openai_account_quota_auto_pause"`
	IgnoreCountTokensErrors         bool                                   `json:"ignore_count_tokens_errors"`
	IgnoreContextCanceled           bool                                   `json:"ignore_context_canceled"`
	IgnoreNoAvailableAccounts       bool                                   `json:"ignore_no_available_accounts"`
	IgnoreInvalidApiKeyErrors       bool                                   `json:"ignore_invalid_api_key_errors"`
	IgnoreInsufficientBalanceErrors bool                                   `json:"ignore_insufficient_balance_errors"`
	DisplayOpenAITokenStats         bool                                   `json:"display_openai_token_stats"`
	DisplayAlertEvents              bool                                   `json:"display_alert_events"`
	AutoRefreshEnabled              bool                                   `json:"auto_refresh_enabled"`
	AutoRefreshIntervalSec          int                                    `json:"auto_refresh_interval_seconds"`
}

type OpsOpenAIAccountQuotaAutoPauseSettings struct {
	DefaultThreshold5h float64 `json:"default_threshold_5h"`
	DefaultThreshold7d float64 `json:"default_threshold_7d"`
}

type OpsDataRetentionSettings struct {
	CleanupEnabled             bool   `json:"cleanup_enabled"`
	CleanupSchedule            string `json:"cleanup_schedule"`
	ErrorLogRetentionDays      int    `json:"error_log_retention_days"`
	MinuteMetricsRetentionDays int    `json:"minute_metrics_retention_days"`
	HourlyMetricsRetentionDays int    `json:"hourly_metrics_retention_days"`
}

type OpsAggregationSettings struct {
	AggregationEnabled bool `json:"aggregation_enabled"`
}
