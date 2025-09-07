package oauth

type Token struct {
    AccessToken  string
    RefreshToken string
    ExpiresIn    int64
    TokenType    string
    Scope        string
    OpenID       string
}

