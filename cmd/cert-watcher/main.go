// cert-watcher は Akamai 構成サイトの Edge / Origin 証明書の有効期限を監視する CLI。
//
// 使い方:
//
//	cert-watcher check --input config/targets.csv --output output/report.csv \
//	    [--html output/report.html] [--timeout 10s] \
//	    [--warn-days 30] [--critical-days 14] [--urgent-days 7]
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"cert-watcher/internal/checker"
	"cert-watcher/internal/model"
	"cert-watcher/internal/notify"
	"cert-watcher/internal/report"
	"cert-watcher/internal/runconfig"
)

// 終了コード。
const (
	exitOK      = 0 // 正常（アラート無し）
	exitAlert   = 1 // EXPIRED / URGENT / CRITICAL / ERROR を検知
	exitFailure = 2 // 実行失敗（引数不正・CSV読込失敗など）
)

func main() {
	// サブコマンド未指定 or help は使い方を表示。
	if len(os.Args) < 2 {
		usage()
		os.Exit(exitFailure)
	}

	switch os.Args[1] {
	case "check":
		os.Exit(runCheck(os.Args[2:]))
	case "validate":
		os.Exit(runValidate(os.Args[2:]))
	case "-h", "--help", "help":
		usage()
		os.Exit(exitOK)
	default:
		fmt.Fprintf(os.Stderr, "不明なサブコマンド: %q\n\n", os.Args[1])
		usage()
		os.Exit(exitFailure)
	}
}

// usage はトップレベルの使い方を表示する。
func usage() {
	fmt.Fprint(os.Stderr, `cert-watcher - Akamai Edge / Origin 証明書期限監視ツール

使い方:
  cert-watcher check --input <CSV> --output <CSV> [オプション]
  cert-watcher check --sites <CSV> --origins <CSV> --output <CSV> [オプション]
  cert-watcher check --sites <CSV> --origins <CSV> --report-root <DIR> [オプション]
  cert-watcher validate --sites <CSV> --origins <CSV> [--actions <CSV>] [オプション]

オプション:
  --input         入力CSV（従来互換。--sites/--origins とは排他）
  --sites        サイト一覧CSV（sites.csv。--input の代替）
  --origins      Origin情報CSV（origins.secure.csv。origin/both では必須）
  --output        詳細CSVレポート出力先（--report-root なしでは必須）
  --html          HTMLレポート出力先（任意）
  --public-output 非エンジニア向けCSVレポート出力先（任意）
  --public-html   非エンジニア向けHTMLレポート出力先（任意）
  --summary-json  実行サマリJSON出力先（任意）
  --report-root   推奨フォルダ構成へ latest/history をまとめて出力するルート（任意）
  --actions       対応管理CSV（action_status.csv。任意）
  --config        実行設定JSON（しきい値・タイムアウト。Secretは入れない）
  --check         チェック対象 edge / origin / both（既定 both）
  --teams-webhook Teams Incoming Webhook URL（指定時のみ通知）
  --timeout       接続タイムアウト（既定 10s, 例: 5s, 1m）
  --warn-days     WARN しきい値（既定 30）
  --critical-days CRITICAL しきい値（既定 14）
  --urgent-days   URGENT しきい値（既定 7）

例:
  cert-watcher check --input config/targets.csv --output output/report.csv \
      --html output/report.html --timeout 10s
`)
}

