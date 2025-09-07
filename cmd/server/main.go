package main

import (
    "context"
    "net/http"
    "os"
    "time"

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

    // Healthz
    e.GET("/healthz", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })

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

