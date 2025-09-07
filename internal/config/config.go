package config

import (
    "os"
)

type Config struct {
    ClientKey    string
    ClientSecret string
    RedirectURI  string
    Scope        string
}

// Load reads environment variables and applies defaults.
func Load() Config {
    scope := os.Getenv("TIKTOK_SCOPE")
    if scope == "" {
        scope = "user.info.basic"
    }
    return Config{
        ClientKey:    os.Getenv("TIKTOK_CLIENT_KEY"),
        ClientSecret: os.Getenv("TIKTOK_CLIENT_SECRET"),
        RedirectURI:  os.Getenv("OAUTH_REDIRECT_URI"),
        Scope:        scope,
    }
}

