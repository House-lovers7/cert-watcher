<!-- generated-by: scripts/generate_engineering_docs.py -->
# certificate_chekcer — One Pager / オンボーディング概要

> 生成日: 2026-07-15 / 対象: `certificate_chekcer` / 確度: [低]
> 実装・manifest・既存資料の静的棚卸しに基づく。外部サービスの稼働状態と本番構成は未検証。

## コンセプト

certificate_chekcer は セキュリティ/認証/監査/決済 領域で、不安 の解消を狙うプロジェクトと推定。

## 誰の何を解くか

- 対象領域: セキュリティ/認証/監査/決済
- 想定利用者: 開発/セキュリティ/運用担当
- 価値仮説: 設定/ログ/リクエスト/差分を検査し、危険差分・承認・監査証跡・決済境界を可視化。

## 現在地

| 項目 | 観測結果 |
|---|---|
| 技術スタック | Go |
| API | 0 endpoint signal |
| データモデル | 0 unique entity signal |
| 画面 | 0 route/screen signal |
| 実行基盤 | Docker (`cert-watcher/Dockerfile`) |
| package / module | 1 component signal |
| tests | 11 file signal |

## ソースマップ

| Component | Path | 責務 |
|---|---|---|
| `main` | `cert-watcher/cmd/cert-watcher/main.go` | 実行entrypoint |

## 最初に使うコマンド

| 目的 | Command |
|---|---|
| 未検出 | READMEまたはCIから確認 |

## 変更箇所の入口

| 変更対象 | 最初に読むpath | 同時に確認するもの |
|---|---|---|
| 実行・配備 | `cert-watcher/Dockerfile` | 環境変数、service依存、rollback |
| 回帰検査 | `cert-watcher/cmd/cert-watcher/main_test.go` | 変更対象に近いtestと全体check |

## 引継ぎ時の未解決ギャップ

| Priority | Requirement | 状態・理由 | Evidence |
|---|---|---|---|
| P1 | `observability` | missing: 可観測性の実装証拠を特定できない。生成文書の一般要件・提案は現行実装の証拠ではない。 | `cert-watcher/README.md` |
| P1 | `rollback` | missing: build/deploy可能だが実行可能なrollback手順がない。生成NFRもrelease前に定義としている。 | `docs/engineering/05_nfr_slo.md` |
| P1 | `startup_and_verification_commands` | missing: 起動・検証commandを静的証拠から特定できず、生成文書にも実行可能な手順がない。 | `cert-watcher/README.md` |

## スコープ境界

- [高] productionの稼働、外部provider設定、secret値は未確認。
- [高] API・DB・画面が未検出の場合は推測せず、実装入口の追加を課題として残す。
- [中] 初回変更前に `07_traceability.md` の根拠と未確認事項を確認する。