// runValidate はCSV台帳の形式・重複・紐付けを検証する。TLS接続は行わない。
func runValidate(args []string) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	var (
		input    = fs.String("input", "", "入力CSVのパス（従来互換。--sites/--origins とは排他）")
		sites    = fs.String("sites", "", "サイト一覧CSVのパス（--input の代替）")
		origins  = fs.String("origins", "", "Origin情報CSVのパス（origin/both では必須）")
		actions  = fs.String("actions", "", "対応管理CSVのパス（任意）")
		checkStr = fs.String("check", "both", "検証対象 edge / origin / both")
	)
	if err := fs.Parse(args); err != nil {
		return exitFailure
	}
	if *input == "" && *sites == "" {
		fmt.Fprintln(os.Stderr, "エラー: --input または --sites のどちらかは必須です")
		fs.Usage()
		return exitFailure
	}
	if *input != "" && (*sites != "" || *origins != "") {
		fmt.Fprintln(os.Stderr, "エラー: --input と --sites/--origins は同時に指定できません")
		fs.Usage()
		return exitFailure
	}
	mode, err := checker.ParseCheckMode(*checkStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
		return exitFailure
	}

	targets, err := loadTargets(*input, *sites, *origins, mode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
		return exitFailure
	}
	fmt.Fprintf(os.Stderr, "入力台帳OK: 監視対象 %d 件（検証対象: %s）\n", len(targets), mode)

	if *actions != "" {
		actionStatuses, err := model.LoadActionStatuses(*actions)
		if err != nil {
			fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
			return exitFailure
		}
		fmt.Fprintf(os.Stderr, "対応管理OK: %d 件\n", len(actionStatuses))
	}

	fmt.Fprintln(os.Stdout, "validate OK")
	return exitOK
}

