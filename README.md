# cert-watcher

Akamai 構成サイトの **Edge 証明書**（利用者 → Akamai Edge）と **Origin 証明書**（Akamai → Origin）の有効期限を一括取得し、期限切れ・期限接近・接続失敗を検知する CLI ツールです。

- Go 標準ライブラリのみで実装（外部依存なし）
- Windows / Linux / macOS で動作、単一バイナリ配布が可能
- 1 対象の接続失敗で全体が止まらず、対象ごとに `ERROR` として記録

> v0.2: 証明書期限取得 → CSV/HTML 出力 → Teams 通知、Edge/Origin 実行分離まで対応。

📐 設計ドキュメント（コンセプト・アーキテクチャ・API定義・ER図・テーブル設計・技術スタック）は [docs/](docs/README.md) を参照。

---

## 仕組み

各サービス（CSV の 1 行）について、以下の 2 系統をそれぞれ TLS 接続で確認します。

| 種別 | 接続先 | SNI |
|------|--------|-----|
| **Edge** | `edge_fqdn:443` | `edge_fqdn` |
| **Origin** | `origin_host:443` | `origin_sni` |

取得する証明書情報: Subject / Issuer / SAN（DNS Names）/ NotBefore / NotAfter / 残日数 / エラー内容。

> 証明書情報そのものを取得するため、TLS 検証は意図的にスキップしています（期限切れ・自己署名でも情報を取得し、期限判定は本ツールが独自に行います）。

---

## ビルド

```bash
cd cert-watcher
go build -o cert-watcher ./cmd/cert-watcher
```

### クロスコンパイル（単一バイナリ配布）

```bash
# Windows (amd64)
GOOS=windows GOARCH=amd64 go build -o cert-watcher.exe ./cmd/cert-watcher

# Linux (amd64)
GOOS=linux GOARCH=amd64 go build -o cert-watcher-linux ./cmd/cert-watcher

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o cert-watcher-mac ./cmd/cert-watcher
```

---

## 使い方

```bash
cert-watcher check --input config/targets.csv --output output/report.csv
```

台帳の形式だけを事前検証する場合:

```bash
cert-watcher validate \
  --sites config/sites.csv \
  --origins config/origins.secure.csv \
  --actions config/action_status.csv \
  --check both
```

`validate` は TLS 接続を行わず、CSVの必須列、重複、`service_id` の紐付けだけを確認します。

HTML レポートも出力する場合:

```bash
cert-watcher check \
  --input config/targets.csv \
  --output output/report.csv \
  --html output/report.html \
  --timeout 10s
```

ビルドせずに実行する場合:

```bash
go run ./cmd/cert-watcher check --input config/targets.csv --output output/report.csv --html output/report.html
```

### オプション

| フラグ | 既定値 | 説明 |
|--------|--------|------|
| `--input` | なし | 入力 CSV のパス（従来互換。`--sites/--origins` とは排他） |
| `--sites` | なし | サイト一覧 CSV のパス（`--input` の代替） |
| `--origins` | なし | Origin 情報 CSV のパス（`--check origin/both` では必須） |
| `--output` | なし | 詳細 CSV レポートのパス（`--report-root` なしでは必須） |
| `--html` | なし | HTML レポート出力先（指定時のみ出力） |
| `--public-output` | なし | 非エンジニア向け CSV レポート出力先（指定時のみ出力） |
| `--public-html` | なし | 非エンジニア向け HTML レポート出力先（指定時のみ出力） |
| `--summary-json` | なし | 実行サマリ JSON 出力先（指定時のみ出力） |
| `--report-root` | なし | 推奨フォルダ構成へ latest/history をまとめて出力するルート |
| `--actions` | なし | 対応管理 CSV のパス（指定時のみ public レポートへ反映） |
| `--config` | なし | 実行設定 JSON のパス（しきい値・タイムアウト。Secretは入れない） |
| `--check` | `both` | チェック対象 `edge` / `origin` / `both` |
| `--teams-webhook` | なし | Teams Incoming Webhook URL（指定時のみ通知） |
| `--timeout` | `10s` | 接続タイムアウト（例: `5s`, `1m`） |
| `--warn-days` | `30` | WARN しきい値（日数） |
| `--critical-days` | `14` | CRITICAL しきい値（日数） |
| `--urgent-days` | `7` | URGENT しきい値（日数） |

### 台帳検証（`validate`）

