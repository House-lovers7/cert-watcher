<!-- generated-by: scripts/generate_engineering_docs.py -->
# certificate_chekcer — 実装トレーサビリティ

> 生成日: 2026-07-15 / 対象: `certificate_chekcer` / 確度: [低]
> 実装・manifest・既存資料の静的棚卸しに基づく。外部サービスの稼働状態と本番構成は未検証。

## 根拠台帳

| 種別 | Path | 状態 |
|---|---|---|
| manifest | `cert-watcher/go.mod` | 静的確認済み |
| entrypoint | `cert-watcher/cmd/cert-watcher/main.go` | 静的確認済み |
| test | `cert-watcher/cmd/cert-watcher/main_test.go` | 静的確認済み |
| test | `cert-watcher/internal/notify/teams_test.go` | 静的確認済み |
| test | `cert-watcher/internal/checker/tls_checker_test.go` | 静的確認済み |
| test | `cert-watcher/internal/model/action_status_test.go` | 静的確認済み |
| test | `cert-watcher/internal/model/target_test.go` | 静的確認済み |
| test | `cert-watcher/internal/model/result_test.go` | 静的確認済み |
| test | `cert-watcher/internal/model/separated_input_test.go` | 静的確認済み |
| test | `cert-watcher/internal/runconfig/settings_test.go` | 静的確認済み |
| test | `cert-watcher/internal/report/summary_test.go` | 静的確認済み |
| test | `cert-watcher/internal/report/public_report_test.go` | 静的確認済み |
| test | `cert-watcher/internal/report/bundle_test.go` | 静的確認済み |
| existing docs | `architecture.drawio` | 静的確認済み |
| existing docs | `cert-watcher/README.md` | 静的確認済み |
| existing docs | `cert-watcher/docs/04_api_spec.md` | 静的確認済み |
| existing docs | `cert-watcher/docs/05_er_diagram.md` | 静的確認済み |
| existing docs | `cert-watcher/docs/03_tech_stack.md` | 静的確認済み |
| existing docs | `cert-watcher/docs/02_architecture.md` | 静的確認済み |
| existing docs | `cert-watcher/docs/README.md` | 静的確認済み |
| existing docs | `cert-watcher/docs/07_directory_structure.md` | 静的確認済み |
| existing docs | `cert-watcher/docs/06_table_design.md` | 静的確認済み |
| existing docs | `cert-watcher/docs/01_concept.md` | 静的確認済み |

## 検出した検証command

- manifestから標準command未検出

## 設定契約（名前のみ）

- example/sourceから環境変数名を検出できず

値、credential、顧客データは収集していない。設定のrequired/optional、format、取得元は各entrypointとruntimeで確認する。

## 既存文書との関係

- `architecture.drawio`
- `cert-watcher/README.md`
- `cert-watcher/docs/04_api_spec.md`
- `cert-watcher/docs/05_er_diagram.md`
- `cert-watcher/docs/03_tech_stack.md`
- `cert-watcher/docs/02_architecture.md`
- `cert-watcher/docs/README.md`
- `cert-watcher/docs/07_directory_structure.md`
- `cert-watcher/docs/06_table_design.md`
- `cert-watcher/docs/01_concept.md`

既存ADR・公式schema・運用runbookがある場合はそれらを正典とし、generated docsは発見用索引として扱う。矛盾を見つけたら実装・正式文書・生成器のどれを直すかをreviewで決める。

## 未確認事項

- 動的route/schema/plugin、external gateway、mobile native設定。
- secret manager、provider console、production runtimeの値と適用version。
- migration適用状態、SLO実績、実データ量、owner/on-call。

## 更新ルール

- route/schema/screen/runtime構成を変更した差分では、対応する文書を同時更新する。
- 生成し直す前に手書き文書を正典へ昇格するか、生成対象外へ分離する。
- このディレクトリの `generated-by` marker付きファイルは本スクリプトで再生成できる。
