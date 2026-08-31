// Package report はチェック結果を CSV / HTML レポートとして出力する。
package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cert-watcher/internal/model"
)

// dateLayout はレポート上の日付フォーマット。
const dateLayout = "2006-01-02"

// csvHeader は出力CSVのヘッダ行。
var csvHeader = []string{
	"service_name", "kind", "host", "port", "sni", "status", "days_left",
	"not_before", "not_after", "subject", "issuer", "dns_names",
	"owner", "vendor", "notes", "error",
}

// WriteCSV は結果スライスを指定パスへ CSV として書き出す。
// 出力先ディレクトリが無ければ作成する。
func WriteCSV(path string, results []model.CertResult) error {
	if err := ensureDir(path); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("CSV出力ファイルを作成できません (%s): %w", path, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write(csvHeader); err != nil {
		return fmt.Errorf("CSVヘッダの書き込みに失敗: %w", err)
	}

	for _, r := range results {
		row := []string{
			r.ServiceName,
			r.Kind,
			r.Host,
			strconv.Itoa(r.Port),
			r.SNI,
			string(r.Status),
			daysLeftCell(r),
			dateCell(r.NotBefore),
			dateCell(r.NotAfter),
			r.Subject,
			r.Issuer,
			strings.Join(r.DNSNames, ";"),
			r.Owner,
			r.Vendor,
			r.Notes,
			r.Error,
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("CSV行の書き込みに失敗: %w", err)
		}
	}

	if err := w.Error(); err != nil {
		return fmt.Errorf("CSV書き込み中にエラー: %w", err)
	}
	return nil
}

// daysLeftCell はエラー行では残日数を空欄にする。
func daysLeftCell(r model.CertResult) string {
	if r.Status == model.StatusError {
		return ""
	}
	return strconv.Itoa(r.DaysLeft)
}

// dateCell はエラー等で Zero 値となった日付を空欄として出力する。
func dateCell(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(dateLayout)
}

// ensureDir は出力ファイルの親ディレクトリを必要に応じて作成する。
func ensureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("出力ディレクトリを作成できません (%s): %w", dir, err)
	}
	return nil
}
