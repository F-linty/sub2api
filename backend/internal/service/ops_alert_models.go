package service

import (
	"encoding/json"
	"strconv"
	"time"
)

// Ops alert rule/event models.
//
// NOTE: These are admin-facing DTOs and intentionally keep JSON naming aligned
// with the existing ops dashboard frontend (backup style).

const (
	OpsAlertStatusFiring         = "firing"
	OpsAlertStatusResolved       = "resolved"
	OpsAlertStatusManualResolved = "manual_resolved"
)

type OpsAlertRule struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`

	Enabled  bool   `json:"enabled"`
	Severity string `json:"severity"`

	MetricType string  `json:"metric_type"`
	Operator   string  `json:"operator"`
	Threshold  float64 `json:"threshold"`

	WindowMinutes    int `json:"window_minutes"`
	SustainedMinutes int `json:"sustained_minutes"`
	CooldownMinutes  int `json:"cooldown_minutes"`

	NotifyEmail bool `json:"notify_email"`

	Filters map[string]any `json:"filters,omitempty"`

	LastTriggeredAt *time.Time `json:"last_triggered_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type OpsAlertEvent struct {
	ID       int64  `json:"id"`
	RuleID   int64  `json:"rule_id"`
	Severity string `json:"severity"`
	Status   string `json:"status"`

	Title       string `json:"title"`
	Description string `json:"description"`

	MetricValue    *float64 `json:"metric_value,omitempty"`
	ThresholdValue *float64 `json:"threshold_value,omitempty"`

	Dimensions map[string]any `json:"dimensions,omitempty"`

	FiredAt    time.Time  `json:"fired_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`

	EmailSent bool      `json:"email_sent"`
	CreatedAt time.Time `json:"created_at"`
}

type OpsAlertSilence struct {
	ID int64 `json:"id"`

	RuleID   int64   `json:"rule_id"`
	Platform string  `json:"platform"`
	GroupID  *int64  `json:"group_id,omitempty"`
	Region   *string `json:"region,omitempty"`

	Until  time.Time `json:"until"`
	Reason string    `json:"reason"`

	CreatedBy *int64    `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// MarshalJSON keeps database BIGINT identifiers exact in JavaScript clients.
// CockroachDB may generate IDs that exceed Number.MAX_SAFE_INTEGER.
func (r OpsAlertRule) MarshalJSON() ([]byte, error) {
	type alias OpsAlertRule
	return json.Marshal(struct {
		ID string `json:"id"`
		*alias
	}{
		ID:    strconv.FormatInt(r.ID, 10),
		alias: (*alias)(&r),
	})
}

func (e OpsAlertEvent) MarshalJSON() ([]byte, error) {
	type alias OpsAlertEvent
	return json.Marshal(struct {
		ID     string `json:"id"`
		RuleID string `json:"rule_id"`
		*alias
	}{
		ID:     strconv.FormatInt(e.ID, 10),
		RuleID: strconv.FormatInt(e.RuleID, 10),
		alias:  (*alias)(&e),
	})
}

func (s OpsAlertSilence) MarshalJSON() ([]byte, error) {
	type alias OpsAlertSilence
	return json.Marshal(struct {
		ID     string `json:"id"`
		RuleID string `json:"rule_id"`
		*alias
	}{
		ID:     strconv.FormatInt(s.ID, 10),
		RuleID: strconv.FormatInt(s.RuleID, 10),
		alias:  (*alias)(&s),
	})
}

type OpsAlertEventFilter struct {
	Limit int

	// Cursor pagination (descending by fired_at, then id).
	BeforeFiredAt *time.Time
	BeforeID      *int64

	// Optional filters.
	Status    string
	Severity  string
	EmailSent *bool

	StartTime *time.Time
	EndTime   *time.Time

	// Dimensions filters (best-effort).
	Platform string
	GroupID  *int64
}
