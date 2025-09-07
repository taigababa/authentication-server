package tiktok

import (
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
    "time"

    doauth "tiktok-oauth/internal/domain/oauth"
)

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

func defaultHTTPClient(c *http.Client) *http.Client {
    if c != nil {
        return c
    }
    return &http.Client{Timeout: 10 * time.Second}
}

// AuthURL builds TikTok v2 authorization URL.
func (c *Client) AuthURL(state, redirectURI, scope string) string {
    v := url.Values{}
    v.Set("client_key", c.ClientKey)
    v.Set("response_type", "code")
    if scope != "" {
        v.Set("scope", scope)
    }
    v.Set("redirect_uri", redirectURI)
    if state != "" {
        v.Set("state", state)
    }
    return AuthEndpoint + "?" + v.Encode()
}

// Exchange exchanges authorization code for tokens using x-www-form-urlencoded.
func (c *Client) Exchange(ctx context.Context, code, redirectURI string) (doauth.Token, error) {
    form := url.Values{}
    form.Set("client_key", c.ClientKey)
    form.Set("client_secret", c.ClientSecret)
    form.Set("code", code)
    form.Set("grant_type", "authorization_code")
    // Important: include redirect_uri in body (must match auth request)
    form.Set("redirect_uri", redirectURI)

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, TokenEndpoint, strings.NewReader(form.Encode()))
    if err != nil {
        return doauth.Token{}, err
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    httpClient := defaultHTTPClient(c.HTTP)
    resp, err := httpClient.Do(req)
    if err != nil {
        return doauth.Token{}, err
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return doauth.Token{}, fmt.Errorf("token exchange failed: status=%d body=%s", resp.StatusCode, string(body))
    }

    // TikTok v2 typically wraps in { "data": { ... } }
    var wrapped struct {
        Data map[string]any `json:"data"`
        Error string        `json:"error"`
        Message string      `json:"message"`
    }
    if err := json.Unmarshal(body, &wrapped); err != nil {
        return doauth.Token{}, fmt.Errorf("decode token response: %w", err)
    }

    var data map[string]any
    switch {
    case wrapped.Data != nil:
        data = wrapped.Data
    default:
        // Try decode as flat response
        if err := json.Unmarshal(body, &data); err != nil {
            return doauth.Token{}, fmt.Errorf("unexpected token response format")
        }
    }

    if wrapped.Error != "" {
        return doauth.Token{}, errors.New(wrapped.Error + ": " + wrapped.Message)
    }

    token := doauth.Token{
        AccessToken:  strVal(data["access_token"]),
        RefreshToken: strVal(data["refresh_token"]),
        TokenType:    strVal(data["token_type"]),
        Scope:        strVal(data["scope"]),
        OpenID:       strVal(data["open_id"]),
    }
    if v, ok := numToInt64(data["expires_in"]); ok {
        token.ExpiresIn = v
    }
    return token, nil
}

// GetUserInfo fetches user info with given fields using Bearer token.
func (c *Client) GetUserInfo(ctx context.Context, accessToken string, fields []string) (map[string]any, error) {
    if accessToken == "" {
        return nil, errors.New("missing access token")
    }
    q := url.Values{}
    if len(fields) > 0 {
        q.Set("fields", strings.Join(fields, ","))
    }
    u := UserInfoURL
    if q.Encode() != "" {
        u += "?" + q.Encode()
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Authorization", "Bearer "+accessToken)

    httpClient := defaultHTTPClient(c.HTTP)
    resp, err := httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return nil, fmt.Errorf("user info failed: status=%d body=%s", resp.StatusCode, trunc(body, 2048))
    }
    var out map[string]any
    if err := json.Unmarshal(body, &out); err != nil {
        return nil, fmt.Errorf("decode user info: %w", err)
    }
    return out, nil
}

func strVal(v any) string {
    if v == nil { return "" }
    switch t := v.(type) {
    case string:
        return t
    case json.Number:
        return t.String()
    default:
        b, _ := json.Marshal(t)
        return string(bytes.Trim(b, "\""))
    }
}

func numToInt64(v any) (int64, bool) {
    switch t := v.(type) {
    case float64:
        return int64(t), true
    case int64:
        return t, true
    case json.Number:
        i, err := t.Int64(); if err == nil { return i, true }
    }
    return 0, false
}

func trunc(b []byte, n int) string {
    if len(b) <= n { return string(b) }
    return string(b[:n]) + "..."
}

