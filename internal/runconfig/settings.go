package runconfig

import (
	"encoding/json"
	"fmt"
	"os"
)

// Settings はCLI引数へ外出しできる実行設定を表す。
// SecretやWebhook URLはここに置かない。
type Settings struct {
	Check        string `json:"check"`
	Timeout      string `json:"timeout"`
	WarnDays     int    `json:"warn_days"`
	CriticalDays int    `json:"critical_days"`
	UrgentDays   int    `json:"urgent_days"`
}

// Load はJSON設定ファイルを読み込む。
func Load(path string) (Settings, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Settings{}, fmt.Errorf("設定ファイルを読めません (%s): %w", path, err)
	}

	var settings Settings
	if err := json.Unmarshal(body, &settings); err != nil {
		return Settings{}, fmt.Errorf("設定ファイルのJSON解析に失敗しました (%s): %w", path, err)
	}
	if err := settings.Validate(); err != nil {
		return Settings{}, err
	}
	return settings, nil
}

// Validate は設定値の基本的な整合性を確認する。
func (s Settings) Validate() error {
	if s.WarnDays < 0 {
		return fmt.Errorf("warn_days は0以上で指定してください")
	}
	if s.CriticalDays < 0 {
		return fmt.Errorf("critical_days は0以上で指定してください")
	}
	if s.UrgentDays < 0 {
		return fmt.Errorf("urgent_days は0以上で指定してください")
	}
	return nil
}
