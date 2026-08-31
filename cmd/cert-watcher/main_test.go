package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"cert-watcher/internal/model"
	"cert-watcher/internal/runconfig"
)

// TestSortResults はレポート用ソート（重大度→残日数）を検証する。
func TestSortResults(t *testing.T) {
	results := []model.CertResult{
		{ServiceName: "ok", Status: model.StatusOK, DaysLeft: 100},
		{ServiceName: "warn", Status: model.StatusWarn, DaysLeft: 20},
		{ServiceName: "err", Status: model.StatusError},
		{ServiceName: "expired", Status: model.StatusExpired, DaysLeft: -3},
		{ServiceName: "crit", Status: model.StatusCritical, DaysLeft: 10},
		{ServiceName: "urgent", Status: model.StatusUrgent, DaysLeft: 2},
	}

	sortResults(results)

	wantOrder := []model.Status{
		model.StatusError, model.StatusExpired, model.StatusUrgent,
		model.StatusCritical, model.StatusWarn, model.StatusOK,
	}
	for i, w := range wantOrder {
		if results[i].Status != w {
			t.Fatalf("位置 %d のステータス = %s, want %s（全体: %v）", i, results[i].Status, w, statuses(results))
		}
	}
}

// TestSortResults_DaysLeftWithinStatus は同ステータス内で残日数昇順になることを検証する。
func TestSortResults_DaysLeftWithinStatus(t *testing.T) {
	results := []model.CertResult{
		{ServiceName: "warnB", Status: model.StatusWarn, DaysLeft: 25},
		{ServiceName: "warnA", Status: model.StatusWarn, DaysLeft: 16},
		{ServiceName: "warnC", Status: model.StatusWarn, DaysLeft: 29},
	}
	sortResults(results)
	if results[0].DaysLeft != 16 || results[1].DaysLeft != 25 || results[2].DaysLeft != 29 {
		t.Fatalf("同ステータス内の残日数昇順に失敗: %v", statuses(results))
	}
}

func statuses(rs []model.CertResult) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.ServiceName + ":" + string(r.Status)
	}
	return out
}

func TestRunValidate_SeparatedInputOK(t *testing.T) {
	dir := t.TempDir()
	sites := writeCmdTestFile(t, dir, "sites.csv",
		"service_id,service_name,edge_fqdn,owner,vendor,notes\n"+
			"corp,コーポレートサイト,www.example.com,情シス,Akamai,本番\n")
	origins := writeCmdTestFile(t, dir, "origins.secure.csv",
		"service_id,origin_host,origin_sni,origin_port\n"+
			"corp,origin.example.internal,origin.example.internal,443\n")
	actions := writeCmdTestFile(t, dir, "action_status.csv",
		"service_id,kind,action_status\n"+
			"corp,Origin,確認中\n")

	code := runValidate([]string{"--sites", sites, "--origins", origins, "--actions", actions, "--check", "both"})
	if code != exitOK {
		t.Fatalf("runValidate code = %d, want %d", code, exitOK)
	}
}

func TestRunValidate_MissingOriginFails(t *testing.T) {
	dir := t.TempDir()
	sites := writeCmdTestFile(t, dir, "sites.csv",
		"service_id,service_name,edge_fqdn,owner,vendor,notes\n"+
			"corp,コーポレートサイト,www.example.com,情シス,Akamai,本番\n")

	code := runValidate([]string{"--sites", sites, "--check", "both"})
	if code != exitFailure {
		t.Fatalf("runValidate code = %d, want %d", code, exitFailure)
	}
}

func TestRunValidate_EdgeOnlyAllowsSitesOnly(t *testing.T) {
	dir := t.TempDir()
	sites := writeCmdTestFile(t, dir, "sites.csv",
		"service_id,service_name,edge_fqdn,owner,vendor,notes\n"+
			"corp,コーポレートサイト,www.example.com,情シス,Akamai,本番\n")

	code := runValidate([]string{"--sites", sites, "--check", "edge"})
	if code != exitOK {
		t.Fatalf("runValidate code = %d, want %d", code, exitOK)
	}
}

func writeCmdTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("テストファイル作成に失敗: %v", err)
	}
	return path
}

func TestApplySettings_UsesConfigWhenFlagUnset(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	timeoutStr := fs.String("timeout", "10s", "")
	warnDays := fs.Int("warn-days", 30, "")
	critDays := fs.Int("critical-days", 14, "")
	urgentDays := fs.Int("urgent-days", 7, "")
	checkStr := fs.String("check", "both", "")
	if err := fs.Parse(nil); err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	applySettings(fs, runconfig.Settings{
		Check:        "edge",
		Timeout:      "5s",
		WarnDays:     45,
		CriticalDays: 21,
		UrgentDays:   10,
	}, timeoutStr, warnDays, critDays, urgentDays, checkStr)

	if *timeoutStr != "5s" || *warnDays != 45 || *critDays != 21 || *urgentDays != 10 || *checkStr != "edge" {
		t.Fatalf("settings not applied: timeout=%s warn=%d critical=%d urgent=%d check=%s",
			*timeoutStr, *warnDays, *critDays, *urgentDays, *checkStr)
	}
}

func TestApplySettings_KeepsExplicitFlags(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	timeoutStr := fs.String("timeout", "10s", "")
	warnDays := fs.Int("warn-days", 30, "")
	critDays := fs.Int("critical-days", 14, "")
	urgentDays := fs.Int("urgent-days", 7, "")
	checkStr := fs.String("check", "both", "")
	if err := fs.Parse([]string{"--timeout", "2s", "--warn-days", "20", "--check", "origin"}); err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	applySettings(fs, runconfig.Settings{
		Check:        "edge",
		Timeout:      "5s",
		WarnDays:     45,
		CriticalDays: 21,
		UrgentDays:   10,
	}, timeoutStr, warnDays, critDays, urgentDays, checkStr)

	if *timeoutStr != "2s" || *warnDays != 20 || *checkStr != "origin" {
		t.Fatalf("explicit flags were overwritten: timeout=%s warn=%d check=%s", *timeoutStr, *warnDays, *checkStr)
	}
	if *critDays != 21 || *urgentDays != 10 {
		t.Fatalf("unset flags should be filled from config: critical=%d urgent=%d", *critDays, *urgentDays)
	}
}
