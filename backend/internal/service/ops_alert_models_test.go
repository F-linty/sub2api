package service

import (
	"encoding/json"
	"math"
	"testing"
)

func TestOpsAlertModelsMarshalBigIntIDsAsStrings(t *testing.T) {
	tests := []struct {
		name  string
		value any
		ids   map[string]string
	}{
		{
			name:  "rule",
			value: OpsAlertRule{ID: math.MaxInt64},
			ids:   map[string]string{"id": "9223372036854775807"},
		},
		{
			name:  "event",
			value: OpsAlertEvent{ID: math.MaxInt64, RuleID: math.MaxInt64 - 1},
			ids:   map[string]string{"id": "9223372036854775807", "rule_id": "9223372036854775806"},
		},
		{
			name:  "silence",
			value: OpsAlertSilence{ID: math.MaxInt64, RuleID: math.MaxInt64 - 1},
			ids:   map[string]string{"id": "9223372036854775807", "rule_id": "9223372036854775806"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			var got map[string]any
			if err := json.Unmarshal(encoded, &got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			for field, want := range tt.ids {
				value, ok := got[field]
				if !ok {
					t.Fatalf("%s missing from %s", field, encoded)
				}
				if actual, ok := value.(string); !ok || actual != want {
					t.Fatalf("%s = %#v, want string %q", field, value, want)
				}
			}
		})
	}
}

func TestOpsAlertSilenceEntryRuleIDJSONCompatibility(t *testing.T) {
	largeID := int64(math.MaxInt64)
	entry := OpsAlertSilenceEntry{
		RuleID:       &largeID,
		UntilRFC3339: "2026-07-15T08:00:00Z",
		Reason:       "test",
	}

	encoded, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("Unmarshal map: %v", err)
	}
	if got["rule_id"] != "9223372036854775807" {
		t.Fatalf("rule_id = %#v, want string", got["rule_id"])
	}

	for _, raw := range []string{
		`{"rule_id":"9223372036854775807","until_rfc3339":"2026-07-15T08:00:00Z","reason":"test"}`,
		`{"rule_id":9223372036854775807,"until_rfc3339":"2026-07-15T08:00:00Z","reason":"test"}`,
	} {
		var decoded OpsAlertSilenceEntry
		if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
			t.Fatalf("Unmarshal %s: %v", raw, err)
		}
		if decoded.RuleID == nil || *decoded.RuleID != largeID {
			t.Fatalf("decoded RuleID = %#v, want %d", decoded.RuleID, largeID)
		}
	}
}