`validate` サブコマンドは、証明書チェック前に入力ファイルだけを検証します。

```bash
# Edge + Origin の台帳を検証
cert-watcher validate --sites config/sites.csv --origins config/origins.secure.csv --check both

# Edge のみなら sites.csv だけで検証可能
cert-watcher validate --sites config/sites.csv --check edge

# 従来の単一CSVも検証可能
cert-watcher validate --input config/targets.csv
```

検証する内容:

- 必須ヘッダの有無
- 空行スキップ
- `service_id` の重複
- `--check origin/both` 時の Origin 情報不足
- `origin_port` の数値範囲
- `action_status.csv` の `service_id + kind` 重複

### 実行設定JSON（任意）

`--config` を指定すると、しきい値やタイムアウトをファイルから読み込めます。
CLIで明示したフラグは設定ファイルより優先されます。

`settings.json`:

```json
{
  "check": "both",
  "timeout": "10s",
  "warn_days": 30,
  "critical_days": 14,
  "urgent_days": 7
}
```

SecretやWebhook URLは `settings.json` に書かないでください。Teams Webhook URL、Microsoft Graph client secret、AWS認証情報などは環境変数やAWS Secrets Managerで管理します。

### Edge / Origin の実行環境分離（`--check`）

Origin は SiteShield や Security Group の制限により、手元 PC からは到達できないことがあります。
`--check` で系統を分けて実行できるため、それぞれ到達可能な環境で回せます。

```bash
# 公開環境・手元PCから: Edge のみ
cert-watcher check --input config/targets.csv --output output/edge.csv --check edge

# 社内NW / AWS内（Lambda・EC2・ECS）から: Origin のみ
cert-watcher check --input config/targets.csv --output output/origin.csv --check origin
```

分離CSVで実行する場合:

```bash
# 非エンジニア向け sites.csv と管理者向け origins.secure.csv を結合して両方チェック
cert-watcher check \
  --sites config/sites.csv \
  --origins config/origins.secure.csv \
  --report-root output/CertWatcher \
  --actions config/action_status.csv \
  --config config/settings.json \
  --check both

# EdgeのみならOrigin情報なしでも実行可能
cert-watcher check \
  --sites config/sites.csv \
  --output output/edge.csv \
  --check edge
```

### Teams 通知（`--teams-webhook`）

`--teams-webhook` を指定すると、**ERROR / EXPIRED / URGENT / CRITICAL** のみを Teams に通知します
（WARN 以下は通知せず HTML/CSV で確認）。通知対象が 0 件なら送信しません。
通知の送信に失敗してもレポート出力は完了し、全体は停止しません。

Teams通知にはOrigin接続先、SNI、詳細エラーは出しません。詳細確認は管理者向けの詳細レポートで行います。

```bash
cert-watcher check --input config/targets.csv --output output/report.csv \
  --teams-webhook "https://outlook.office.com/webhook/xxxx"
```

> **注意**: Microsoft Teams の従来型 Incoming Webhook（Office 365 Connector）は廃止方向で、
> 現在は Power Automate「Workflows」が推奨されています。本ツールは従来型コネクタ互換の
> **MessageCard 形式**で送信します。Workflows（Adaptive Card 形式）へ移行する場合は
> `internal/notify/teams.go` のペイロード生成を差し替えてください。

### 終了コード

| コード | 意味 |
|--------|------|
| `0` | 正常（アラート無し） |
| `1` | `EXPIRED` / `URGENT` / `CRITICAL` / `ERROR` を 1 件以上検知 |
| `2` | 実行失敗（引数不正・CSV 読込失敗など） |

cron / CI で `exit code` を見て異常を検知できます。

---

## 入力 CSV フォーマット

### 単一CSV（従来互換）

ヘッダ必須。列の並び順は問わず、**ヘッダ名でマッピング**します。

```csv
service_name,edge_fqdn,origin_host,origin_sni,owner,vendor,notes
Example Site,example.com,example.com,example.com,platform-team,Akamai,メモ
```

| カラム | 説明 |
|--------|------|
| `service_name` | サービス名（識別ラベル） |
| `edge_fqdn` | Akamai Edge 側 FQDN（接続先 / SNI） |
| `origin_host` | Origin 接続先ホスト |
| `origin_sni` | Origin 接続時に提示する SNI |
| `owner` | 管理担当者 |
| `vendor` | ベンダー |
| `notes` | 備考 |