// runCheck は check サブコマンドの本体。終了コードを返す。
func runCheck(args []string) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	var (
		input      = fs.String("input", "", "入力CSVのパス（従来互換。--sites/--origins とは排他）")
		sites      = fs.String("sites", "", "サイト一覧CSVのパス（--input の代替）")
		origins    = fs.String("origins", "", "Origin情報CSVのパス（origin/both では必須）")
		output     = fs.String("output", "", "詳細CSVレポートのパス（--report-root なしでは必須）")
		htmlOut    = fs.String("html", "", "HTMLレポート出力先（任意）")
		publicOut  = fs.String("public-output", "", "非エンジニア向けCSVレポート出力先（任意）")
		publicHTML = fs.String("public-html", "", "非エンジニア向けHTMLレポート出力先（任意）")
		summaryOut = fs.String("summary-json", "", "実行サマリJSON出力先（任意）")
		reportRoot = fs.String("report-root", "", "推奨フォルダ構成へ latest/history をまとめて出力するルート（任意）")
		actions    = fs.String("actions", "", "対応管理CSVのパス（任意）")
		configPath = fs.String("config", "", "実行設定JSONのパス（任意）")
		timeoutStr = fs.String("timeout", "10s", "接続タイムアウト（例: 10s, 1m）")
		warnDays   = fs.Int("warn-days", 30, "WARN しきい値（日数）")
		critDays   = fs.Int("critical-days", 14, "CRITICAL しきい値（日数）")
		urgentDays = fs.Int("urgent-days", 7, "URGENT しきい値（日数）")
		checkStr   = fs.String("check", "both", "チェック対象 edge / origin / both")
		teamsHook  = fs.String("teams-webhook", "", "Teams Incoming Webhook URL（指定時のみ通知）")
	)
	if err := fs.Parse(args); err != nil {
		return exitFailure
	}

	// 必須フラグの検証。
	if *output == "" && *reportRoot == "" {
		fmt.Fprintln(os.Stderr, "エラー: --output または --report-root のどちらかは必須です")
		fs.Usage()
		return exitFailure
	}
	if *input == "" && *sites == "" {
		fmt.Fprintln(os.Stderr, "エラー: --input または --sites のどちらかは必須です")
		fs.Usage()
		return exitFailure
	}
	if *input != "" && (*sites != "" || *origins != "") {
		fmt.Fprintln(os.Stderr, "エラー: --input と --sites/--origins は同時に指定できません")
		fs.Usage()
		return exitFailure
	}

	if *configPath != "" {
		settings, err := runconfig.Load(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
			return exitFailure
		}
		applySettings(fs, settings, timeoutStr, warnDays, critDays, urgentDays, checkStr)
	}

	// タイムアウトを解析。
	timeout, err := time.ParseDuration(*timeoutStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "エラー: --timeout の指定が不正です (%q): %v\n", *timeoutStr, err)
		return exitFailure
	}

	// チェック対象モードを解析。
	mode, err := checker.ParseCheckMode(*checkStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
		return exitFailure
	}

	th := model.Thresholds{
		WarnDays:     *warnDays,
		CriticalDays: *critDays,
		UrgentDays:   *urgentDays,
	}

	// 1. CSV読み込み
	targets, err := loadTargets(*input, *sites, *origins, mode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
		return exitFailure
	}
	fmt.Fprintf(os.Stderr, "監視対象 %d 件を読み込みました（チェック対象: %s）\n", len(targets), mode)

	// 2. 各 Target を並行チェック（対象ごとに panic を隔離）。
	results := checkAll(targets, th, timeout, mode)

	// 3. 対応管理表が指定されていれば結果へ反映。
	if *actions != "" {
		actionStatuses, err := model.LoadActionStatuses(*actions)
		if err != nil {
			fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
			return exitFailure
		}
		model.ApplyActionStatuses(results, actionStatuses)
		fmt.Fprintf(os.Stderr, "対応管理 %d 件を読み込みました\n", len(actionStatuses))
	}

	// 4. ソート: ステータス優先度 → 残日数昇順。
	sortResults(results)
	generatedAt := time.Now()

	// 5. CSV 出力
	if *output != "" {
		if err := report.WriteCSV(*output, results); err != nil {
			fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
			return exitFailure
		}
		fmt.Fprintf(os.Stderr, "CSVレポートを書き出しました: %s\n", *output)
	}

	// HTML 出力（指定時のみ）
	if *htmlOut != "" {
		if err := report.WriteHTML(*htmlOut, results, generatedAt); err != nil {
			fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
			return exitFailure
		}
		fmt.Fprintf(os.Stderr, "HTMLレポートを書き出しました: %s\n", *htmlOut)
	}

	if *publicOut != "" {
		if err := report.WritePublicCSV(*publicOut, results); err != nil {
			fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
			return exitFailure
		}
		fmt.Fprintf(os.Stderr, "public CSVレポートを書き出しました: %s\n", *publicOut)
	}

	if *publicHTML != "" {
		if err := report.WritePublicHTML(*publicHTML, results, generatedAt); err != nil {
			fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
			return exitFailure
		}
		fmt.Fprintf(os.Stderr, "public HTMLレポートを書き出しました: %s\n", *publicHTML)
	}

	if *summaryOut != "" {
		if err := report.WriteSummaryJSON(*summaryOut, results, generatedAt); err != nil {
			fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
			return exitFailure
		}
		fmt.Fprintf(os.Stderr, "summary JSONを書き出しました: %s\n", *summaryOut)
	}

	if *reportRoot != "" {
		paths, err := report.WriteBundle(*reportRoot, results, generatedAt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
			return exitFailure
		}
		fmt.Fprintf(os.Stderr, "推奨フォルダ構成へレポートを書き出しました: %s\n", *reportRoot)
		fmt.Fprintf(os.Stderr, "  private latest: %s / %s\n", paths.PrivateLatestCSV, paths.PrivateLatestHTML)
		fmt.Fprintf(os.Stderr, "  public latest : %s / %s\n", paths.PublicLatestCSV, paths.PublicLatestHTML)
		fmt.Fprintf(os.Stderr, "  summary       : %s\n", paths.LatestSummaryJSON)
	}

	// 6. Teams通知（Webhook指定時のみ。ERROR/EXPIRED/URGENT/CRITICAL のみ送信）。
	//    通知失敗は記録するが、レポート自体は成功しているため全体は止めない。
	if *teamsHook != "" {
		sent, nerr := notify.SendTeams(*teamsHook, results, nil)
		switch {
		case nerr != nil:
			fmt.Fprintf(os.Stderr, "警告: Teams通知に失敗しました: %v\n", nerr)
		case sent:
			fmt.Fprintln(os.Stderr, "Teams通知を送信しました")
		default:
			fmt.Fprintln(os.Stderr, "Teams通知対象（ERROR/EXPIRED/URGENT/CRITICAL）が無いため送信しませんでした")
		}
	}

	// 7. サマリを標準出力に表示し、終了コードを決定。
	alert := printSummary(results)
	if alert {
		return exitAlert
	}
	return exitOK
}

func loadTargets(input, sitesPath, originsPath string, mode checker.CheckMode) ([]model.Target, error) {
	if input != "" {
		return model.LoadTargets(input)
	}

	sites, err := model.LoadSites(sitesPath)
	if err != nil {
		return nil, err
	}

	requireOrigin := mode == checker.CheckOrigin || mode == checker.CheckBoth
	if requireOrigin && originsPath == "" {
		return nil, fmt.Errorf("--check %s では --origins が必須です", mode)
	}

	var origins []model.OriginConfig
	if originsPath != "" {
		origins, err = model.LoadOrigins(originsPath)
		if err != nil {
			return nil, err
		}
	}
	return model.MergeSiteOrigins(sites, origins, requireOrigin)
}

