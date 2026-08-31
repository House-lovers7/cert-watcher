package notify

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cert-watcher/internal/model"
)

// sample は各ステータスを1件ずつ含む結果セット。
func sample() []model.CertResult {
	return []model.CertResult{
		{ServiceName: "a", Kind: "Edge", Status: model.StatusOK, DaysLeft: 100},
		{ServiceName: "b", Kind: "Edge", Status: model.StatusWarn, DaysLeft: 20},
		{ServiceName: "c", Kind: "Edge", Status: model.StatusCritical, DaysLeft: 10},
		{ServiceName: "d", Kind: "Origin", Status: model.StatusUrgent, DaysLeft: 3},
		{ServiceName: "e", Kind: "Edge", Status: model.StatusExpired, DaysLeft: -1},
		{ServiceName: "f", Kind: "Origin", Status: model.StatusError},
	}
}

// TestFilterAlerts は ERROR/EXPIRED/URGENT/CRITICAL のみ抽出し、
// WARN/OK を除外することを検証する。
func TestFilterAlerts(t *testing.T) {
	got := FilterAlerts(sample())
	if len(got) != 4 {
		t.Fatalf("抽出件数 = %d, want 4", len(got))
	}
	for _, r := range got {
		if !AlertStatuses[r.Status] {
			t.Errorf("対象外ステータスが含まれる: %s", r.Status)
		}
	}
}

// TestSendTeams_SkipWhenNoAlerts は通知対象0件で送信スキップ(sent=false,err=nil)を検証する。
func TestSendTeams_SkipWhenNoAlerts(t *testing.T) {
	results := []model.CertResult{
		{ServiceName: "a", Status: model.StatusOK},
		{ServiceName: "b", Status: model.StatusWarn},
	}
	sent, err := SendTeams("http://example.invalid", results, http.DefaultClient)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if sent {
		t.Error("通知対象0件なのに sent=true")
	}
}

// TestSendTeams_PostsMessageCard はアラートありの時に MessageCard が POST されることを検証する。
func TestSendTeams_PostsMessageCard(t *testing.T) {
	var received map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("1"))
	}))
	defer srv.Close()

	sent, err := SendTeams(srv.URL, sample(), srv.Client())
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if !sent {
		t.Fatal("sent=false（送信されていない）")
	}
	if received["@type"] != "MessageCard" {
		t.Errorf("@type = %v, want MessageCard", received["@type"])
	}
	// 件数サマリの文字列が含まれること。
	if txt, _ := received["text"].(string); !strings.Contains(txt, "ERROR") || !strings.Contains(txt, "EXPIRED") {
		t.Errorf("text にサマリが含まれない: %q", txt)
	}
}

// TestBuildMessageCard_MasksPrivateDetails はTeams本文にOrigin詳細や詳細エラーを出さないことを検証する。
func TestBuildMessageCard_MasksPrivateDetails(t *testing.T) {
	alerts := []model.CertResult{
		{
			ServiceName: "ECサイト",
			Kind:        "Origin",
			Host:        "ec-origin.example.internal",
			SNI:         "origin.example.internal",
			Status:      model.StatusError,
			Owner:       "EC担当",
			Vendor:      "Akamai",
			Error:       "TLS接続に失敗 (ec-origin.example.internal:443): dial tcp",
		},
	}

	body, err := json.Marshal(buildMessageCard(alerts))
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}
	got := string(body)

	if !strings.Contains(got, "管理者確認が必要") {
		t.Fatalf("ERROR時の対応概要が含まれていません: %s", got)
	}
	for _, leaked := range []string{
		"ec-origin.example.internal",
		"origin.example.internal",
		"TLS接続に失敗",
		"dial tcp",
	} {
		if strings.Contains(got, leaked) {
			t.Fatalf("Teams通知に出すべきでない詳細 %q が含まれています: %s", leaked, got)
		}
	}
}

// TestSendTeams_Non2xxIsError は非2xx応答がエラーになることを検証する。
func TestSendTeams_Non2xxIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	_, err := SendTeams(srv.URL, sample(), srv.Client())
	if err == nil {
		t.Fatal("非2xx応答でエラーになるべきだが nil")
	}
}
