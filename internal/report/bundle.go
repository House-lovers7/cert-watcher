package report

import (
	"fmt"
	"path/filepath"
	"time"

	"cert-watcher/internal/model"
)

// BundlePaths は推奨フォルダ構成で生成されるレポートの出力先を表す。
type BundlePaths struct {
	PrivateLatestCSV   string
	PrivateLatestHTML  string
	PublicLatestCSV    string
	PublicLatestHTML   string
	LatestSummaryJSON  string
	PrivateHistoryCSV  string
	PrivateHistoryHTML string
	PublicHistoryCSV   string
	PublicHistoryHTML  string
	HistorySummaryJSON string
}

// WriteBundle は SharePoint 配置を想定した推奨フォルダ構成へ
// private/public の latest と日別履歴をまとめて書き出す。
func WriteBundle(root string, results []model.CertResult, generatedAt time.Time) (BundlePaths, error) {
	if root == "" {
		return BundlePaths{}, fmt.Errorf("report root が空です")
	}

	date := generatedAt.Format(dateLayout)
	paths := BundlePaths{
		PrivateLatestCSV:   filepath.Join(root, "03_Report", "private", "latest.csv"),
		PrivateLatestHTML:  filepath.Join(root, "03_Report", "private", "latest.html"),
		PublicLatestCSV:    filepath.Join(root, "03_Report", "public", "latest.csv"),
		PublicLatestHTML:   filepath.Join(root, "03_Report", "public", "latest.html"),
		LatestSummaryJSON:  filepath.Join(root, "03_Report", "summary.json"),
		PrivateHistoryCSV:  filepath.Join(root, "04_History", date, "private.csv"),
		PrivateHistoryHTML: filepath.Join(root, "04_History", date, "private.html"),
		PublicHistoryCSV:   filepath.Join(root, "04_History", date, "public.csv"),
		PublicHistoryHTML:  filepath.Join(root, "04_History", date, "public.html"),
		HistorySummaryJSON: filepath.Join(root, "04_History", date, "summary.json"),
	}

	writes := []struct {
		path string
		fn   func(string) error
	}{
		{paths.PrivateLatestCSV, func(path string) error { return WriteCSV(path, results) }},
		{paths.PrivateLatestHTML, func(path string) error { return WriteHTML(path, results, generatedAt) }},
		{paths.PublicLatestCSV, func(path string) error { return WritePublicCSV(path, results) }},
		{paths.PublicLatestHTML, func(path string) error { return WritePublicHTML(path, results, generatedAt) }},
		{paths.LatestSummaryJSON, func(path string) error { return WriteSummaryJSON(path, results, generatedAt) }},
		{paths.PrivateHistoryCSV, func(path string) error { return WriteCSV(path, results) }},
		{paths.PrivateHistoryHTML, func(path string) error { return WriteHTML(path, results, generatedAt) }},
		{paths.PublicHistoryCSV, func(path string) error { return WritePublicCSV(path, results) }},
		{paths.PublicHistoryHTML, func(path string) error { return WritePublicHTML(path, results, generatedAt) }},
		{paths.HistorySummaryJSON, func(path string) error { return WriteSummaryJSON(path, results, generatedAt) }},
	}

	for _, w := range writes {
		if err := w.fn(w.path); err != nil {
			return BundlePaths{}, err
		}
	}
	return paths, nil
}
