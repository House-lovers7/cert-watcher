# API 定義書 — Akamai-Origin Certificate Watcher

> 本 API は **将来構成**（API + DB）のための定義。現状の CLI と同じチェックエンジンを HTTP 経由で提供する。

## 0. 基本仕様

| 項目 | 内容 |
|------|------|
| ベースURL | `/api/v1` |
| プロトコル | HTTPS |
| 形式 | リクエスト/レスポンスとも `application/json`（UTF-8） |
| 認証 | Bearer Token（`Authorization: Bearer <token>`）※社内利用想定。将来 OIDC 化 |
| 日付形式 | ISO 8601（`2026-07-01T00:00:00Z`） |
| タイムゾーン | UTC で保存・返却。表示側でローカル変換 |

### 共通エラー形式

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "edge_fqdn は必須です",
    "details": [
      { "field": "edge_fqdn", "issue": "required" }
    ]
  }
}
```

### 主なステータスコード

| コード | 意味 |
|--------|------|
| 200 | 取得・更新成功 |
| 201 | 作成成功 |
| 202 | チェック実行を受理（非同期） |
| 204 | 削除成功（本文なし） |
| 400 | バリデーションエラー |
| 401 | 認証エラー |
| 404 | リソースなし |
| 409 | 競合（重複登録など） |
| 500 | サーバ内部エラー |

### ステータス ENUM
`OK` / `WARN` / `CRITICAL` / `URGENT` / `EXPIRED` / `ERROR`

### Kind / Mode ENUM
- `kind`: `EDGE` / `ORIGIN`
- `mode`: `edge` / `origin` / `both`

---

## 1. Targets（監視対象）

### 1.1 一覧取得 — `GET /api/v1/targets`

クエリ: `?enabled=true&owner=platform-team&q=bisco&limit=50&offset=0`

```json
200 OK
{
  "items": [
    {
      "id": "t_01HF...",
      "service_name": "s-bisco",
      "edge_fqdn": "www.s-bisco.jp",
      "origin_host": "origin.example.com",
      "origin_sni": "www.s-bisco.jp",
      "owner": "担当者",
      "vendor": "取引先",
      "notes": "Akamai経由",
      "enabled": true,
      "created_at": "2026-05-01T00:00:00Z",
      "updated_at": "2026-05-20T00:00:00Z"
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
```

### 1.2 作成 — `POST /api/v1/targets`

```json
// Request
{
  "service_name": "s-bisco",
  "edge_fqdn": "www.s-bisco.jp",
  "origin_host": "origin.example.com",
  "origin_sni": "www.s-bisco.jp",
  "owner": "担当者",
  "vendor": "取引先",
  "notes": "Akamai経由",
  "enabled": true
}
// Response 201 Created → 作成された Target オブジェクト
```

必須: `service_name`, `edge_fqdn`, `origin_host`, `origin_sni`。

### 1.3 取得 — `GET /api/v1/targets/{id}`
→ `200` Target オブジェクト / `404`

### 1.4 更新 — `PUT /api/v1/targets/{id}`
→ Target の全項目を更新。`200` 更新後オブジェクト

### 1.5 削除 — `DELETE /api/v1/targets/{id}`
→ `204`（論理削除: `enabled=false` 運用も可）

### 1.6 一括取込 — `POST /api/v1/targets:import`
CSV を本文（`text/csv`）または `{ "csv": "..." }` で受け取り一括登録。

```json
200 OK
{ "created": 12, "updated": 3, "skipped": 1, "errors": [] }
```

---

## 2. Checks（チェック実行）

### 2.1 実行 — `POST /api/v1/checks`

```json
// Request
{
  "mode": "both",                 // edge / origin / both
  "target_ids": ["t_01HF..."],    // 省略時は enabled 全件
  "timeout": "10s",
  "thresholds": { "warn_days": 30, "critical_days": 14, "urgent_days": 7 },
  "notify": { "teams": true }
}
// Response 202 Accepted
{
  "check_run_id": "cr_01HF...",
  "status": "running",
  "accepted_at": "2026-05-30T08:00:00Z"
}
```

非同期実行。進捗は 2.3 でポーリング。

### 2.2 実行一覧 — `GET /api/v1/checks`
クエリ: `?status=finished&from=2026-05-01&to=2026-05-31&limit=20`

```json
200 OK
{
  "items": [
    {
      "id": "cr_01HF...",
      "mode": "both",
      "trigger": "scheduled",      // manual / scheduled / api
      "status": "finished",        // running / finished / failed
      "started_at": "2026-05-30T08:00:00Z",
      "finished_at": "2026-05-30T08:00:12Z",
      "total": 6,
      "summary": { "OK": 3, "WARN": 1, "CRITICAL": 1, "URGENT": 0, "EXPIRED": 1, "ERROR": 0 }
    }
  ],
  "total": 1
}
```

### 2.3 実行詳細 — `GET /api/v1/checks/{id}`
→ `200` 上記 1 件 + 進捗（`progress: { done, total }`）

### 2.4 実行結果 — `GET /api/v1/checks/{id}/results`
クエリ: `?status=EXPIRED,URGENT&kind=ORIGIN`

```json
200 OK
{
  "items": [
    {
      "id": "res_01HF...",
      "check_run_id": "cr_01HF...",
      "target_id": "t_01HF...",
      "service_name": "s-bisco",
      "kind": "ORIGIN",
      "host": "origin.example.com",
      "port": 443,
      "sni": "www.s-bisco.jp",
      "status": "EXPIRED",
      "days_left": 0,
      "not_before": "2025-05-29T00:00:00Z",
      "not_after": "2026-05-29T00:00:00Z",
      "subject": "CN=origin.example.com",
      "issuer": "CN=Example CA",
      "dns_names": ["origin.example.com"],
      "error": "",
      "checked_at": "2026-05-30T08:00:05Z"
    }
  ],
  "total": 1
}
```

---

## 3. Results（横断検索）

### 3.1 最新結果横断 — `GET /api/v1/results`
各 target × kind の **最新**結果を返す（ダッシュボードの一覧用）。
クエリ: `?status=EXPIRED,URGENT,CRITICAL&owner=platform-team&kind=ORIGIN`

レスポンスは 2.4 と同形式。

---

## 4. Reports（レポート出力）

### 4.1 CSV — `GET /api/v1/checks/{id}/report.csv`
→ `200`、`Content-Type: text/csv`、CLI と同一カラム。

### 4.2 HTML — `GET /api/v1/checks/{id}/report.html`
→ `200`、`Content-Type: text/html`、CLI と同一の色分けレポート。

---

## 5. Notifications（通知ログ）

### 5.1 一覧 — `GET /api/v1/notifications`
クエリ: `?check_run_id=cr_01HF...`

```json
200 OK
{
  "items": [
    {
      "id": "ntf_01HF...",
      "check_run_id": "cr_01HF...",
      "channel": "teams",         // teams / email
      "status": "sent",           // sent / skipped / failed
      "target_count": 2,
      "summary": "EXPIRED: 1 / URGENT: 0 / CRITICAL: 1",
      "sent_at": "2026-05-30T08:00:13Z",
      "error": ""
    }
  ],
  "total": 1
}
```

---

## 6. ヘルスチェック

- `GET /healthz` → `200 {"status":"ok"}`
- `GET /readyz` → DB接続確認込み

---

## 7. エンドポイント一覧（早見表）

| メソッド | パス | 用途 |
|----------|------|------|
| GET | `/api/v1/targets` | 監視対象一覧 |
| POST | `/api/v1/targets` | 監視対象作成 |
| GET | `/api/v1/targets/{id}` | 監視対象取得 |
| PUT | `/api/v1/targets/{id}` | 監視対象更新 |
| DELETE | `/api/v1/targets/{id}` | 監視対象削除 |
| POST | `/api/v1/targets:import` | CSV一括取込 |
| POST | `/api/v1/checks` | チェック実行 |
| GET | `/api/v1/checks` | 実行一覧 |
| GET | `/api/v1/checks/{id}` | 実行詳細 |
| GET | `/api/v1/checks/{id}/results` | 実行結果 |
| GET | `/api/v1/checks/{id}/report.csv` | CSVレポート |
| GET | `/api/v1/checks/{id}/report.html` | HTMLレポート |
| GET | `/api/v1/results` | 最新結果横断検索 |
| GET | `/api/v1/notifications` | 通知ログ |
| GET | `/healthz` `/readyz` | 死活監視 |