### 分離CSV（推奨）

非エンジニアが編集するサイト一覧と、管理者・Akamai担当者が編集するOrigin情報を分けられます。
`service_id` で結合し、チェック処理には従来と同じ `Target` として渡します。

`sites.csv`:

```csv
service_id,service_name,edge_fqdn,owner,vendor,notes,business_impact,enabled
corp,コーポレートサイト,www.example.com,情シス,Akamai,本番サイト,中,true
ec,ECサイト,shop.example.com,EC担当,Akamai,売上影響大,高,true
```

| カラム | 説明 |
|--------|------|
| `service_id` | サイト一覧とOrigin情報を紐付けるID |
| `service_name` | サービス名 |
| `edge_fqdn` | 利用者がアクセスする公開FQDN |
| `owner` | 社内担当者 |
| `vendor` | ベンダー |
| `notes` | 備考 |
| `business_impact` | 影響度（任意） |
| `enabled` | チェック対象にするか（任意。省略時は true） |

`origins.secure.csv`:

```csv
service_id,origin_host,origin_sni,origin_port,check_location,site_shield,entered_by,source,last_reviewed_at,notes_private
corp,origin-corp.example.internal,origin-corp.example.internal,443,AWS VPC内,yes,Akamai担当者,Akamai設定,2026-06-23,社内限定
ec,ec-origin.example.internal,ec-origin.example.internal,443,AWS VPC内,yes,制作ベンダー,ベンダー回答,2026-06-23,売上影響大
```

| カラム | 説明 |
|--------|------|
| `service_id` | `sites.csv` との紐付けID |
| `origin_host` | Origin 接続先ホスト |
| `origin_sni` | Origin 接続時に提示する SNI |
| `origin_port` | Origin 接続ポート（任意。省略時は 443） |
| `check_location` | 確認すべき実行場所（任意） |
| `site_shield` | Site Shield 有無（任意） |
| `entered_by` | 入力者（任意） |
| `source` | 情報源（任意） |
| `last_reviewed_at` | 最終確認日（任意） |
| `notes_private` | 管理者向けメモ（任意。レポートには出力しません） |

`--check origin` または `--check both` では `--origins` が必須です。
`--check edge` では `--sites` だけで実行できます。

### 対応管理CSV（任意）

`--actions` を指定すると、人間の対応状況を public レポートへ反映できます。
システム判定の `status` は上書きせず、対応状況は別列として扱います。

`action_status.csv`:

```csv
service_id,kind,action_status,action_owner,vendor_ticket,due_date,comment
ec,Origin,更新依頼済,EC担当,TICKET-123,2026-07-01,ベンダー確認中
corp,Edge,確認中,情シス,TICKET-124,2026-07-05,更新予定確認中
```

| カラム | 説明 |
|--------|------|
| `service_id` | `sites.csv` との紐付けID |
| `kind` | `Edge` または `Origin` |
| `action_status` | 未対応 / 確認中 / 更新依頼済 / 完了 など |
| `action_owner` | 対応者 |
| `vendor_ticket` | ベンダー問い合わせ番号 |
| `due_date` | 対応期限 |
| `comment` | 対応メモ |

---

## ステータス定義

| ステータス | 条件 |
|------------|------|
| `OK` | 残日数 31 日以上 |
| `WARN` | 残日数 30 日未満 |
| `CRITICAL` | 残日数 14 日未満 |
| `URGENT` | 残日数 7 日未満 |
| `EXPIRED` | 期限切れ |
| `ERROR` | 接続不可・証明書取得失敗 |

しきい値は `--warn-days` / `--critical-days` / `--urgent-days` で変更できます（判定は「未満」）。

レポートは重大度の高い順（`ERROR` / `EXPIRED` / `URGENT` / `CRITICAL` を上位 → `WARN` → `OK`）に並び、同ステータス内は残日数の少ない順です。

---

## 出力

### 詳細CSVレポート

カラム: `service_name, kind, host, port, sni, status, days_left, not_before, not_after, subject, issuer, dns_names, owner, vendor, notes, error`

- `kind`: `Edge` または `Origin`
- 日付は `YYYY-MM-DD` 形式
- `dns_names`（SAN）は `;` 区切り

### 詳細HTMLレポート

ステータスを色分けしたテーブルと、上部にステータス別件数サマリを表示する 1 ページの HTML（外部依存なし）。

