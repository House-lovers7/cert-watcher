# システム設計図 — Akamai-Origin Certificate Watcher

現状の CLI を「コアエンジン」とし、その周辺に API / DB / スケジューラ / 通知を配置した **目標アーキテクチャ** を示す。

---

## 1. 全体構成図

```mermaid
flowchart TB
    subgraph User["利用者"]
        OP[運用担当者]
        WEB[Webブラウザ/ダッシュボード<br/>※将来構想]
    end

    subgraph Sched["スケジューラ"]
        EB[AWS EventBridge<br/>cron実行]
    end

    subgraph App["cert-watcher アプリ"]
        API[API サーバ<br/>Go]
        ENGINE[チェックエンジン<br/>internal/checker]
        NOTIFY[通知<br/>internal/notify]
        REPORT[レポート生成<br/>internal/report]
    end

    subgraph Exec["実行環境（到達性で分離）"]
        EDGEJOB[Edge チェック<br/>公開環境/Lambda]
        ORIGINJOB[Origin チェック<br/>VPC内 ECS/EC2]
    end

    subgraph Data["データストア"]
        DB[(PostgreSQL)]
        FILES[CSV / HTML<br/>レポート]
    end

    subgraph External["監視対象 & 連携先"]
        EDGE[Akamai Edge :443]
        ORIGIN[Origin :443]
        TEAMS[Microsoft Teams]
        SES[Amazon SES<br/>※任意]
    end

    OP -->|targets登録/結果閲覧| API
    WEB -.-> API
    EB -->|定期トリガ| API
    API --> ENGINE
    ENGINE --> EDGEJOB
    ENGINE --> ORIGINJOB
    EDGEJOB -->|TLS handshake| EDGE
    ORIGINJOB -->|TLS handshake| ORIGIN
    ENGINE --> DB
    API --> REPORT
    REPORT --> FILES
    API --> NOTIFY
    NOTIFY --> TEAMS
    NOTIFY -.-> SES
    API --> DB
```

---

## 2. コンポーネント責務

| コンポーネント | 責務 | 現状の対応物 |
|----------------|------|--------------|
| **チェックエンジン** | TLS接続して証明書取得・期限判定。Edge/Origin の系統分離 | `internal/checker` ✅ |
| **レポート生成** | CSV / HTML レポート出力 | `internal/report` ✅ |
| **通知** | ERROR/EXPIRED/URGENT/CRITICAL を Teams 通知 | `internal/notify` ✅ |
| **モデル** | Target / 結果 / ステータス判定 | `internal/model` ✅ |
| **CLI** | コマンド実行・フラグ処理 | `cmd/cert-watcher` ✅ |
| **API サーバ** | targets管理・チェック実行・結果配信 | ⬜ 将来（`internal/api`） |
| **永続化** | targets・実行履歴・結果・通知ログを保存 | ⬜ 将来（PostgreSQL） |
| **スケジューラ** | 定期実行トリガ | cron/タスクスケジューラ ✅ / EventBridge ⬜ |

---

## 3. データフロー（1回のチェック）

```mermaid
sequenceDiagram
    participant T as トリガ<br/>(CLI/API/EventBridge)
    participant API as API/CLI
    participant E as チェックエンジン
    participant X as Edge/Origin :443
    participant DB as PostgreSQL
    participant N as 通知(Teams)
    participant R as レポート

    T->>API: チェック実行 (mode, targets)
    API->>DB: check_run 作成 (running)
    loop 各 target × 系統
        API->>E: CheckTarget(target, mode)
        E->>X: TLS handshake (SNI指定)
        alt 成功
            X-->>E: 証明書 (NotAfter等)
            E->>E: 残日数→ステータス判定
        else 接続失敗
            E-->>E: ERROR として記録
        end
        E->>DB: cert_result 保存
    end
    API->>DB: check_run 更新 (finished, 集計)
    API->>R: CSV/HTML 生成
    API->>N: ERROR/EXPIRED/URGENT/CRITICAL を通知
    N-->>DB: notification ログ保存
```

---

## 4. Edge / Origin 実行環境の分離

Origin は **SiteShield の許可IP・Security Group・社内NW** などにより、任意の場所からは到達できないことがある。
そのため `--check`（API では `mode`）で系統を分け、それぞれ到達可能な実行環境で動かす。

```mermaid
flowchart LR
    subgraph Public["公開環境 / 手元PC / Lambda"]
        EJ[Edge チェック<br/>--check edge]
    end
    subgraph Internal["社内NW / AWS VPC内 ECS・EC2"]
        OJ[Origin チェック<br/>--check origin]
    end
    EJ -->|公開FQDN:443| AK[Akamai Edge]
    OJ -->|origin_host:443<br/>SNI=origin_sni| OG[Origin]
    EJ --> DB[(共有DB)]
    OJ --> DB
```

- **Edge チェック**: 公開 FQDN へ。どこからでも到達可能。
- **Origin チェック**: Origin への到達経路を持つ環境（VPC内など）で実行。
- 双方の結果を同一 `check_run` / 同一 DB に集約してレポート化する。

---

## 5. デプロイ構成（AWS 例）

```mermaid
flowchart TB
    EB[EventBridge Scheduler] --> LMB[Lambda: Edge チェック]
    EB --> FAR[ECS Fargate: Origin チェック<br/>VPC内]
    LMB --> RDS[(RDS PostgreSQL)]
    FAR --> RDS
    LMB --> TEAMS[Teams Webhook]
    FAR --> TEAMS
    RDS --> APISRV[API/ダッシュボード<br/>ECS/Lambda]
    OP[運用者] --> APISRV
```

- 小規模（〜数百サイト）なら **Lambda + EventBridge** でほぼ無料枠内。
- Origin が VPC 内のみ到達可なら **ECS Fargate（VPC内）** で Origin チェックを実行。
- 状態は **RDS PostgreSQL** に集約。

---

## 6. 設計上の重要な決定

| 決定 | 理由 |
|------|------|
| TLS検証は **InsecureSkipVerify=true** | 期限切れ・自己署名でも「証明書情報そのもの」を取得するため。期限判定は自前で実施 |
| 1対象失敗で全体停止しない | 監視ツールとしての堅牢性。失敗は `ERROR` として記録し継続 |
| Edge/Origin の実行環境を分離可能に | Origin の到達性制約（SiteShield/VPC）に対応 |
| 通知は ERROR/EXPIRED/URGENT/CRITICAL のみ | 通知過多を防ぎ、要対応だけを届ける |
| 自動更新はしない | 誤更新による事故影響が大きいため、検出・確認に専念 |
