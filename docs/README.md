# 設計ドキュメント — Akamai-Origin Certificate Watcher

Akamai 構成サイトの **Edge 証明書** / **Origin 証明書** の有効期限を監視するツールの設計ドキュメント群。

現状の CLI（v0.2）を「コアエンジン」とし、API + DB + 通知 + スケジューラを備えた **目標アーキテクチャ** を設計図として記述している。

> 図はすべて **Mermaid** 記法。GitHub / VS Code 等でそのまま描画される。

---

## 目次

| # | ドキュメント | 内容 |
|---|--------------|------|
| 01 | [コンセプト](01_concept.md) | 課題・目的・ターゲット・提供価値・スコープ・ロードマップ・非ゴール |
| 02 | [システム設計図](02_architecture.md) | 全体構成・コンポーネント責務・データフロー・Edge/Origin実行環境分離・AWSデプロイ |
| 03 | [技術スタック](03_tech_stack.md) | 言語/DB/インフラ/CI/通知の選定と理由・現状vs目標 |
| 04 | [API定義書](04_api_spec.md) | REST API（targets/checks/results/reports/notifications）・req/res・ステータスコード |
| 05 | [ER図](05_er_diagram.md) | エンティティ関連図・リレーション・状態遷移 |
| 06 | [テーブル設計](06_table_design.md) | PostgreSQL DDL・制約・インデックス・代表クエリ |
| 07 | [ディレクトリ構成図](07_directory_structure.md) | 現状構成・目標構成・増設方針 |

利用方法・CLIオプションは ルートの [README.md](../README.md) を参照。

---

## ステータス定義（共通）

| ステータス | 条件 | 通知 |
|------------|------|------|
| `OK` | 残 31 日以上 | — |
| `WARN` | 残 30 日以下 | — |
| `CRITICAL` | 残 14 日未満 | ✅ |
| `URGENT` | 残 7 日未満 | ✅ |
| `EXPIRED` | 期限切れ | ✅ |
| `ERROR` | 接続不可・取得失敗 | レポートに記録 |

---

## 実装状況サマリ

| Phase | 内容 | 状態 |
|-------|------|------|
| Phase 1 | ローカル実行版（CSV→TLS→判定→CSV/HTML） | ✅ v0.1 |
| Phase 2 | 定期実行（タスクスケジューラ/cron/Docker） | ✅ v0.2 |
| Phase 3 | Teams 通知（ERROR/EXPIRED/URGENT/CRITICAL） | ✅ |
| Phase 4 | Akamai / AWS 連携（ACM・ALB・Property） | ⬜ 未着手 |
| 将来構想 | API + DB + ダッシュボード | ⬜ 本ドキュメント群で設計 |
