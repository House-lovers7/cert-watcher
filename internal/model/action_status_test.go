package model

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadActionStatuses_Basic(t *testing.T) {
	csv := "service_id,kind,action_status,action_owner,vendor_ticket,due_date,comment\n" +
		"ec,Origin,更新依頼済,EC担当,TICKET-1,2026-07-01,ベンダー確認中\n"

	got, err := LoadActionStatuses(writeTempActionCSV(t, csv))
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("件数 = %d, want 1", len(got))
	}
	w := got[0]
	if w.ServiceID != "ec" || w.Kind != "Origin" || w.ActionStatus != "更新依頼済" ||
		w.ActionOwner != "EC担当" || w.VendorTicket != "TICKET-1" {
		t.Fatalf("マッピング不一致: %+v", w)
	}
}

func TestLoadActionStatuses_Duplicate(t *testing.T) {
	csv := "service_id,kind,action_status\n" +
		"ec,Origin,確認中\n" +
		"ec,origin,更新依頼済\n"
	_, err := LoadActionStatuses(writeTempActionCSV(t, csv))
	if err == nil {
		t.Fatal("service_id + kind 重複でエラーになるべき")
	}
}

func TestApplyActionStatuses(t *testing.T) {
	results := []CertResult{
		{ServiceID: "ec", ServiceName: "ECサイト", Kind: "Origin", Status: StatusError},
		{ServiceID: "ec", ServiceName: "ECサイト", Kind: "Edge", Status: StatusOK},
	}
	actions := []ActionStatus{
		{
			ServiceID:    "ec",
			Kind:         "Origin",
			ActionStatus: "更新依頼済",
			ActionOwner:  "EC担当",
			VendorTicket: "TICKET-1",
			DueDate:      "2026-07-01",
			Comment:      "ベンダー確認中",
		},
	}

	ApplyActionStatuses(results, actions)

	if results[0].ActionStatus != "更新依頼済" || results[0].VendorTicket != "TICKET-1" {
		t.Fatalf("対応管理が反映されていません: %+v", results[0])
	}
	if results[1].ActionStatus != "" {
		t.Fatalf("一致しないkindへ対応管理が反映されています: %+v", results[1])
	}
}

func writeTempActionCSV(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "action_status.csv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("一時CSVの作成に失敗: %v", err)
	}
	return path
}
