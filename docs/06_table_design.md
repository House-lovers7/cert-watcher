# テーブル設計 — Akamai-Origin Certificate Watcher

> PostgreSQL 16 想定。ステータス等は CHECK 制約で値を限定する（ENUM 型でも可だが、追加変更の柔軟性から本設計では `text + CHECK` を採用）。
> ID は ULID/UUID を文字列で保持（アプリ生成）。日時は `timestamptz`（UTC）。

---

## 1. `targets` — 監視対象

| カラム | 型 | 制約 | 説明 |
|--------|----|------|------|
| `id` | `text` | PK | ULID |
| `service_name` | `text` | NOT NULL | サービス名 |
| `edge_fqdn` | `text` | NOT NULL | Edge FQDN（接続先 / SNI） |
| `origin_host` | `text` | NOT NULL | Origin 接続先ホスト |
| `origin_sni` | `text` | NOT NULL | Origin 接続時の SNI |
| `owner` | `text` | | 管理担当者 |
| `vendor` | `text` | | ベンダー |
| `notes` | `text` | | 備考 |
| `enabled` | `boolean` | NOT NULL DEFAULT true | 監視有効フラグ |
| `created_at` | `timestamptz` | NOT NULL DEFAULT now() | 作成日時 |
| `updated_at` | `timestamptz` | NOT NULL DEFAULT now() | 更新日時 |

```sql
CREATE TABLE targets (
    id           text PRIMARY KEY,
    service_name text NOT NULL,
    edge_fqdn    text NOT NULL,
    origin_host  text NOT NULL,
    origin_sni   text NOT NULL,
    owner        text NOT NULL DEFAULT '',
    vendor       text NOT NULL DEFAULT '',
    notes        text NOT NULL DEFAULT '',
    enabled      boolean NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- サービス名 + Edge FQDN の重複登録を防止
CREATE UNIQUE INDEX uq_targets_service_edge ON targets (service_name, edge_fqdn);
-- 有効な対象の絞り込み・担当者検索
CREATE INDEX idx_targets_enabled ON targets (enabled);
CREATE INDEX idx_targets_owner   ON targets (owner);
```

---

## 2. `check_runs` — チェック実行バッチ

| カラム | 型 | 制約 | 説明 |
|--------|----|------|------|
| `id` | `text` | PK | ULID |
| `mode` | `text` | NOT NULL, CHECK | `edge` / `origin` / `both` |
| `trigger` | `text` | NOT NULL, CHECK | `manual` / `scheduled` / `api` |
| `status` | `text` | NOT NULL, CHECK | `running` / `finished` / `failed` |
| `timeout_seconds` | `int` | NOT NULL DEFAULT 10 | 接続タイムアウト |
| `warn_days` | `int` | NOT NULL DEFAULT 30 | WARN しきい値 |
| `critical_days` | `int` | NOT NULL DEFAULT 14 | CRITICAL しきい値 |
| `urgent_days` | `int` | NOT NULL DEFAULT 7 | URGENT しきい値 |
| `total` | `int` | NOT NULL DEFAULT 0 | 結果総数 |
| `ok_count` / `warn_count` / `critical_count` / `urgent_count` / `expired_count` / `error_count` | `int` | NOT NULL DEFAULT 0 | ステータス別集計 |
| `started_at` | `timestamptz` | NOT NULL DEFAULT now() | 開始 |
| `finished_at` | `timestamptz` | NULL | 終了（実行中は NULL） |

```sql
CREATE TABLE check_runs (
    id              text PRIMARY KEY,
    mode            text NOT NULL CHECK (mode IN ('edge','origin','both')),
    trigger         text NOT NULL CHECK (trigger IN ('manual','scheduled','api')),
    status          text NOT NULL CHECK (status IN ('running','finished','failed')),
    timeout_seconds int  NOT NULL DEFAULT 10,
    warn_days       int  NOT NULL DEFAULT 30,
    critical_days   int  NOT NULL DEFAULT 14,
    urgent_days     int  NOT NULL DEFAULT 7,
    total           int  NOT NULL DEFAULT 0,
    ok_count        int  NOT NULL DEFAULT 0,
    warn_count      int  NOT NULL DEFAULT 0,
    critical_count  int  NOT NULL DEFAULT 0,
    urgent_count    int  NOT NULL DEFAULT 0,
    expired_count   int  NOT NULL DEFAULT 0,
    error_count     int  NOT NULL DEFAULT 0,
    started_at      timestamptz NOT NULL DEFAULT now(),
    finished_at     timestamptz
);

CREATE INDEX idx_check_runs_started ON check_runs (started_at DESC);
CREATE INDEX idx_check_runs_status  ON check_runs (status);
```

---

## 3. `cert_results` — 証明書チェック結果

| カラム | 型 | 制約 | 説明 |
|--------|----|------|------|
| `id` | `text` | PK | ULID |
| `check_run_id` | `text` | NOT NULL, FK→check_runs(id) ON DELETE CASCADE | 実行バッチ |
| `target_id` | `text` | FK→targets(id) ON DELETE SET NULL | 監視対象（削除後も証跡保持） |
| `kind` | `text` | NOT NULL, CHECK | `EDGE` / `ORIGIN` |
| `host` | `text` | NOT NULL | 接続先ホスト |
| `port` | `int` | NOT NULL DEFAULT 443 | 接続先ポート |
| `sni` | `text` | NOT NULL | 提示 SNI |
| `status` | `text` | NOT NULL, CHECK | 6種ステータス |
| `days_left` | `int` | NULL | 残日数（ERROR時 NULL） |
| `not_before` | `timestamptz` | NULL | 有効期間開始 |
| `not_after` | `timestamptz` | NULL | 有効期間終了 |
| `subject` | `text` | NOT NULL DEFAULT '' | Subject |
| `issuer` | `text` | NOT NULL DEFAULT '' | Issuer |
| `dns_names` | `text[]` | NOT NULL DEFAULT '{}' | SAN（配列） |
| `error` | `text` | NOT NULL DEFAULT '' | エラー内容 |
| `checked_at` | `timestamptz` | NOT NULL DEFAULT now() | 検査時刻 |

