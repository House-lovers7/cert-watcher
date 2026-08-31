package report

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteBundle_WritesLatestAndHistory(t *testing.T) {
	root := t.TempDir()
	generatedAt := time.Date(2026, 6, 23, 8, 0, 0, 0, time.UTC)

	paths, err := WriteBundle(root, publicReportTestResults(), generatedAt)
	if err != nil {
		t.Fatalf("WriteBundle failed: %v", err)
	}

	wantPaths := []string{
		filepath.Join(root, "03_Report", "private", "latest.csv"),
		filepath.Join(root, "03_Report", "private", "latest.html"),
		filepath.Join(root, "03_Report", "public", "latest.csv"),
		filepath.Join(root, "03_Report", "public", "latest.html"),
		filepath.Join(root, "03_Report", "summary.json"),
		filepath.Join(root, "04_History", "2026-06-23", "private.csv"),
		filepath.Join(root, "04_History", "2026-06-23", "private.html"),
		filepath.Join(root, "04_History", "2026-06-23", "public.csv"),
		filepath.Join(root, "04_History", "2026-06-23", "public.html"),
		filepath.Join(root, "04_History", "2026-06-23", "summary.json"),
	}
	for _, path := range wantPaths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected bundle output missing: %s: %v", path, err)
		}
	}

	if paths.PublicLatestCSV != filepath.Join(root, "03_Report", "public", "latest.csv") {
		t.Fatalf("PublicLatestCSV = %s", paths.PublicLatestCSV)
	}
	if paths.LatestSummaryJSON != filepath.Join(root, "03_Report", "summary.json") {
		t.Fatalf("LatestSummaryJSON = %s", paths.LatestSummaryJSON)
	}

	publicCSV, err := os.ReadFile(paths.PublicLatestCSV)
	if err != nil {
		t.Fatalf("public latest read failed: %v", err)
	}
	mustNotContain(t, string(publicCSV), "ec-origin.example.internal")
	mustContain(t, string(publicCSV), "接続または証明書取得に失敗")
}
