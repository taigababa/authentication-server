# TikTok OAuth (Go + Echo)

Go 1.22 + Echo v4 を用いた TikTok OAuth v2 の最小実装です。Render へのデプロイを想定し、ヘルスチェックとタスク実行を用意しています。

- 主なエンドポイント:
  - `GET /` トップページ（`contents/index.html` を返却）
  - `GET /auth/login` TikTok ログイン開始
  - `GET /auth/callback` ログイン後のコールバック（HTMLでトークン/ユーザ表示、JSONも選択可）
  - `GET /healthz` ヘルスチェック
  - `GET /terms-of-service` 利用規約（`contents/terms_of_service.txt`）
  - `GET /privacy-policy` プライバシーポリシー（`contents/privacy_policy.txt`）
  - `GET /:filename` 署名ファイル配信（`contents/signature/:filename` のみ）

- ビルド/起動: `go build -o app ./cmd/server` → `./app`

## 環境変数
- `TIKTOK_CLIENT_KEY`: TikTok Developer Portal の Client Key（client_id ではなく client_key）
- `TIKTOK_CLIENT_SECRET`: Client Secret
- `OAUTH_REDIRECT_URI`: リダイレクトURI（TikTok側の設定と完全一致が必要）
- `TIKTOK_SCOPE`: 省略時は `user.info.basic`

## 実行方法（go-task）
Taskfile.yaml を使ってコマンドをまとめています。

- `task tidy` — 依存の取得（`go mod tidy`）
- `task run` — サーバ起動（`/healthz` が `ok` を返せば正常）
- `task build` — バイナリ出力（`./app`）
- `task test` — テスト実行
- `task lint` — `go vet`

Gitpod を使う場合は `.gitpod.yml` にタスクが配線済みで、ポート `3000` が自動でプレビュー表示されます。

### go-task のインストール（ローカル）
- macOS: `brew install go-task/tap/go-task`
- Go ツールチェーン経由: `go install github.com/go-task/task/v3/cmd/task@latest`

## 実装メモ
- 認可リクエストとトークン交換の両方で `redirect_uri` を一致させる必要があります。
- `client_key` を使用します（`client_id` ではありません）。
- `state` の生成は実装済みですが、最小構成につきサーバ側での検証は省略しています（本番では CSRF 対策として要検証）。
- 外部HTTPのタイムアウトは 10s に設定しています。

### コールバックの表示仕様
- `/auth/callback` は成功時、`contents/callback.html` テンプレートを用いて以下を表示します。
  - アクセストークン（`access_token`）
  - リフレッシュトークン（`refresh_token`）
  - ユーザのアバター（`avatar_url`）
  - ユーザの表示名（`display_name`）
- JSON が必要な場合はリクエストに `Accept: application/json` を付与、またはクエリ `?format=json` を指定してください。

### 署名ファイル（ファイル名可変）の公開
- `contents/signature/` 配下にファイルを配置すると、`/<ファイル名>` でアクセスできます。
  - 例: `contents/signature/tiktokDuXXXX.txt` → `GET /tiktokDuXXXX.txt`
- セキュリティのため、動的配信は `contents/signature/` のみを参照します（`..` や `/` を含むパスは拒否）。
- 既存の静的ルート（`/healthz`, `/auth/*`, `/terms-of-service`, `/privacy-policy`, `/`）が動的ルートより優先されます。

### 静的コンテンツの取り扱い
- トップページ: `contents/index.html`
- 利用規約: `contents/terms_of_service.txt`
- プライバシーポリシー: `contents/privacy_policy.txt`
- コールバック表示テンプレート: `contents/callback.html`

注意: これらのファイルはアプリのカレントディレクトリからの相対パスで読み込みます。サーバはリポジトリのルートで起動してください（例: ルートで `./app` 実行）。

## Render へのデプロイ例
- Build Command: `go build -o app ./cmd/server`
- Start Command: `./app`
- Health Check Path: `/healthz`
