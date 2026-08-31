<!-- generated-by: scripts/generate_engineering_docs.py -->
# certificate_chekcer — 非機能要件・SLO/SLI

> 生成日: 2026-07-15 / 対象: `certificate_chekcer` / 確度: [低]
> 実装・manifest・既存資料の静的棚卸しに基づく。外部サービスの稼働状態と本番構成は未検証。

## 現在コード化されている品質ゲート

| Gate | Command | 根拠 |
|---|---|---|
| validation | 未検出 | README/CIで正典を確定 |

- test files: 11（`cert-watcher/cmd/cert-watcher/main_test.go`, `cert-watcher/internal/notify/teams_test.go`, `cert-watcher/internal/checker/tls_checker_test.go`, `cert-watcher/internal/model/action_status_test.go`, `cert-watcher/internal/model/target_test.go`, `cert-watcher/internal/model/result_test.go`, `cert-watcher/internal/model/separated_input_test.go`, `cert-watcher/internal/runconfig/settings_test.go`, `cert-watcher/internal/report/summary_test.go`, `cert-watcher/internal/report/public_report_test.go`, `cert-watcher/internal/report/bundle_test.go`）
- quality/CI config: 未検出
- security/resilience signal: resilience (`cert-watcher/cmd/cert-watcher/main.go`), resilience (`cert-watcher/cmd/cert-watcher/main_test.go`), resilience (`cert-watcher/internal/notify/teams.go`), resilience (`cert-watcher/internal/checker/tls_checker.go`), resilience (`cert-watcher/internal/checker/tls_checker_test.go`), resilience (`cert-watcher/internal/runconfig/settings_test.go`), resilience (`cert-watcher/internal/runconfig/settings.go`)

## 計測すべきSLI

| Boundary | SLI | 最初の計測根拠 |
|---|---|---|
| CLI/Job | exit code・処理件数・失敗件数・処理時間 | `cert-watcher/cmd/cert-watcher/main.go` |

## SLOの状態

[高] 合意済みSLO数値はrepository内の実装・資料から確認できていない。任意の99%や2秒を現在要件として記載しない。利用者、運用時間帯、障害コスト、予算を確認してから、上記SLIごとにtarget/window/error budgetを決める。

## 運用境界

- runtime/config: Docker (`cert-watcher/Dockerfile`)
- required config names: example/sourceから未検出
- 外部integration: 静的検出なし
- rollbackはcode、schema、generated artifact、provider設定を分ける。production操作は人間承認後に行う。
