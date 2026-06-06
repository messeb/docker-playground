package jwks

import (
	"context"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
)

// KeyCache wraps jwk.Cache with auto-refresh. The Keycloak JWKS is fetched
// once at startup and refreshed in the background. No Keycloak call per request.
type KeyCache struct {
	cache *jwk.Cache
	url   string
}

// NewKeyCache creates and populates a JWKS cache.
// Performs an initial blocking fetch so startup fails fast if Keycloak is down.
func NewKeyCache(ctx context.Context, jwksURL string, refreshInterval time.Duration) (*KeyCache, error) {
	cache := jwk.NewCache(ctx)
	if err := cache.Register(jwksURL, jwk.WithMinRefreshInterval(refreshInterval)); err != nil {
		return nil, fmt.Errorf("register JWKS url: %w", err)
	}
	if _, err := cache.Refresh(ctx, jwksURL); err != nil {
		return nil, fmt.Errorf("initial JWKS fetch from %s: %w", jwksURL, err)
	}
	return &KeyCache{cache: cache, url: jwksURL}, nil
}

func (kc *KeyCache) KeySet(ctx context.Context) (jwk.Set, error) {
	return kc.cache.Get(ctx, kc.url)
}

// Refresh forces a re-fetch of the JWKS document — call this after a JWT
// signature validation failure to recover from key rotation without restarting.
func (kc *KeyCache) Refresh(ctx context.Context) (jwk.Set, error) {
	return kc.cache.Refresh(ctx, kc.url)
}