func applySettings(fs *flag.FlagSet, settings runconfig.Settings, timeoutStr *string, warnDays, critDays, urgentDays *int, checkStr *string) {
	if settings.Timeout != "" && !flagWasSet(fs, "timeout") {
		*timeoutStr = settings.Timeout
	}
	if settings.WarnDays > 0 && !flagWasSet(fs, "warn-days") {
		*warnDays = settings.WarnDays
	}
	if settings.CriticalDays > 0 && !flagWasSet(fs, "critical-days") {
		*critDays = settings.CriticalDays
	}
	if settings.UrgentDays > 0 && !flagWasSet(fs, "urgent-days") {
		*urgentDays = settings.UrgentDays
	}
	if settings.Check != "" && !flagWasSet(fs, "check") {
		*checkStr = settings.Check
	}
}

func flagWasSet(fs *flag.FlagSet, name string) bool {
	wasSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			wasSet = true
		}
	})
	return wasSet
}

// checkAll は全対象を並行にチェックする。
// 各 goroutine 内で recover し、1対象の panic で全体を止めない。
func checkAll(targets []model.Target, th model.Thresholds, timeout time.Duration, mode checker.CheckMode) []model.CertResult {
	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results []model.CertResult
	)

	for _, t := range targets {
		wg.Add(1)
		go func(t model.Target) {
			defer wg.Done()
			// panic を ERROR 結果に変換して隔離する。
			defer func() {
				if rec := recover(); rec != nil {
					mu.Lock()
					results = append(results, model.CertResult{
						ServiceID:   t.ServiceID,
						ServiceName: t.ServiceName,
						Kind:        "Unknown",
						Status:      model.StatusError,
						Error:       fmt.Sprintf("内部エラー(panic): %v", rec),
						Owner:       t.Owner,
						Vendor:      t.Vendor,
						Notes:       t.Notes,
					})
					mu.Unlock()
				}
			}()

			rs := checker.CheckTarget(t, th, timeout, mode)
			mu.Lock()
			results = append(results, rs...)
			mu.Unlock()
		}(t)
	}

	wg.Wait()
	return results
}

// sortResults はレポート用に結果を並べ替える。
// プライマリ: StatusRank（重大度）昇順、セカンダリ: 残日数昇順、
// 同点はサービス名→Kind で安定化。
func sortResults(results []model.CertResult) {
	sort.SliceStable(results, func(i, j int) bool {
		ri, rj := results[i], results[j]
		if a, b := model.StatusRank(ri.Status), model.StatusRank(rj.Status); a != b {
			return a < b
		}
		// 同ステータス内は残日数が少ない（期限が近い）ほど上。
		// ERROR は残日数が無意味なので比較対象から外し、サービス名で安定化。
		if ri.Status != model.StatusError && ri.DaysLeft != rj.DaysLeft {
			return ri.DaysLeft < rj.DaysLeft
		}
		if ri.ServiceName != rj.ServiceName {
			return ri.ServiceName < rj.ServiceName
		}
		return ri.Kind < rj.Kind
	})
}

// printSummary はステータス別件数を標準出力に表示し、
// アラート（EXPIRED/URGENT/CRITICAL/ERROR）が1件でもあれば true を返す。
func printSummary(results []model.CertResult) bool {
	order := []model.Status{
		model.StatusError, model.StatusExpired, model.StatusUrgent,
		model.StatusCritical, model.StatusWarn, model.StatusOK,
	}
	counts := map[model.Status]int{}
	for _, r := range results {
		counts[r.Status]++
	}

	fmt.Println("=== サマリ ===")
	alert := false
	for _, s := range order {
		fmt.Printf("  %-9s : %d\n", s, counts[s])
		if s.IsAlert() && counts[s] > 0 {
			alert = true
		}
	}
	fmt.Printf("  合計      : %d\n", len(results))
	return alert
}
