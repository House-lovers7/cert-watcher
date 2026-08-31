package report

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildSummary(t *testing.T) {
	generatedAt := time.Date(2026, 6, 23, 8, 0, 0, 0, time.UTC)
	got := BuildSummary(publicReportTestResults(), generatedAt)

	if got.GeneratedAt != "2026-06-23T08:00:00Z" {
		t.Fatalf("GeneratedAt = %s", got.GeneratedAt)
	}
	if got.Total != 2 || got.AlertCount != 2 || !got.Alert {
		t.Fatalf("summary counts mismatch: %+v", got)
	}
	if got.Counts["ERROR"] != 1 || got.Counts["CRITICAL"] != 1 || got.Counts["OK"] != 0 {
		t.Fatalf("status counts mismatch: %+v", got.Counts)
	}
}

func TestWriteSummaryJSON_MasksPrivateFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "summary.json")
	if err := WriteSummaryJSON(path, publicReportTestResults(), time.Date(2026, 6, 23, 8, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("WriteSummaryJSON failed: %v", err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("summary JSON read failed: %v", err)
	}
	got := string(body)

	mustContain(t, got, `"total": 2`)
	mustContain(t, got, `"alert": true`)
	mustContain(t, got, `"ERROR": 1`)
	mustNotContain(t, got, "ec-origin.example.internal")
	mustNotContain(t, got, "origin.example.internal")
	mustNotContain(t, got, "dial tcp")
}
