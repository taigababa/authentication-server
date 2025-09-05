package store

import (
    "context"
    "sync"

    doauth "tiktok-oauth/internal/domain/oauth"
)

type Memory struct {
    mu    sync.Mutex
    token doauth.Token
}

func (m *Memory) Save(ctx context.Context, t doauth.Token) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.token = t
    return nil
}

