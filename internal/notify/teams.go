// Package notify はチェック結果の通知（Microsoft Teams など）を担う。
package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"cert-watcher/internal/model"
)

// AlertStatuses は Teams 通知の対象とするステータス。
// 構想に従い ERROR / EXPIRED / URGENT / CRITICAL を通知し、WARN 以下は送らない。
var AlertStatuses = map[model.Status]bool{
	model.StatusError:    true,
	model.StatusExpired:  true,
	model.StatusUrgent:   true,
	model.StatusCritical: true,
}

// messageCard は Teams 従来型 Incoming Webhook（Office 365 Connector）が
// 受け付ける MessageCard 形式の最小ペイロード。
//
// 注意: Teams の従来型コネクタは廃止方向で、現在は Power Automate「Workflows」が
// 推奨されている。Workflows は Adaptive Card 形式を期待するため、その場合は
// buildWorkflowsPayload 相当を別途用意して差し替える想定（本実装は MessageCard）。
type messageCard struct {
	Type       string          `json:"@type"`
	Context    string          `json:"@context"`
	ThemeColor string          `json:"themeColor"`
	Summary    string          `json:"summary"`
	Title      string          `json:"title"`
	Text       string          `json:"text"`
	Sections   []messageCardMS `json:"sections,omitempty"`
}

// messageCardMS は MessageCard の1セクション。
type messageCardMS struct {
	ActivityTitle string            `json:"activityTitle,omitempty"`
	Facts         []messageCardFact `json:"facts,omitempty"`
	Markdown      bool              `json:"markdown"`
	Text          string            `json:"text,omitempty"`
}

// messageCardFact は key/value の表示項目。
type messageCardFact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// FilterAlerts は通知対象（ERROR/EXPIRED/URGENT/CRITICAL）の結果だけを抽出する。
func FilterAlerts(results []model.CertResult) []model.CertResult {
	var out []model.CertResult
	for _, r := range results {
		if AlertStatuses[r.Status] {
			out = append(out, r)
		}
	}
	return out
}

// SendTeams は通知対象を MessageCard 形式で Teams Webhook へ送信する。
// 通知対象が0件のときは送信せず (sent=false, err=nil) を返す。
// 送信失敗はエラーを返すが、呼び出し側は全体を止めない想定。
func SendTeams(webhookURL string, results []model.CertResult, client *http.Client) (sent bool, err error) {
	alerts := FilterAlerts(results)
	if len(alerts) == 0 {
		return false, nil // 対象なし → スキップ
	}

	card := buildMessageCard(alerts)
	body, err := json.Marshal(card)
	if err != nil {
		return false, fmt.Errorf("Teams通知ペイロードの生成に失敗: %w", err)
	}

	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("Teams通知リクエストの生成に失敗: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("Teams通知の送信に失敗: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return false, fmt.Errorf("Teams通知が失敗ステータスを返しました: %s (%s)",
			resp.Status, strings.TrimSpace(string(snippet)))
	}
	return true, nil
}

// buildMessageCard は通知対象から MessageCard を組み立てる。
func buildMessageCard(alerts []model.CertResult) messageCard {
	// 重大度→残日数で並べ替えて、危険なものを上に。
	sort.SliceStable(alerts, func(i, j int) bool {
		if a, b := model.StatusRank(alerts[i].Status), model.StatusRank(alerts[j].Status); a != b {
			return a < b
		}
		return alerts[i].DaysLeft < alerts[j].DaysLeft
	})

	// ステータス別件数を集計。
	counts := map[model.Status]int{}
	for _, a := range alerts {
		counts[a.Status]++
	}
	summaryLine := fmt.Sprintf("ERROR: %d / EXPIRED: %d / URGENT: %d / CRITICAL: %d",
		counts[model.StatusError], counts[model.StatusExpired],
		counts[model.StatusUrgent], counts[model.StatusCritical])

	// 各対象を facts として列挙。
	facts := make([]messageCardFact, 0, len(alerts))
	for _, a := range alerts {
		name := fmt.Sprintf("[%s] %s (%s)", a.Status, a.ServiceName, a.Kind)
		facts = append(facts, messageCardFact{Name: name, Value: notificationSummary(a)})
	}

	return messageCard{
		Type:       "MessageCard",
		Context:    "http://schema.org/extensions",
		ThemeColor: "D32F2F", // 赤系（警告）
		Summary:    "証明書期限アラート",
		Title:      "cert-watcher 証明書期限アラート",
		Text:       summaryLine,
		Sections: []messageCardMS{
			{
				ActivityTitle: "対応が必要な証明書",
				Markdown:      true,
				Facts:         facts,
			},
		},
	}
}

func notificationSummary(r model.CertResult) string {
	owner := valueOrDash(r.Owner)
	vendor := valueOrDash(r.Vendor)
	if r.Status == model.StatusError {
		return fmt.Sprintf("対応: 管理者確認が必要です / 担当: %s / ベンダー: %s", owner, vendor)
	}
	return fmt.Sprintf("残 %d 日 / 期限日: %s / 担当: %s / ベンダー: %s",
		r.DaysLeft, dateOrDash(r.NotAfter), owner, vendor)
}

func valueOrDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

// dateOrDash は Zero 値の日付を "-" にして返す。
func dateOrDash(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02")
}
