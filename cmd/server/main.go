package main

import (
    "net/http"
    "os"
    "time"
    "path/filepath"
    "strings"

    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
    "github.com/labstack/gommon/log"

    "tiktok-oauth/internal/config"
    "tiktok-oauth/internal/domain/oauth"
    httpiface "tiktok-oauth/internal/interface/http"
    "tiktok-oauth/internal/infrastructure/store"
    "tiktok-oauth/internal/infrastructure/tiktok"
    "tiktok-oauth/internal/pkg/logging"
)

func main() {
    cfg := config.Load()

    e := echo.New()
    e.HideBanner = true
    // Split stdout/stderr: Info/Warn/Debug -> stdout, Error/Fatal/Panic -> stderr
    e.Logger = logging.NewSplitLogger()
    e.Logger.SetLevel(log.INFO)
    e.Use(middleware.Recover())
    // Access logs to stdout explicitly
    e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{Output: os.Stdout}))

    // Top page (read from contents/index.html)
    e.GET("/", func(c echo.Context) error {
        path := filepath.Join("contents", "index.html")
        b, err := os.ReadFile(path)
        if err != nil {
            c.Logger().Errorf("failed to read index.html: %v", err)
            return c.String(http.StatusInternalServerError, "index.html not found")
        }
        return c.HTML(http.StatusOK, string(b))
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

    // Privacy Policy (read from contents/privacy_policy.txt)
    e.GET("/privacy-policy", func(c echo.Context) error {
        path := filepath.Join("contents", "privacy_policy.txt")
        b, err := os.ReadFile(path)
        if err != nil {
            c.Logger().Errorf("failed to read privacy policy: %v", err)
            return c.String(http.StatusInternalServerError, "privacy_policy.txt not found")
        }
        return c.String(http.StatusOK, string(b))
    })

    // (removed) specific TikTok sign route; now served by "/:filename"

    httpClient := &http.Client{Timeout: 10 * time.Second}
    client := &tiktok.Client{ClientKey: cfg.ClientKey, ClientSecret: cfg.ClientSecret, HTTP: httpClient}
    mem := &store.Memory{}
    uc := oauth.NewUseCase(client, mem, cfg.Scope)

    h := &httpiface.Handler{UC: uc, RedirectURI: cfg.RedirectURI}
    e.GET("/auth/login", h.Login)
    e.GET("/auth/callback", h.Callback)

    // Dynamic file serving: GET /:filename -> contents/signature/:filename
    e.GET("/:filename", func(c echo.Context) error {
        name := c.Param("filename")
        if name == "" || strings.Contains(name, "..") || strings.ContainsAny(name, "/\\") {
            return c.String(http.StatusBadRequest, "invalid filename")
        }
        path := filepath.Join("contents", "signature", name)
        b, err := os.ReadFile(path)
        if err != nil {
            c.Logger().Warnf("file not found in contents: %s", name)
            return c.String(http.StatusNotFound, "file not found")
        }
        ct := "application/octet-stream"
        lower := strings.ToLower(name)
        switch {
        case strings.HasSuffix(lower, ".txt"):
            ct = "text/plain; charset=utf-8"
        case strings.HasSuffix(lower, ".html"):
            ct = "text/html; charset=utf-8"
        }
        return c.Blob(http.StatusOK, ct, b)
    })

    addr := ":3000"
    if p := os.Getenv("PORT"); p != "" {
        addr = ":" + p
    }
    e.Logger.Infof("starting server on %s", addr)
    if err := e.Start(addr); err != nil {
        e.Logger.Fatalf("server error: %v", err)
    }
}