詳細レポートは管理者向けです。`host`、`sni`、証明書 Subject/Issuer/SAN、詳細エラーを含むため、SharePoint 等では閲覧権限を絞ってください。

### public CSV / HTML レポート

`--public-output` / `--public-html` を指定すると、非エンジニア向けの簡易レポートも出力できます。

カラム: `service_name, kind, status, days_left, not_after, owner, vendor, action_status, action_owner, vendor_ticket, due_date, comment, error_summary`

public レポートには以下を出力しません。

- Origin 接続先 `host`
- 接続時の `sni`
- 証明書 Subject / Issuer / SAN
- 詳細エラー本文
- 管理者向け private メモ

`ERROR` の場合、詳細エラーの代わりに `接続または証明書取得に失敗` という概要だけを出します。

### summary JSON

`--summary-json` を指定すると、実行結果の件数サマリをJSONで出力できます。
接続先ホスト、SNI、証明書詳細、詳細エラーは含めません。

```json
{
  "generated_at": "2026-06-23T08:00:00Z",
  "total": 10,
  "alert": true,
  "alert_count": 2,
  "counts": {
    "ERROR": 1,
    "EXPIRED": 0,
    "URGENT": 0,
    "CRITICAL": 1,
    "WARN": 0,
    "OK": 8
  },
  "notify_targets": ["ERROR", "EXPIRED", "URGENT", "CRITICAL"]
}
```

### 推奨フォルダ構成への出力

`--report-root` を指定すると、SharePoint 配置を想定した構成へ private/public の latest と日別履歴をまとめて出力します。

```bash
cert-watcher check \
  --sites config/sites.csv \
  --origins config/origins.secure.csv \
  --actions config/action_status.csv \
  --report-root output/CertWatcher \
  --check both
```

生成される構成:

```text
output/CertWatcher/
  03_Report/
    summary.json
    public/
      latest.csv
      latest.html
    private/
      latest.csv
      latest.html
  04_History/
    2026-06-23/
      summary.json
      public.csv
      public.html
      private.csv
      private.html
```

`03_Report/public` は関係者向け、`03_Report/private` は管理者向けとして扱ってください。

---

## 定期実行

### Windows タスクスケジューラ

PowerShell（管理者）で、毎朝 8:00 に Edge をチェックする例:

```powershell
$action  = New-ScheduledTaskAction -Execute "C:\tools\cert-watcher.exe" `
  -Argument "check --input C:\tools\config\targets.csv --output C:\tools\output\report.csv --html C:\tools\output\report.html --check edge" `
  -WorkingDirectory "C:\tools"
$trigger = New-ScheduledTaskTrigger -Daily -At 8:00AM
Register-ScheduledTask -TaskName "cert-watcher" -Action $action -Trigger $trigger -Description "証明書期限監視"
```

GUI から登録する場合は「基本タスクの作成」→ プログラム `cert-watcher.exe`、引数に上記 `check ...` を指定します。

### Linux cron

毎朝 8:00 に実行する例（`crontab -e`）:

```cron
0 8 * * * cd /opt/cert-watcher && ./cert-watcher check --input config/targets.csv --output output/report.csv --html output/report.html --check origin >> /var/log/cert-watcher.log 2>&1
```

### Docker

```bash
# イメージをビルド
docker build -t cert-watcher .

# CSV と出力先をマウントして実行
docker run --rm \
  -v "$PWD/config:/app/config" \
  -v "$PWD/output:/app/output" \
  cert-watcher check --input config/targets.csv --output output/report.csv --html output/report.html
```

イメージは distroless ベースで、TLS 接続用の CA 証明書を同梱しています。

---

## 開発・テスト

```bash
go build ./...    # ビルド
go vet ./...      # 静的解析
go test ./...     # ユニットテスト
```

---

## ディレクトリ構成

```
cert-watcher/
  README.md
  Dockerfile
  .dockerignore
  go.mod
  config/targets.csv
  cmd/cert-watcher/main.go          # CLI エントリポイント
  internal/checker/tls_checker.go   # TLS 接続・証明書取得・CheckMode
  internal/model/target.go          # 入力 CSV / Target 定義
  internal/model/result.go          # 結果・ステータス判定
  internal/notify/teams.go          # Teams 通知（MessageCard）
  internal/report/csv_report.go     # CSV 出力
  internal/report/html_report.go    # HTML 出力
```
