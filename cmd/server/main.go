package main

import (
    "context"
    "net/http"
    "os"
    "time"
    "path/filepath"

    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
    "github.com/labstack/gommon/log"

    "tiktok-oauth/internal/config"
    "tiktok-oauth/internal/domain/oauth"
    httpiface "tiktok-oauth/internal/interface/http"
    "tiktok-oauth/internal/infrastructure/store"
    "tiktok-oauth/internal/infrastructure/tiktok"
)

func main() {
    cfg := config.Load()

    e := echo.New()
    e.HideBanner = true
    e.Logger.SetLevel(log.INFO)
    e.Use(middleware.Recover())
    e.Use(middleware.Logger())

    // Top page
    e.GET("/", func(c echo.Context) error {
        return c.HTML(http.StatusOK, indexHTML)
    })

    // Healthz
    e.GET("/healthz", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })

    // Terms of Service (read from contents/terms_of_service.txt)
    e.GET("/terms-of-service", func(c echo.Context) error {
        path := filepath.Join("contents", "terms_of_service.txt")
        b, err := os.ReadFile(path)
        if err != nil {
            c.Logger().Errorf("failed to read terms: %v", err)
            return c.String(http.StatusInternalServerError, "terms_of_service.txt not found")
        }
        return c.String(http.StatusOK, string(b))
    })

    httpClient := &http.Client{Timeout: 10 * time.Second}
    client := &tiktok.Client{ClientKey: cfg.ClientKey, ClientSecret: cfg.ClientSecret, HTTP: httpClient}
    mem := &store.Memory{}
    uc := oauth.NewUseCase(client, mem, cfg.Scope)

    h := &httpiface.Handler{UC: uc, RedirectURI: cfg.RedirectURI}
    e.GET("/auth/login", h.Login)
    e.GET("/auth/callback", h.Callback)

    addr := ":3000"
    if p := os.Getenv("PORT"); p != "" {
        addr = ":" + p
    }
    e.Logger.Infof("starting server on %s", addr)
    if err := e.Start(addr); err != nil {
        e.Logger.Fatalf("server error: %v", err)
    }
}

// silence unused import warnings for context in some environments
var _ = context.Background

const indexHTML = `<!doctype html>
<html lang="ja">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>TikTok OAuth Demo</title>
    <style>
      :root { color-scheme: light dark; }
      body { font-family: -apple-system, system-ui, Segoe UI, Roboto, Helvetica, Arial, sans-serif; margin: 2rem; line-height: 1.6; }
      .box { max-width: 720px; margin: 0 auto; padding: 1.25rem 1.5rem; border: 1px solid #ccc; border-radius: 12px; }
      h1 { margin-top: 0; font-size: 1.6rem; }
      .actions { margin-top: 1rem; }
      .btn { display: inline-block; background: #000; color: #fff; text-decoration: none; padding: 0.65rem 1rem; border-radius: 8px; font-weight: 600; }
      .links { margin-top: 0.75rem; font-size: 0.95rem; }
      code { background: rgba(127,127,127,0.15); padding: 0.1rem 0.35rem; border-radius: 6px; }
    </style>
  </head>
  <body>
    <div class="box">
      <h1>TikTok OAuth Demo</h1>
      <p>このサーバは TikTok OAuth v2 のサンプル実装です。</p>
      <div class="actions">
        <a class="btn" href="/auth/login">Login with TikTok</a>
      </div>
      <div class="links">
        <p>
          エンドポイント: <code>GET /auth/login</code>, <code>GET /auth/callback</code>, <code>GET /healthz</code>, <code>GET /terms-of-service</code>
        </p>
        <p>
          利用規約は <a href="/terms-of-service">/terms-of-service</a> から確認できます。
        </p>
      </div>
    </div>
  </body>
</html>`
