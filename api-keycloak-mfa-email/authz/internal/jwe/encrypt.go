package jwe

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	jwelib "github.com/lestrrat-go/jwx/v2/jwe"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// PublicKey caches the RSA public key fetched from the API's JWKS endpoint.
// The key is fetched once at startup; if the API restarts with a new key the
// authz process must also be restarted (or a refresh loop added).
type PublicKey struct {
	raw *rsa.PublicKey
}

// FetchPublicKey performs a blocking GET against the API's /.well-known/jwks.json
// and extracts the first RSA key. Fails fast if the API is unreachable so the
// authz process never starts in a broken state.
func FetchPublicKey(ctx context.Context, url string) (*PublicKey, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	var lastErr error
	for attempt := 1; attempt <= 30; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			set, err := jwk.Parse(body)
			if err != nil {
				return nil, fmt.Errorf("parse API JWKS: %w", err)
			}
			if set.Len() == 0 {
				return nil, errors.New("API JWKS is empty")
			}
			k, _ := set.Key(0)
			var raw rsa.PublicKey
			if err := k.Raw(&raw); err != nil {
				return nil, fmt.Errorf("extract RSA public key: %w", err)
			}
			return &PublicKey{raw: &raw}, nil
		}
		if resp != nil {
			resp.Body.Close()
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("fetch API public key from %s: %w", url, lastErr)
}

// Encrypt wraps a JWS compact token as a JWE using RSA-OAEP (key wrap) and
// AES-256-GCM (content encryption) — the same algorithm pair the API expects.
func (p *PublicKey) Encrypt(token string) (string, error) {
	jwe, err := jwelib.Encrypt(
		[]byte(token),
		jwelib.WithKey(jwa.RSA_OAEP, p.raw),
		jwelib.WithContentEncryption(jwa.A256GCM),
	)
	if err != nil {
		return "", fmt.Errorf("JWE encryption failed: %w", err)
	}
	return string(jwe), nil
}

// Helper used by tests / debug to introspect the loaded key.
func (p *PublicKey) DebugJSON() string {
	b, _ := json.Marshal(map[string]any{
		"size": p.raw.Size() * 8,
		"e":    p.raw.E,
	})
	return string(b)
}
