package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
)

// KeyCache wraps jwk.Cache with auto-refresh. JWKS are fetched from Keycloak
// once at startup and refreshed in the background every RefreshInterval.
// The API validates JWTs entirely in-memory — no per-request Keycloak calls.
type KeyCache struct {
	cache *jwk.Cache
	url   string
}

// NewKeyCache creates and populates a JWKS cache. It performs an initial
// blocking fetch so startup fails fast if Keycloak is unreachable.
func NewKeyCache(ctx context.Context, jwksURL string, refreshInterval time.Duration) (*KeyCache, error) {
	cache := jwk.NewCache(ctx)

	if err := cache.Register(jwksURL, jwk.WithMinRefreshInterval(refreshInterval)); err != nil {
		return nil, fmt.Errorf("register JWKS url: %w", err)
	}

	// Blocking initial fetch — fail fast if Keycloak is not ready.
	if _, err := cache.Refresh(ctx, jwksURL); err != nil {
		return nil, fmt.Errorf("initial JWKS fetch from %s: %w", jwksURL, err)
	}

	return &KeyCache{cache: cache, url: jwksURL}, nil
}

// KeySet returns the current cached key set. The background goroutine started
// by jwk.NewCache keeps this up to date automatically.
func (kc *KeyCache) KeySet(ctx context.Context) (jwk.Set, error) {
	return kc.cache.Get(ctx, kc.url)
}