```sql
CREATE TABLE cert_results (
    id           text PRIMARY KEY,
    check_run_id text NOT NULL REFERENCES check_runs(id) ON DELETE CASCADE,
    target_id    text REFERENCES targets(id) ON DELETE SET NULL,
    kind         text NOT NULL CHECK (kind IN ('EDGE','ORIGIN')),
    host         text NOT NULL,
    port         int  NOT NULL DEFAULT 443,
    sni          text NOT NULL DEFAULT '',
    status       text NOT NULL CHECK (status IN ('OK','WARN','CRITICAL','URGENT','EXPIRED','ERROR')),
    days_left    int,
    not_before   timestamptz,
    not_after    timestamptz,
    subject      text NOT NULL DEFAULT '',
    issuer       text NOT NULL DEFAULT '',
    dns_names    text[] NOT NULL DEFAULT '{}',
    error        text NOT NULL DEFAULT '',
    checked_at   timestamptz NOT NULL DEFAULT now()
);

-- 同一実行内で 対象×系統 は一意
CREATE UNIQUE INDEX uq_cert_results_run_target_kind
    ON cert_results (check_run_id, target_id, kind);
-- 実行詳細の取得
CREATE INDEX idx_cert_results_run ON cert_results (check_run_id);
-- 「最新結果」取得用（target×kind の最新）
CREATE INDEX idx_cert_results_target_kind_time
    ON cert_results (target_id, kind, checked_at DESC);
-- アラート横断検索（要対応のみ）
CREATE INDEX idx_cert_results_alert
    ON cert_results (status, not_after)
    WHERE status IN ('EXPIRED','URGENT','CRITICAL','ERROR');
```

---

## 4. `notifications` — 通知ログ

| カラム | 型 | 制約 | 説明 |
|--------|----|------|------|
| `id` | `text` | PK | ULID |
| `check_run_id` | `text` | NOT NULL, FK→check_runs(id) ON DELETE CASCADE | 実行バッチ |
| `channel` | `text` | NOT NULL, CHECK | `teams` / `email` |
| `status` | `text` | NOT NULL, CHECK | `sent` / `skipped` / `failed` |
| `target_count` | `int` | NOT NULL DEFAULT 0 | 通知対象件数 |
| `summary` | `text` | NOT NULL DEFAULT '' | 件数サマリ等 |
| `error` | `text` | NOT NULL DEFAULT '' | 送信エラー |
| `sent_at` | `timestamptz` | NOT NULL DEFAULT now() | 送信/判定時刻 |

```sql
CREATE TABLE notifications (
    id           text PRIMARY KEY,
    check_run_id text NOT NULL REFERENCES check_runs(id) ON DELETE CASCADE,
    channel      text NOT NULL CHECK (channel IN ('teams','email')),
    status       text NOT NULL CHECK (status IN ('sent','skipped','failed')),
    target_count int  NOT NULL DEFAULT 0,
    summary      text NOT NULL DEFAULT '',
    error        text NOT NULL DEFAULT '',
    sent_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_run ON notifications (check_run_id);
```

---

## 5. 代表的なクエリ

### 5.1 各対象×系統の「最新」結果（ダッシュボード一覧）
```sql
SELECT DISTINCT ON (cr.target_id, cr.kind) cr.*
FROM cert_results cr
WHERE cr.target_id IS NOT NULL
ORDER BY cr.target_id, cr.kind, cr.checked_at DESC;
```

### 5.2 要対応（アラート）一覧
```sql
SELECT *
FROM cert_results
WHERE status IN ('EXPIRED','URGENT','CRITICAL','ERROR')
ORDER BY
  CASE status
    WHEN 'ERROR' THEN 0 WHEN 'EXPIRED' THEN 1
    WHEN 'URGENT' THEN 2 WHEN 'CRITICAL' THEN 3 ELSE 4
  END,
  days_left NULLS LAST;
```

### 5.3 直近実行のサマリ
```sql
SELECT id, started_at, total,
       expired_count, urgent_count, critical_count, error_count
FROM check_runs
WHERE status = 'finished'
ORDER BY started_at DESC
LIMIT 10;
```

---

## 6. `updated_at` 自動更新トリガ（targets）

```sql
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN NEW.updated_at = now(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_targets_updated_at
    BEFORE UPDATE ON targets
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
```

---

## 7. 設計メモ

- **証跡保持**: `target_id` は `SET NULL`。対象を消しても過去の検査履歴は残る。
- **集計の二重持ち**: `check_runs` にステータス別カウントを保持（一覧表示の高速化）。整合は実行完了時に確定。
- **ERROR の `days_left`**: NULL とし、アプリ/レポート側で空欄表示。
- **ステータス制約**: `text + CHECK`。値追加時は制約を ALTER で更新（ENUM より柔軟）。
- **CLI との対応**: CLI の `CertResult`（`internal/model/result.go`）と `cert_results` がほぼ1:1対応。CLI 出力をそのまま INSERT 可能。
