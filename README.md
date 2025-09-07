# TikTok OAuth (Go + Echo)

Go 1.22 + Echo v4 を用いた TikTok OAuth v2 の最小実装です。Render へのデプロイを想定し、ヘルスチェックとタスク実行を用意しています。

- エンドポイント: `GET /auth/login`, `GET /auth/callback`, `GET /healthz`
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

## Render へのデプロイ例
- Build Command: `go build -o app ./cmd/server`
- Start Command: `./app`
- Health Check Path: `/healthz`
