package oauth

import (
    "context"
    "errors"
    "reflect"
    "testing"
)

type mockClient struct{
    authURL   string
    token     Token
    exchErr   error
}

func (m *mockClient) AuthURL(state, redirectURI, scope string) string { return m.authURL }
func (m *mockClient) Exchange(ctx context.Context, code, redirectURI string) (Token, error) { return m.token, m.exchErr }
func (m *mockClient) GetUserInfo(ctx context.Context, accessToken string, fields []string) (map[string]any, error) { return map[string]any{"ok": true}, nil }

type mockStore struct{}
func (m *mockStore) Save(ctx context.Context, t Token) error { return nil }

func TestUseCase_LoginURL(t *testing.T) {
    mc := &mockClient{authURL: "https://example/auth?x=y"}
    uc := NewUseCase(mc, &mockStore{}, "user.info.basic")
    got := uc.LoginURL("state", "https://cb")
    if got != mc.authURL {
        t.Fatalf("unexpected LoginURL: got=%s want=%s", got, mc.authURL)
    }
}

func TestUseCase_Callback_Success(t *testing.T) {
    want := Token{AccessToken: "a", RefreshToken: "r", OpenID: "o", Scope: "s", TokenType: "Bearer", ExpiresIn: 1}
    mc := &mockClient{token: want}
    uc := NewUseCase(mc, &mockStore{}, "user.info.basic")
    got, err := uc.Callback(context.Background(), "code", "https://cb")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !reflect.DeepEqual(got, want) {
        t.Fatalf("unexpected token: %#v", got)
    }
}

func TestUseCase_Callback_Error(t *testing.T) {
    mc := &mockClient{exchErr: errors.New("boom")}
    uc := NewUseCase(mc, &mockStore{}, "user.info.basic")
    _, err := uc.Callback(context.Background(), "code", "https://cb")
    if err == nil {
        t.Fatalf("expected error")
    }
}

