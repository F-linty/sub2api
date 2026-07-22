package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tokensaver"
)

func TestRTKCompressionRecorderSnapshot(t *testing.T) {
	recorder := NewRTKCompressionRecorder()
	recorder.Record(42, tokensaver.Result{
		Changed: true,
		Before:  1000,
		After:   350,
		Hits: []tokensaver.Hit{
			{Path: "$.input[0].output", Filter: "file_list", Before: 700, After: 200},
			{Path: "$.input[1].output", Filter: "build_log", Before: 300, After: 150},
		},
	}, []RTKRequestPartStats{
		{Part: "input/tool_output", Count: 2, Bytes: 800},
		{Part: "tools/schema", Count: 1, Bytes: 120},
	})

	snapshot := recorder.Snapshot(true, 2048, 524288)
	if !snapshot.Enabled || snapshot.MinBytes != 2048 || snapshot.MaxBytes != 524288 {
		t.Fatalf("unexpected config fields: %#v", snapshot)
	}
	if snapshot.Requests != 1 || snapshot.Hits != 2 || snapshot.Before != 1000 ||
		snapshot.After != 350 || snapshot.BytesSaved != 650 {
		t.Fatalf("unexpected totals: %#v", snapshot)
	}
	if len(snapshot.ByFilter) != 2 || snapshot.ByFilter[0].Filter != "file_list" {
		t.Fatalf("unexpected filter stats: %#v", snapshot.ByFilter)
	}
	if len(snapshot.RecentEvents) != 1 || snapshot.RecentEvents[0].AccountID != 42 {
		t.Fatalf("unexpected recent events: %#v", snapshot.RecentEvents)
	}
	if len(snapshot.RecentEvents[0].HitStats) != 2 {
		t.Fatalf("expected hit stats in recent event: %#v", snapshot.RecentEvents[0])
	}
	if len(snapshot.ByRequestPart) != 2 || snapshot.ByRequestPart[0].Part != "input/tool_output" {
		t.Fatalf("unexpected request part stats: %#v", snapshot.ByRequestPart)
	}
	if len(snapshot.RecentEvents[0].RequestParts) != 2 {
		t.Fatalf("expected request parts in recent event: %#v", snapshot.RecentEvents[0])
	}
}

func TestRTKCompressionRecorderTracksMissesSeparately(t *testing.T) {
	recorder := NewRTKCompressionRecorder()
	recorder.Record(42, tokensaver.Result{
		Changed: false,
		Before:  5000,
		After:   5000,
		Misses: []tokensaver.Miss{
			{Path: "$.input[0].output", Before: 5000, Reason: "no_filter", Sample: "opaque output"},
		},
	})

	snapshot := recorder.Snapshot(true, 2048, 524288)
	if snapshot.Requests != 0 || snapshot.Hits != 0 || snapshot.Before != 0 || snapshot.BytesSaved != 0 {
		t.Fatalf("miss-only requests should not affect compression totals: %#v", snapshot)
	}
	if snapshot.Misses != 1 || snapshot.MissedBytes != 5000 {
		t.Fatalf("unexpected miss totals: %#v", snapshot)
	}
	if len(snapshot.ByMissReason) != 1 || snapshot.ByMissReason[0].Reason != "no_filter" {
		t.Fatalf("unexpected miss reason stats: %#v", snapshot.ByMissReason)
	}
	if len(snapshot.RecentEvents) != 1 || len(snapshot.RecentEvents[0].MissStats) != 1 {
		t.Fatalf("expected miss details in recent event: %#v", snapshot.RecentEvents)
	}
}
