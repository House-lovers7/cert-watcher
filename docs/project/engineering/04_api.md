<!-- generated-by: scripts/generate_engineering_docs.py -->
# certificate_chekcer — API定義書

> 生成日: 2026-07-15 / 対象: `certificate_chekcer` / 確度: [低]
> 実装・manifest・既存資料の静的棚卸しに基づく。外部サービスの稼働状態と本番構成は未検証。

## 正典とこの文書の関係

> [高] API契約の正典は `cert-watcher/docs/04_api_spec.md`。以下の自動検出台帳は実装探索と差分発見の補助であり、別の契約正典として扱わない。

- 正典から抽出: 15 method/path
- 実装から静的検出: 0 method/path
- 正典のみ: `GET /api/v1/targets`, `POST /api/v1/targets`, `GET /api/v1/targets/{id}`, `PUT /api/v1/targets/{id}`, `DELETE /api/v1/targets/{id}`, `POST /api/v1/checks`, `GET /api/v1/checks`, `GET /api/v1/checks/{id}`, `GET /api/v1/checks/{id}/results`, `GET /api/v1/results`, `GET /api/v1/checks/{id}/report.csv`, `GET /api/v1/checks/{id}/report.html`, `GET /api/v1/notifications`, `GET /healthz`, `GET /readyz`
- 実装検出のみ: なし
- 差分がある場合は、正典を先に読み、route登録・mount/prefix・動的生成を確認してから更新する。


## Public interface inventory

- CLI/script: CLI script未検出
- entrypoint: `cert-watcher/cmd/cert-watcher/main.go`
- HTTP endpoints: 0

## 検出したAPI

API endpointは静的検査で未検出。CLI/library/静的サイトの可能性がある。APIを追加する場合はOpenAPIまたは同等のschemaを正典にする。

## API所有境界

- API所有directory未検出

## 実装から確認できた追加契約

- 追加のmultipart・上限・副作用signalは静的検出できず。実装とcontract testを確認する。

## CLI契約

- argparse/commander等のsubcommand・exit-code契約は静的検出できず。

## 変更時の実務チェック

- caller: UI caller未検出。CLI/job/external callerを検索
- schema: route内inline validationだけでなく共有schema・型・OpenAPIの有無を確認する。
- auth: `未確認` のendpointは公開を意味しない。middleware、gateway、provider側設定も確認する。
- error: 表中にstatus signalがないhandlerは、成功/入力/権限/依存障害の契約をtestで固定する。
- write: POST/PUT/PATCH/DELETEは冪等性、重複retry、監査ログ、rollbackを確認する。

## 未確認

- 動的に登録されるroute、gateway rewrite、provider callback、production側rate limit。
- request/responseの完全なfield定義は、表の実装pathと共有schemaを正典として確認する。
