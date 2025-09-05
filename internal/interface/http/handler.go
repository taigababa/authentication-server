package httpiface

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "net/http"

    "github.com/labstack/echo/v4"
    "github.com/labstack/gommon/log"

    "tiktok-oauth/internal/domain/oauth"
    "tiktok-oauth/internal/pkg/httpx"
)

type Handler struct {
    UC          *oauth.UseCase
    RedirectURI string
}

func (h *Handler) Login(c echo.Context) error {
    state := randomHex(16)
    url := h.UC.LoginURL(state, h.RedirectURI)
    c.Logger().Infof("redirecting to TikTok auth: state_generated")
    return c.Redirect(http.StatusFound, url)
}

func (h *Handler) Callback(c echo.Context) error {
    if e := c.QueryParam("error"); e != "" {
        c.Logger().Errorf("oauth error on callback: %s", e)
        return httpx.JSONError(c, http.StatusBadRequest, "oauth_error", map[string]string{"error": e})
    }
    code := c.QueryParam("code")
    if code == "" {
        return httpx.JSONError(c, http.StatusBadRequest, "missing_code", nil)
    }

    ctx := c.Request().Context()
    tok, err := h.UC.Callback(ctx, code, h.RedirectURI)
    if err != nil {
        c.Logger().Errorf("token exchange failed: %v", err)
        return httpx.JSONError(c, http.StatusBadGateway, "token_exchange_failed", nil)
    }

    // Fetch user info
    fields := []string{"open_id", "display_name", "avatar_url"}
    user, err := h.UC.GetUserInfo(ctx, tok.AccessToken, fields)
    if err != nil {
        c.Logger().Errorf("user info fetch failed: %v", err)
        // Still return token with user error
        return c.JSON(http.StatusOK, map[string]any{
            "token": tokenToMap(tok),
            "user":  map[string]any{"data": nil, "error": err.Error()},
        })
    }

    return c.JSON(http.StatusOK, map[string]any{
        "token": tokenToMap(tok),
        "user":  map[string]any{"data": user, "error": nil},
    })
}

func randomHex(n int) string {
    b := make([]byte, n)
    if _, err := rand.Read(b); err != nil {
        // Should not happen; fall back to timestamp-like pseudo
        return hex.EncodeToString([]byte("fallbackstateseed"))
    }
    return hex.EncodeToString(b)
}

func tokenToMap(t oauth.Token) map[string]any {
    return map[string]any{
        "access_token":  t.AccessToken,
        "refresh_token": t.RefreshToken,
        "expires_in":    t.ExpiresIn,
        "open_id":       t.OpenID,
        "scope":         t.Scope,
        "token_type":    t.TokenType,
    }
}

// Ensure imported log is referenced to satisfy linter if not used elsewhere
var _ = log.INFO
