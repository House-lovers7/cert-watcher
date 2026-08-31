# ER 図 — Akamai-Origin Certificate Watcher

> 将来構成（PostgreSQL）のデータモデル。現状の CLI は CSV 入力 → メモリ処理のため DB を持たないが、目標構成では以下を永続化する。

## 1. エンティティ関連図

```mermaid
erDiagram
    TARGETS ||--o{ CERT_RESULTS : "監視対象として持つ"
    CHECK_RUNS ||--o{ CERT_RESULTS : "実行バッチに含む"
    CHECK_RUNS ||--o{ NOTIFICATIONS : "通知を発生させる"

    TARGETS {
        text   id PK
        text   service_name
        text   edge_fqdn
        text   origin_host
        text   origin_sni
        text   owner
        text   vendor
        text   notes
        boolean enabled
        timestamptz created_at
        timestamptz updated_at
    }

    CHECK_RUNS {
        text   id PK
        text   mode "edge|origin|both"
        text   trigger "manual|scheduled|api"
        text   status "running|finished|failed"
        int    timeout_seconds
        int    warn_days
        int    critical_days
        int    urgent_days
        int    total
        int    ok_count
        int    warn_count
        int    critical_count
        int    urgent_count
        int    expired_count
        int    error_count
        timestamptz started_at
        timestamptz finished_at
    }

    CERT_RESULTS {
        text   id PK
        text   check_run_id FK
        text   target_id FK
        text   kind "EDGE|ORIGIN"
        text   host
        int    port
        text   sni
        text   status "OK|WARN|CRITICAL|URGENT|EXPIRED|ERROR"
        int    days_left
        timestamptz not_before
        timestamptz not_after
        text   subject
        text   issuer
        text   dns_names "配列(JSONB or text[])"
        text   error
        timestamptz checked_at
    }

    NOTIFICATIONS {
        text   id PK
        text   check_run_id FK
        text   channel "teams|email"
        text   status "sent|skipped|failed"
        int    target_count
        text   summary
        text   error
        timestamptz sent_at
    }
```

---

## 2. リレーション要点

| 関係 | カーディナリティ | 説明 |
|------|------------------|------|
| `targets` → `cert_results` | 1 : N | 1つの監視対象は多数の検査結果（実行ごと×Edge/Origin）を持つ |
| `check_runs` → `cert_results` | 1 : N | 1回の実行は複数の証明書結果を含む |
| `check_runs` → `notifications` | 1 : N | 1回の実行から複数チャネルの通知が発生し得る |

- `cert_results` は `(check_run_id, target_id, kind)` の組で一意。
- `target_id` は `ON DELETE SET NULL`（対象削除後も過去の証跡を残す）。
- 「最新結果」は `cert_results` を `target_id, kind` ごとに `checked_at` 降順で取得して得る。

---

## 3. 状態遷移（check_runs.status）

```mermaid
stateDiagram-v2
    [*] --> running: チェック実行受理
    running --> finished: 全対象処理完了
    running --> failed: 致命的エラー(DB障害等)
    finished --> [*]
    failed --> [*]
```

> 個々の対象の接続失敗は `cert_results.status = ERROR` として記録され、`check_runs` 自体は `finished` になる（1対象失敗で全体を止めない設計）。
