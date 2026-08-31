<!-- generated-by: scripts/generate_engineering_docs.py -->
# certificate_chekcer — Engineering Handbook / Start Here

> 生成日: 2026-07-15 / 対象: `certificate_chekcer` / 確度: [低]
> 実装・manifest・既存資料の静的棚卸しに基づく。外部サービスの稼働状態と本番構成は未検証。

## 60分で把握する

1. コンセプト: certificate_chekcer は セキュリティ/認証/監査/決済 領域で、不安 の解消を狙うプロジェクトと推定。
2. classification: `active_project` / stack: Go
3. install: `go mod download`
4. run/check: manifest script未検出
5. entrypoint: `cert-watcher/cmd/cert-watcher/main.go`

## 実装スナップショット

| 項目 | 現在値 | 最初に読むpath |
|---|---:|---|
| package/component | 1 | `cert-watcher/cmd/cert-watcher/main.go` |
| API | 0 | 未検出 |
| entity | 0 | 未検出 |
| screen/entry UI | 0 | 未検出 |
| test files | 11 | `cert-watcher/cmd/cert-watcher/main_test.go` |

## 最初に確認する既存の正典候補

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

既存ADR、OpenAPI、schema、運用runbookがある場合は、下記generated docsより先に読む。

## 引継ぎblocking / partial

| Priority | Requirement | 状態・理由 | Evidence |
|---|---|---|---|
| P1 | `observability` | missing: 可観測性の実装証拠を特定できない。生成文書の一般要件・提案は現行実装の証拠ではない。 | `cert-watcher/README.md` |
| P1 | `rollback` | missing: build/deploy可能だが実行可能なrollback手順がない。生成NFRもrelease前に定義としている。 | `docs/engineering/05_nfr_slo.md` |
| P1 | `startup_and_verification_commands` | missing: 起動・検証commandを静的証拠から特定できず、生成文書にも実行可能な手順がない。 | `cert-watcher/README.md` |

## 読む順番

1. [One Pager](./00_one_pager.md)
2. [技術スタック比較](./01_stack_comparison.md)
3. [アーキテクチャ・システム構成](./02_architecture.md)
4. [ADR](./03_adrs/ADR-0001-current-implementation-baseline.md)
5. [API定義](./04_api.md)
6. [データモデル・ER図](./05_data_model.md)
7. [非機能要件・SLO/SLI](./05_nfr_slo.md)
8. [画面設計](./06_screen_design.md)
9. [P50/P90見積り](./06_estimation.md)
10. [実装トレーサビリティ](./07_traceability.md)
11. [学習・保守ロードマップ](./08_learning_roadmap.md)

## 使い方

- generated docsは実装発見用handbook。既存ADR、OpenAPI、schema、runbookがある場合は既存正典を優先する。
- path・数・versionは静的検出した事実。目的やpath由来の責務は `[中]` の推定を含む。
- production、external console、secret値、migration適用状態は未確認。
