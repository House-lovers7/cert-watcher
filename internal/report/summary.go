package report

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"cert-watcher/internal/model"
)

// RunSummary は実行結果の機械可読サマリを表す。
// 接続先ホストや証明書詳細は含めない。
type RunSummary struct {
	GeneratedAt   string         `json:"generated_at"`
	Total         int            `json:"total"`
	Alert         bool           `json:"alert"`
	AlertCount    int            `json:"alert_count"`
	Counts        map[string]int `json:"counts"`
	NotifyTargets []string       `json:"notify_targets"`
}

// BuildSummary はチェック結果から実行サマリを作る。
func BuildSummary(results []model.CertResult, generatedAt time.Time) RunSummary {
	order := []model.Status{
		model.StatusError,
		model.StatusExpired,
		model.StatusUrgent,
		model.StatusCritical,
		model.StatusWarn,
		model.StatusOK,
	}

	counts := make(map[string]int, len(order))
	for _, status := range order {
		counts[string(status)] = 0
	}
	alertCount := 0
	for _, r := range results {
		counts[string(r.Status)]++
		if r.Status.IsAlert() {
			alertCount++
		}
	}

	return RunSummary{
		GeneratedAt: generatedAt.Format(time.RFC3339),
		Total:       len(results),
		Alert:       alertCount > 0,
		AlertCount:  alertCount,
		Counts:      counts,
		NotifyTargets: []string{
			string(model.StatusError),
			string(model.StatusExpired),
			string(model.StatusUrgent),
			string(model.StatusCritical),
		},
	}
}

// WriteSummaryJSON は実行サマリをJSONとして書き出す。
func WriteSummaryJSON(path string, results []model.CertResult, generatedAt time.Time) error {
	if err := ensureDir(path); err != nil {
		return err
	}

	body, err := json.MarshalIndent(BuildSummary(results, generatedAt), "", "  ")
	if err != nil {
		return fmt.Errorf("summary JSONの生成に失敗: %w", err)
	}
	body = append(body, '\n')

	if err := os.WriteFile(path, body, 0o644); err != nil {
		return fmt.Errorf("summary JSONを書き出せません (%s): %w", path, err)
	}
	return nil
}
