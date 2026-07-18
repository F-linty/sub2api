package dto

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLargeIDsMarshalAsJSONStrings(t *testing.T) {
	const largeID int64 = 1192354569694740481
	const largeIDString = "1192354569694740481"

	payloads := []struct {
		name string
		v    any
		want string
	}{
		{
			name: "scalar id",
			v:    User{ID: largeID},
			want: `"id":"` + largeIDString + `"`,
		},
		{
			name: "id slice",
			v:    User{AllowedGroups: []string{largeIDString}},
			want: `"allowed_groups":["` + largeIDString + `"]`,
		},
		{
			name: "account group ids",
			v:    Account{GroupIDs: []string{largeIDString}},
			want: `"group_ids":["` + largeIDString + `"]`,
		},
		{
			name: "group model routing ids",
			v:    AdminGroup{ModelRouting: map[string][]string{"gpt-5": {largeIDString}}},
			want: `"model_routing":{"gpt-5":["` + largeIDString + `"]}`,
		},
	}

	for _, tt := range payloads {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.v)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if !strings.Contains(string(b), tt.want) {
				t.Fatalf("expected JSON to contain %s, got %s", tt.want, string(b))
			}
		})
	}
}
