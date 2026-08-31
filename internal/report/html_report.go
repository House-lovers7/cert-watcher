package report

import (
	"fmt"
	"html/template"
	"os"
	"time"

	"cert-watcher/internal/model"
)

// htmlView はテンプレートに渡すトップレベルのビューモデル。
type htmlView struct {
	GeneratedAt string
	Summary     []summaryItem
	Rows        []rowView
}

// summaryItem はステータスごとの件数（サマリ表示用）。
type summaryItem struct {
	Status model.Status
	Count  int
	Class  string
}

// rowView は1行分の表示用データ。
type rowView struct {
	ServiceName string
	Kind        string
	Host        string
	Port        int
	SNI         string
	Status      model.Status
	StatusClass string
	DaysLeft    string
	NotBefore   string
	NotAfter    string
	Subject     string
	Issuer      string
	DNSNames    string
	Owner       string
	Vendor      string
	Notes       string
	Error       string
}

// statusClass はステータスに対応する CSS クラス名を返す。
func statusClass(s model.Status) string {
	switch s {
	case model.StatusExpired, model.StatusUrgent:
		return "s-red"
	case model.StatusCritical:
		return "s-orange"
	case model.StatusWarn:
		return "s-yellow"
	case model.StatusOK:
		return "s-green"
	case model.StatusError:
		return "s-gray"
	default:
		return ""
	}
}

// WriteHTML は結果を1ページの HTML レポートとして書き出す。
// ステータスを色分けし、上部にサマリ（件数）を表示する。
func WriteHTML(path string, results []model.CertResult, generatedAt time.Time) error {
	if err := ensureDir(path); err != nil {
		return err
	}

	// サマリ件数を集計（表示順は重大度順）。
	order := []model.Status{
		model.StatusError, model.StatusExpired, model.StatusUrgent,
		model.StatusCritical, model.StatusWarn, model.StatusOK,
	}
	counts := map[model.Status]int{}
	for _, r := range results {
		counts[r.Status]++
	}
	var summary []summaryItem
	for _, s := range order {
		summary = append(summary, summaryItem{Status: s, Count: counts[s], Class: statusClass(s)})
	}

	// 行データを表示用に変換。
	rows := make([]rowView, 0, len(results))
	for _, r := range results {
		rows = append(rows, rowView{
			ServiceName: r.ServiceName,
			Kind:        r.Kind,
			Host:        r.Host,
			Port:        r.Port,
			SNI:         r.SNI,
			Status:      r.Status,
			StatusClass: statusClass(r.Status),
			DaysLeft:    daysLeftCell(r),
			NotBefore:   zeroSafeDate(r.NotBefore),
			NotAfter:    zeroSafeDate(r.NotAfter),
			Subject:     r.Subject,
			Issuer:      r.Issuer,
			DNSNames:    joinDNS(r.DNSNames),
			Owner:       r.Owner,
			Vendor:      r.Vendor,
			Notes:       r.Notes,
			Error:       r.Error,
		})
	}

	view := htmlView{
		GeneratedAt: generatedAt.Format("2006-01-02 15:04:05 MST"),
		Summary:     summary,
		Rows:        rows,
	}

	tmpl, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("HTMLテンプレートの解析に失敗: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("HTML出力ファイルを作成できません (%s): %w", path, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, view); err != nil {
		return fmt.Errorf("HTMLレポートの生成に失敗: %w", err)
	}
	return nil
}

// zeroSafeDate は Zero 値の日付を空文字にする。
func zeroSafeDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(dateLayout)
}

// joinDNS は SAN を改行可能な表示文字列に整形する。
func joinDNS(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}

// htmlTemplate は単一ページのレポートテンプレート。
// 外部依存を避けるため CSS はインライン。
const htmlTemplate = `<!DOCTYPE html>
<html lang="ja">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>cert-watcher レポート</title>
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
  .small { color: #777; font-size: 11px; word-break: break-all; }
  /* ステータス配色 */
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
        <th>Service</th><th>Kind</th><th>Host</th><th>Port</th><th>SNI</th>
        <th>Status</th><th>Days</th><th>NotBefore</th><th>NotAfter</th>
        <th>Subject</th><th>Issuer</th><th>SAN</th>
        <th>Owner</th><th>Vendor</th><th>Notes</th><th>Error</th>
      </tr>
    </thead>
    <tbody>
      {{range .Rows}}
      <tr class="{{.StatusClass}}">
        <td>{{.ServiceName}}</td>
        <td>{{.Kind}}</td>
        <td>{{.Host}}</td>
        <td class="num">{{.Port}}</td>
        <td>{{.SNI}}</td>
        <td><span class="badge {{.StatusClass}}">{{.Status}}</span></td>
        <td class="num">{{.DaysLeft}}</td>
        <td>{{.NotBefore}}</td>
        <td>{{.NotAfter}}</td>
        <td class="small">{{.Subject}}</td>
        <td class="small">{{.Issuer}}</td>
        <td class="small">{{.DNSNames}}</td>
        <td>{{.Owner}}</td>
        <td>{{.Vendor}}</td>
        <td>{{.Notes}}</td>
        <td class="small">{{.Error}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>
</body>
</html>
`
