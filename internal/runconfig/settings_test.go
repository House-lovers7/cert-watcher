package runconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	path := writeSettingsFile(t, `{
  "check": "edge",
  "timeout": "5s",
  "warn_days": 45,
  "critical_days": 21,
  "urgent_days": 10
}`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if got.Check != "edge" || got.Timeout != "5s" ||
		got.WarnDays != 45 || got.CriticalDays != 21 || got.UrgentDays != 10 {
		t.Fatalf("settings mismatch: %+v", got)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	_, err := Load(writeSettingsFile(t, `{`))
	if err == nil {
		t.Fatal("invalid JSONでエラーになるべき")
	}
}

func TestLoad_InvalidThreshold(t *testing.T) {
	_, err := Load(writeSettingsFile(t, `{"warn_days": -1}`))
	if err == nil {
		t.Fatal("負のしきい値でエラーになるべき")
	}
}

func writeSettingsFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("設定ファイル作成に失敗: %v", err)
	}
	return path
}
