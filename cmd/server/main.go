package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	"tiktok-oauth/internal/config"
	"tiktok-oauth/internal/domain/oauth"
	"tiktok-oauth/internal/infrastructure/store"
	"tiktok-oauth/internal/infrastructure/tiktok"
	httpiface "tiktok-oauth/internal/interface/http"
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
	// Structured access logs to stdout; 5xx only go to stderr via Errorj
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:        true,
		LogURI:           true,
		LogMethod:        true,
		LogRemoteIP:      true,
		LogUserAgent:     true,
		LogHost:          true,
		LogLatency:       true,
		LogRequestID:     true,
		LogContentLength: true,
		LogResponseSize:  true,
		HandleError:      false,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			// parse content-length to int when possible
			var bytesIn int64
			if v.ContentLength != "" {
				// best-effort parse; ignore errors
				for i := 0; i < len(v.ContentLength); i++ {
					if v.ContentLength[i] < '0' || v.ContentLength[i] > '9' {
						bytesIn = 0
						break
					}
				}
			}
			m := log.JSON{
				"id":            v.RequestID,
				"remote_ip":     v.RemoteIP,
				"host":          v.Host,
				"method":        v.Method,
				"uri":           v.URI,
				"user_agent":    v.UserAgent,
				"status":        v.Status,
				"latency":       v.Latency.Nanoseconds(),
				"latency_human": v.Latency.String(),
				"bytes_in":      bytesIn,
				"bytes_out":     v.ResponseSize,
			}
			if v.Status >= 400 {
				m["level"] = "ERROR"
				if v.Error != nil {
					m["error"] = v.Error.Error()
				}
				c.Logger().Errorj(m)
			} else {
				m["level"] = "INFO"
				// Do not include error field for 4xx to avoid platform error classification
				c.Logger().Infoj(m)
			}
			return nil
		},
	}))

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

	// Insights dashboard (static HTML demo page)
	e.GET("/insights", func(c echo.Context) error {
		path := filepath.Join("contents", "insights.html")
		b, err := os.ReadFile(path)
		if err != nil {
			c.Logger().Errorf("failed to read insights.html: %v", err)
			return c.String(http.StatusInternalServerError, "insights.html not found")
		}
		return c.HTML(http.StatusOK, string(b))
	})

	// Serve mock response samples (for demo / recorder)
	e.Static("/docs", "docs")

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
