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

func NewUseCase(c TikTokClient, s Store, scope string) *UseCase {
    return &UseCase{client: c, store: s, scope: scope}
}

func (u *UseCase) LoginURL(state, redirectURI string) string {
    return u.client.AuthURL(state, redirectURI, u.scope)
}

func (u *UseCase) Callback(ctx context.Context, code, redirectURI string) (Token, error) {
    tok, err := u.client.Exchange(ctx, code, redirectURI)
    if err != nil {
        return Token{}, err
    }
    _ = u.store.Save(ctx, tok)
    return tok, nil
}

// Optional convenience to fetch user info via UseCase.
func (u *UseCase) GetUserInfo(ctx context.Context, accessToken string, fields []string) (map[string]any, error) {
    return u.client.GetUserInfo(ctx, accessToken, fields)
}

