package httpiface

import (
    "crypto/rand"
    "encoding/hex"
    "net/http"
    "strings"
    "fmt"
    "html/template"
    "path/filepath"

    "github.com/labstack/echo/v4"

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

    // If client explicitly requests JSON, keep existing JSON response
    if strings.Contains(c.Request().Header.Get("Accept"), "application/json") || c.QueryParam("format") == "json" {
        return c.JSON(http.StatusOK, map[string]any{
            "token": tokenToMap(tok),
            "user":  map[string]any{"data": user, "error": nil},
        })
    }

    avatarURL, displayName := extractUserProfile(user)
    // Render HTML using a template file under contents/callback.html
    tplPath := filepath.Join("contents", "callback.html")
    t, err := template.ParseFiles(tplPath)
    if err != nil {
        c.Logger().Errorf("failed to parse callback template: %v", err)
        return c.String(http.StatusInternalServerError, "callback template not found")
    }
    data := struct {
        AccessToken  string
        RefreshToken string
        AvatarURL    string
        DisplayName  string
    }{
        AccessToken:  tok.AccessToken,
        RefreshToken: tok.RefreshToken,
        AvatarURL:    avatarURL,
        DisplayName:  displayName,
    }
    c.Response().Header().Set(echo.HeaderContentType, "text/html; charset=utf-8")
    c.Response().WriteHeader(http.StatusOK)
    if err := t.Execute(c.Response().Writer, data); err != nil {
        c.Logger().Errorf("failed to execute callback template: %v", err)
        return c.String(http.StatusInternalServerError, "failed to render template")
    }
    return nil
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

// extractUserProfile attempts to pull avatar_url and display_name
// from TikTok user info response structure.
func extractUserProfile(m map[string]any) (avatarURL, displayName string) {
    // Expected shapes:
    // m = { "data": { "user": { "avatar_url": "", "display_name": "" } }, ... }
    // or sometimes m = { "data": { "data": { "user": { ... } } } }
    getStr := func(v any) string {
        if v == nil { return "" }
        if s, ok := v.(string); ok { return s }
        return fmt.Sprint(v)
    }
    var data any
    if v, ok := m["data"]; ok {
        data = v
    }
    if dm, ok := data.(map[string]any); ok {
        // direct user
        if u, ok := dm["user"].(map[string]any); ok {
            return getStr(u["avatar_url"]), getStr(u["display_name"])
        }
        // nested data.user
        if inner, ok := dm["data"].(map[string]any); ok {
            if u, ok := inner["user"].(map[string]any); ok {
                return getStr(u["avatar_url"]), getStr(u["display_name"])
            }
        }
    }
    return "", ""
}
