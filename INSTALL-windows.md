# cert-watcher Windows 導入手順（会社PC向け）

会社の Windows PC / サーバに cert-watcher を配置し、毎日自動で証明書期限をチェックする状態にするまでの手順です。

- 対象OS: Windows 10 / 11（Server も同手順）
- 必要権限: タスクスケジューラ登録時のみ管理者権限
- 前提: 監視対象へ **443/TCP に直接到達できる**こと（後述のトラブルシューティング参照）

---

## 1. 入手と展開

1. GitHub Releases から `cert-watcher-vX.Y.Z-windows-amd64.zip` をダウンロードする
2. **展開する前に** zip のブロックを解除する（Mark of the Web の除去）
   - エクスプローラ: zip を右クリック → プロパティ → 「許可する」にチェック → OK
   - または PowerShell: `Unblock-File .\cert-watcher-vX.Y.Z-windows-amd64.zip`
   - 先に解除しておくと、展開された全ファイルへブロックが伝播しません。展開後に気づいた場合は `Get-ChildItem -Recurse C:\tools\cert-watcher | Unblock-File`
3. `C:\tools\cert-watcher\` へ展開する（以下このパス前提。変える場合は読み替え）

## 2. 台帳の作成

同梱の `config\` はすべてサンプルです。実データに差し替えます。

| ファイル | 対応 |
|----------|------|
| `config\sites.csv` | 実際の監視対象（公開FQDN・担当者）に書き換える |
| `config\origins.secure.example.csv` | Origin もチェックする場合のみ `origins.secure.csv` にコピーして記入 |
| `config\settings.example.json` | しきい値を変える場合のみ `settings.json` にコピーして編集 |
| `config\action_status.example.csv` | 対応状況を public レポートへ出す場合のみ `action_status.csv` にコピー |

> **重要**: 実データ入りの CSV（特に `origins.secure.csv`）は社内限定情報です。GitHub・個人PC・社外へ持ち出さないでください。リポジトリの `.gitignore` 運用とは別に、この展開フォルダごと社外に出さない運用にしてください。

## 3. 手動実行で動作確認

PowerShell で:

```powershell
cd C:\tools\cert-watcher
.\cert-watcher.exe check --sites config\sites.csv --report-root output\CertWatcher --check edge
```

- `output\CertWatcher\03_Report\` にレポートが生成されれば成功
- 終了コード: `0`=アラート無し / `1`=要対応あり / `2`=実行失敗（`$LASTEXITCODE` で確認）

うまくいかない場合:

- **SmartScreen「WindowsによってPCが保護されました」** → 「詳細情報」→「実行」。手順1のブロック解除漏れが典型原因。組織ポリシーで実行自体が禁止されている場合は情シスへ相談（ソースからのビルドという代替もあります）
- **全対象が ERROR になる** → 本ツールは 443 へ直接 TLS 接続します（**プロキシ非対応**）。社内NWが外向き通信をプロキシ強制している場合は到達できないため、プロキシ例外の申請かサーバ設置場所の変更を情シスへ相談してください

## 4. 定期実行の登録

管理者として起動した PowerShell で:

```powershell
cd C:\tools\cert-watcher
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\register-task.ps1
```

これで毎朝 8:00 に Edge チェックが登録されます。オプション:

```powershell
# 時刻変更・Origin込み・配置先変更
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\register-task.ps1 -Time 07:30 -CheckMode both
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\register-task.ps1 -InstallDir D:\ops\cert-watcher

# 登録解除
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\register-task.ps1 -Unregister
```

制約・注意:

- **ログオン中のみ実行**: 資格情報を保存しない登録のため、タスクはこのユーザーがログオンしている間だけ動きます（画面ロック中はOK、サインアウト中は動きません）。PCを点けっぱなしにできない場合や無人サーバで回したい場合は、サービスアカウントでの登録を情シスと相談してください
- 実行時刻に PC が起きていなかった場合は、次に使える状態になった時点で実行されます（`StartWhenAvailable`）
- GPO で ExecutionPolicy が固定されておりスクリプトを実行できない場合は、README「定期実行」の GUI 手順（タスクスケジューラの「基本タスクの作成」）で同内容を手動登録してください

## 5. レポートの共有（任意）

- `--report-root` を SharePoint 同期フォルダ配下（例: `C:\Users\<user>\<会社名>\<サイト> - Documents\CertWatcher`）にすると、関係者がブラウザでレポートを閲覧できます
- `03_Report\public` は関係者向け、`03_Report\private` は管理者向けです。private には Origin ホスト名や詳細エラーが含まれるため、閲覧権限を絞ってください
- Teams 通知を使う場合はタスクの引数に `--teams-webhook "<URL>"` を追加します。**タスクの引数は同じPCの他ユーザーから見える**ため、Webhook URL の扱いは社内ルールに従ってください

## 6. バージョンアップ

新しい zip から `cert-watcher.exe` だけを差し替えます（`config\` と `output\` はそのまま）。差し替え前に手順1のブロック解除を忘れずに。

## 7. アンインストール

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\register-task.ps1 -Unregister
Remove-Item -Recurse C:\tools\cert-watcher
```
