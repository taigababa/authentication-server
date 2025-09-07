# instruction.md — TikTok OAuth (Go + Echo on Render)

## プロジェクトの目的
- **Go + Echo** で **TikTok OAuth v2** の最小実装をクリーンアーキテクチャで構築する。
- エンドポイントは **/auth/login**（リダイレクト）と **/auth/callback**（トークン交換）の2つを中核にする。
- 認可後にユーザーの基本情報（`user.info.basic`）を取得して JSON で返す最小 API を実装。
- **Render.com** へデプロイ可能な構成（ヘルスチェック、ビルド/起動コマンド、環境変数）を用意。

## 仕様（必読）
- 認可エンドポイント（v2）: `https://www.tiktok.com/v2/auth/authorize/`
  - クエリ: `client_key`, `response_type=code`, `scope`, `redirect_uri`, `state`
- トークンエンドポイント（v2）: `https://open.tiktokapis.com/v2/oauth/token/`
  - フォーム: `client_key`, `client_secret`, `code`, `grant_type=authorization_code`, **`redirect_uri` をボディに含める**
- ユーザー情報 API（v2）: `https://open.tiktokapis.com/v2/user/info/`
  - ヘッダ: `Authorization: Bearer <access_token>`
  - クエリ: `fields=open_id,display_name,avatar_url` など
- **client_id ではなく `client_key`** を使用する点に注意。
- **redirect_uri は認可リクエストとトークン交換で完全一致**させる（Render の本番 URL を登録・使用）。
- スコープは最小構成で `user.info.basic`。

## 非機能要件
- フレームワーク: Go 1.22+ / Echo v4
- クリーンアーキテクチャのレイヤリング
- ログは構造化（最低限：レベル/メッセージ/エラー）
- エラーハンドリングは **HTTP ステータス + JSON** で返却
- Lint/Format（`gofmt`, `go vet`, `staticcheck`）を通す
- 単体テスト（主要ユースケース/クライアントをモック）

## ディレクトリ構成（生成指示）
```
/tiktok-oauth
  /cmd/server/main.go
  /internal/config/config.go
  /internal/domain/oauth/entity.go
  /internal/domain/oauth/usecase.go
  /internal/interface/http/handler.go
  /internal/infrastructure/tiktok/client.go
  /internal/infrastructure/store/memory.go
  /internal/pkg/httpx/response.go
  go.mod
  Makefile
  README.md
  instruction.md  <-- 本書
```

## 環境変数（Render の Dashboard で設定）
- `TIKTOK_CLIENT_KEY`（= Client Key）
- `TIKTOK_CLIENT_SECRET`
- `OAUTH_REDIRECT_URI`（例：`https://<app>.onrender.com/auth/callback`）
- `TIKTOK_SCOPE`（未設定時は `user.info.basic` を既定に）
- （任意）`PORT`（Render では自動注入されることがある）

## ルーティング（要実装）
- `GET /healthz` … 200 `"ok"`
- `GET /auth/login`
  - ランダム `state` を生成（16バイト以上/hex）。（最小構成はメモリ保持省略可）
  - TikTok 認可 URL に **302** でリダイレクト
- `GET /auth/callback`
  - `error` があれば 400 JSON
  - `code` が無ければ 400 JSON
  - トークン交換（`redirect_uri` をボディに含める）
  - 取得した `access_token` で `user/info` を呼び、**下記 JSON** を 200 で返却

### 返却 JSON（例）
```json
{
  "token": {
    "access_token": "...",
    "refresh_token": "...",
    "expires_in": 86400,
    "open_id": "xxxx",
    "scope": "user.info.basic",
    "token_type": "Bearer"
  },
  "user": {
    "data": { "...": "..." }, 
    "error": null
  }
}
```

## ドメイン層（インターフェース/エンティティ）
```go
// internal/domain/oauth/entity.go
package oauth

type Token struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	TokenType    string
	Scope        string
	OpenID       string
}
```

```go
// internal/domain/oauth/usecase.go
package oauth

import "context"

type Store interface {
	Save(ctx context.Context, t Token) error
}

type TikTokClient interface {
	AuthURL(state, redirectURI, scope string) string
	Exchange(ctx context.Context, code, redirectURI string) (Token, error)
	GetUserInfo(ctx context.Context, accessToken string, fields []string) (map[string]any, error)
}

type UseCase struct {
	client TikTokClient
	store  Store
	scope  string
}

func NewUseCase(c TikTokClient, s Store, scope string) *UseCase
func (u *UseCase) LoginURL(state, redirectURI string) string
func (u *UseCase) Callback(ctx context.Context, code, redirectURI string) (Token, error)
```

## インフラ層：TikTok クライアント（仕様どおり実装）
```go
// internal/infrastructure/tiktok/client.go
package tiktok

const (
	AuthEndpoint  = "https://www.tiktok.com/v2/auth/authorize/"
	TokenEndpoint = "https://open.tiktokapis.com/v2/oauth/token/"
	UserInfoURL   = "https://open.tiktokapis.com/v2/user/info/"
)

type Client struct {
	ClientKey    string
	ClientSecret string
	HTTP         *http.Client
}

// 必須：AuthURL, Exchange, GetUserInfo を上記仕様で実装。
// 注意：Exchange は application/x-www-form-urlencoded で POST。
//       body に redirect_uri を含める。
//       エラーは HTTP ステータスおよび TikTok のエラーボディを考慮して返す。
```

