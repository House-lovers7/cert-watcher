package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cert-watcher/internal/model"
)

func TestWritePublicCSV_MasksPrivateFields(t *testing.T) {
	results := publicReportTestResults()
	path := filepath.Join(t.TempDir(), "public.csv")

	if err := WritePublicCSV(path, results); err != nil {
		t.Fatalf("WritePublicCSV failed: %v", err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("public CSV read failed: %v", err)
	}
	got := string(body)

	mustContain(t, got, "service_name,kind,status,days_left,not_after,owner,vendor,action_status,action_owner,vendor_ticket,due_date,comment,error_summary")
	mustContain(t, got, "ECサイト,Origin,ERROR")
	mustContain(t, got, "更新依頼済,EC担当,TICKET-1,2026-07-01,ベンダー確認中")
	mustContain(t, got, "接続または証明書取得に失敗")
	mustNotContain(t, got, "ec-origin.example.internal")
	mustNotContain(t, got, "origin.example.internal")
	mustNotContain(t, got, "Issuer Inc")
	mustNotContain(t, got, "dial tcp")
}

func TestWritePublicHTML_MasksPrivateFields(t *testing.T) {
	results := publicReportTestResults()
	path := filepath.Join(t.TempDir(), "public.html")

	if err := WritePublicHTML(path, results, time.Date(2026, 6, 23, 8, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("WritePublicHTML failed: %v", err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("public HTML read failed: %v", err)
	}
	got := string(body)

	mustContain(t, got, "ECサイト")
	mustContain(t, got, "更新依頼済")
	mustContain(t, got, "TICKET-1")
	mustContain(t, got, "接続または証明書取得に失敗")
	mustNotContain(t, got, "ec-origin.example.internal")
	mustNotContain(t, got, "origin.example.internal")
	mustNotContain(t, got, "Issuer Inc")
	mustNotContain(t, got, "dial tcp")
}

func publicReportTestResults() []model.CertResult {
	return []model.CertResult{
		{
			ServiceName:  "ECサイト",
			Kind:         "Origin",
			Host:         "ec-origin.example.internal",
			Port:         443,
			SNI:          "origin.example.internal",
			Status:       model.StatusError,
			Subject:      "CN=origin.example.internal",
			Issuer:       "CN=Issuer Inc",
			DNSNames:     []string{"origin.example.internal"},
			Owner:        "EC担当",
			Vendor:       "Akamai",
			Error:        "dial tcp: lookup ec-origin.example.internal failed",
			ActionStatus: "更新依頼済",
			ActionOwner:  "EC担当",
			VendorTicket: "TICKET-1",
			DueDate:      "2026-07-01",
			Comment:      "ベンダー確認中",
		},
		{
			ServiceName: "コーポレートサイト",
			Kind:        "Edge",
			Status:      model.StatusCritical,
			DaysLeft:    10,
			NotAfter:    time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
			Owner:       "情シス",
			Vendor:      "Akamai",
		},
	}
}

func mustContain(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("出力に %q が含まれていません:\n%s", want, got)
	}
}

func mustNotContain(t *testing.T, got, want string) {
	t.Helper()
	if strings.Contains(got, want) {
		t.Fatalf("出力に含めるべきでない %q が含まれています:\n%s", want, got)
	}
}
