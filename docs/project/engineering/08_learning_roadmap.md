<!-- generated-by: scripts/generate_engineering_docs.py -->
# certificate_chekcer — 学習・保守ロードマップ

> 生成日: 2026-07-15 / 対象: `certificate_chekcer` / 確度: [低]
> 実装・manifest・既存資料の静的棚卸しに基づく。外部サービスの稼働状態と本番構成は未検証。

## Day 1: 起動と全体像

1. install候補: `go mod download`
2. 最初の実行/検査: `検証command未検出`
3. `cert-watcher/cmd/cert-watcher/main.go` を読み、実行entrypointの境界を確認


## Day 2–3: 主要契約

- APIがない/未検出であることを確認
- 永続化方式がfile/memory/external/なしのどれかを確定
- CLI/API/docs入口の成功・失敗フィードバックを確認
- external/config: 外部integration未検出 / 設定名未検出

## 最初の変更前

- 変更対象に最も近いtest: `cert-watcher/cmd/cert-watcher/main_test.go`, `cert-watcher/internal/notify/teams_test.go`, `cert-watcher/internal/checker/tls_checker_test.go`, `cert-watcher/internal/model/action_status_test.go`, `cert-watcher/internal/model/target_test.go`, `cert-watcher/internal/model/result_test.go`, `cert-watcher/internal/model/separated_input_test.go`, `cert-watcher/internal/runconfig/settings_test.go`, `cert-watcher/internal/report/summary_test.go`, `cert-watcher/internal/report/public_report_test.go`
- 既存ADR/docs: `architecture.drawio`, `cert-watcher/README.md`, `cert-watcher/docs/04_api_spec.md`, `cert-watcher/docs/05_er_diagram.md`, `cert-watcher/docs/03_tech_stack.md`, `cert-watcher/docs/02_architecture.md`, `cert-watcher/docs/README.md`, `cert-watcher/docs/07_directory_structure.md`, `cert-watcher/docs/06_table_design.md`, `cert-watcher/docs/01_concept.md`
- runtime: Docker (`cert-watcher/Dockerfile`)
- `07_traceability.md` の未確認事項をcloseまたはrisk acceptしてから変更する。

## Doneの定義

- build/type/lint/testのうち存在するgateが通る。
- API/data/UI/runtimeの変更に対応する文書とADRを更新する。
- rollback、秘密情報、外部送信、production影響をreviewで明示する。