## インフラ層：ストア（デモ用メモリ）
```go
// internal/infrastructure/store/memory.go
package store

type Memory struct { /* mutex + oauth.Token を1つ保持 */ }
// Save(ctx, t) を実装（No-Op でも可）
```

## インターフェース層：Echo ハンドラ
```go
// internal/interface/http/handler.go
package httpiface

type Handler struct {
	UC          *oauth.UseCase
	RedirectURI string
}

func (h *Handler) Login(c echo.Context) error
func (h *Handler) Callback(c echo.Context) error
```

## 設定ローダ
```go
// internal/config/config.go
type Config struct {
	ClientKey    string
	ClientSecret string
	RedirectURI  string
	Scope        string
}
func Load() Config
```

## HTTP 共通レスポンス
```go
// internal/pkg/httpx/response.go
package httpx

type ErrorResponse struct { Message string `json:"message"`; Detail any `json:"detail,omitempty"` }
func JSONError(c echo.Context, code int, msg string, detail any) error
```

## エントリポイント / 起動
```go
// cmd/server/main.go
// - Echo を起動
// - ルーティング登録（/healthz, /auth/login, /auth/callback）
// - HTTPクライアントはタイムアウト設定
// - PORT 環境変数があれば利用、無ければ :3000
```

## Makefile（生成指示）
```Makefile
.PHONY: run build test lint

run:
	go run ./cmd/server

build:
	go build -o app ./cmd/server

test:
	go test ./...

lint:
	go vet ./...
```

## go.mod（生成指示）
- モジュール名は `tiktok-oauth` か、ユーザーの GitHub パスに合わせる。
- 依存：`github.com/labstack/echo/v4`, `github.com/labstack/gommon/log` など。

## ロギング方針
- 重要イベント（受信/リダイレクト/トークン交換成功or失敗/外部API失敗）を `INFO/ERROR` で出力。
- PII/シークレットはログに出さない。`code` は原則ログに残さない。

## セキュリティ
- **state**: ランダム値を生成。最小構成では検証省略可だが、実運用は **CSRF 対策として検証必須**（サーバー側に保存/照合）。
- **PKCE**: 任意（将来拡張）。`code_verifier`/`code_challenge` のサポートを追加しやすい設計に。
- HTTPS 前提（ローカルは ngrok 等）。

## Render デプロイ要件
- **Build Command**: `go build -o app ./cmd/server`
- **Start Command**: `./app`
- **Health Check Path**: `/healthz`
- **環境変数**: 本書「環境変数」参照

## 動作確認手順（curl/ブラウザ）
1. `GET /healthz` → 200 `"ok"`
2. ブラウザで `https://<host>/auth/login` へ → TikTok 認可画面へ遷移
3. 認可後 `GET /auth/callback?code=...` → JSON（token & user）

## テスト（生成指示）
- `internal/domain/oauth/usecase.go` のユニットテスト：`TikTokClient` をモック化
  - 正常系：`LoginURL` が期待どおりの URL を返す
  - 正常系：`Callback` が `Exchange` 結果を返却
  - 異常系：`Exchange` エラーの伝播
- `client.go` は HTTP ラウンドトリップをモックし、ステータス異常/JSON 異常を検証。

## コーディング規約
- Go 公式スタイル (`gofmt`) に準拠
- 関数は短く、責務を明確化
- 外部 API の URL/パラメータは **定数化** し、コメントで根拠を残す
- 失敗時のメッセージは利用者が原因を推定できる情報を含めるが、内部情報は漏らさない

## 受け入れ条件（Done の定義）
- ローカルで `/auth/login`→TikTok 認可→`/auth/callback`→トークン取得→ユーザ情報取得が成功
- Render にデプロイし、本番 URL で同様のフローが通る
- `user.info.basic` スコープで **open_id** と **display_name** の取得を確認
- Lint/Tests が通る

---

## Codex への具体的プロンプト例（貼り付け用）

**1) スキャフォールド**
> Go 1.22 と Echo v4 を用いて、上記のディレクトリ構成を作り、空ファイルではなく最小限の動作コードを含めてください。`/healthz` は 200 "ok" を返すこと。Makefile も作成してください。

**2) TikTok クライアント実装**
> `internal/infrastructure/tiktok/client.go` に、AuthURL/Exchange/GetUserInfo を仕様どおり実装してください。Exchange は `application/x-www-form-urlencoded` を使い、body に `redirect_uri` を含めてください。HTTP タイムアウトを 10s に設定し、エラー時は HTTP ステータスを含むエラーを返してください。

**3) UseCase / Handler 実装**
> UseCase と Echo ハンドラを実装し、`GET /auth/login` で認可 URL に 302 リダイレクト、`GET /auth/callback` でトークン交換とユーザ情報取得を行い、指定の JSON 形式で返却してください。状態はメモリストアに保存（No-Op 可）とします。

**4) 設定ローダ**
> `internal/config/config.go` を実装し、環境変数 `TIKTOK_CLIENT_KEY`, `TIKTOK_CLIENT_SECRET`, `OAUTH_REDIRECT_URI`, `TIKTOK_SCOPE` を読み込み、`TIKTOK_SCOPE` 未設定時は `user.info.basic` を既定にしてください。

**5) テスト生成**
> UseCase のユニットテストを作成してください。TikTokClient をモックし、正常系/異常系を検証します。`go test ./...` で通るようにしてください。
