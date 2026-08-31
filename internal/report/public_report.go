package report

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"os"
	"time"

	"cert-watcher/internal/model"
)

var publicCSVHeader = []string{
	"service_name", "kind", "status", "days_left", "not_after",
	"owner", "vendor", "action_status", "action_owner", "vendor_ticket",
	"due_date", "comment", "error_summary",
}

// WritePublicCSV は非エンジニア向けのCSVレポートを書き出す。
// Originの接続先、SNI、証明書詳細、詳細エラーは出力しない。
func WritePublicCSV(path string, results []model.CertResult) error {
	if err := ensureDir(path); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("public CSV出力ファイルを作成できません (%s): %w", path, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write(publicCSVHeader); err != nil {
		return fmt.Errorf("public CSVヘッダの書き込みに失敗: %w", err)
	}

	for _, r := range results {
		row := []string{
			r.ServiceName,
			r.Kind,
			string(r.Status),
			daysLeftCell(r),
			dateCell(r.NotAfter),
			r.Owner,
			r.Vendor,
			r.ActionStatus,
			r.ActionOwner,
			r.VendorTicket,
			r.DueDate,
			r.Comment,
			publicErrorSummary(r),
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("public CSV行の書き込みに失敗: %w", err)
		}
	}

	if err := w.Error(); err != nil {
		return fmt.Errorf("public CSV書き込み中にエラー: %w", err)
	}
	return nil
}

// WritePublicHTML は非エンジニア向けのHTMLレポートを書き出す。
func WritePublicHTML(path string, results []model.CertResult, generatedAt time.Time) error {
	if err := ensureDir(path); err != nil {
		return err
	}

	view := publicHTMLView{
		GeneratedAt: generatedAt.Format("2006-01-02 15:04:05 MST"),
		Summary:     buildSummary(results),
		Rows:        buildPublicRows(results),
	}

	tmpl, err := template.New("public-report").Parse(publicHTMLTemplate)
	if err != nil {
		return fmt.Errorf("public HTMLテンプレートの解析に失敗: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("public HTML出力ファイルを作成できません (%s): %w", path, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, view); err != nil {
		return fmt.Errorf("public HTMLレポートの生成に失敗: %w", err)
	}
	return nil
}

type publicHTMLView struct {
	GeneratedAt string
	Summary     []summaryItem
	Rows        []publicRowView
}

type publicRowView struct {
	ServiceName  string
	Kind         string
	Status       model.Status
	StatusClass  string
	DaysLeft     string
	NotAfter     string
	Owner        string
	Vendor       string
	ActionStatus string
	ActionOwner  string
	VendorTicket string
	DueDate      string
	Comment      string
	ErrorSummary string
}

func buildSummary(results []model.CertResult) []summaryItem {
	order := []model.Status{
		model.StatusError, model.StatusExpired, model.StatusUrgent,
		model.StatusCritical, model.StatusWarn, model.StatusOK,
	}
	counts := map[model.Status]int{}
	for _, r := range results {
		counts[r.Status]++
	}
	summary := make([]summaryItem, 0, len(order))
	for _, s := range order {
		summary = append(summary, summaryItem{Status: s, Count: counts[s], Class: statusClass(s)})
	}
	return summary
}

func buildPublicRows(results []model.CertResult) []publicRowView {
	rows := make([]publicRowView, 0, len(results))
	for _, r := range results {
		rows = append(rows, publicRowView{
			ServiceName:  r.ServiceName,
			Kind:         r.Kind,
			Status:       r.Status,
			StatusClass:  statusClass(r.Status),
			DaysLeft:     daysLeftCell(r),
			NotAfter:     zeroSafeDate(r.NotAfter),
			Owner:        r.Owner,
			Vendor:       r.Vendor,
			ActionStatus: r.ActionStatus,
			ActionOwner:  r.ActionOwner,
			VendorTicket: r.VendorTicket,
			DueDate:      r.DueDate,
			Comment:      r.Comment,
			ErrorSummary: publicErrorSummary(r),
		})
	}
	return rows
}

func publicErrorSummary(r model.CertResult) string {
	if r.Status != model.StatusError {
		return ""
	}
	return "接続または証明書取得に失敗"
}

const publicHTMLTemplate = `<!DOCTYPE html>
<html lang="ja">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>cert-watcher public レポート</title>
<style>
  :root { font-family: -apple-system, "Segoe UI", "Hiragino Sans", Meiryo, sans-serif; }
  body { margin: 24px; color: #1a1a1a; background: #fafafa; }
  h1 { font-size: 20px; margin-bottom: 4px; }
  .meta { color: #666; font-size: 13px; margin-bottom: 16px; }
  .summary { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 20px; }
  .summary .chip {
    padding: 6px 12px; border-radius: 6px; font-size: 13px; font-weight: 600;
    border: 1px solid rgba(0,0,0,.1);
  }
  table { border-collapse: collapse; width: 100%; background: #fff; font-size: 13px; }
  th, td { border: 1px solid #e0e0e0; padding: 6px 8px; text-align: left; vertical-align: top; }
  th { background: #f0f0f0; position: sticky; top: 0; }
  td.num { text-align: right; }
  .badge { padding: 2px 8px; border-radius: 4px; font-weight: 700; font-size: 12px; white-space: nowrap; }
  .s-red    { background: #fdecea; color: #b71c1c; }
  .s-orange { background: #fff3e0; color: #e65100; }
  .s-yellow { background: #fffde7; color: #9e7700; }
  .s-green  { background: #e8f5e9; color: #1b5e20; }
  .s-gray   { background: #eceff1; color: #455a64; }
</style>
</head>
<body>
  <h1>cert-watcher 証明書期限レポート</h1>
  <div class="meta">生成日時: {{.GeneratedAt}}</div>

  <div class="summary">
    {{range .Summary}}<span class="chip {{.Class}}">{{.Status}}: {{.Count}}</span>{{end}}
  </div>

  <table>
    <thead>
      <tr>
        <th>Service</th><th>Kind</th><th>Status</th><th>Days</th>
        <th>NotAfter</th><th>Owner</th><th>Vendor</th><th>Action</th>
        <th>Action Owner</th><th>Ticket</th><th>Due</th><th>Comment</th><th>Error</th>
      </tr>
    </thead>
    <tbody>
      {{range .Rows}}
      <tr class="{{.StatusClass}}">
        <td>{{.ServiceName}}</td>
        <td>{{.Kind}}</td>
        <td><span class="badge {{.StatusClass}}">{{.Status}}</span></td>
        <td class="num">{{.DaysLeft}}</td>
        <td>{{.NotAfter}}</td>
        <td>{{.Owner}}</td>
        <td>{{.Vendor}}</td>
        <td>{{.ActionStatus}}</td>
        <td>{{.ActionOwner}}</td>
        <td>{{.VendorTicket}}</td>
        <td>{{.DueDate}}</td>
        <td>{{.Comment}}</td>
        <td>{{.ErrorSummary}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>
</body>
</html>
`
