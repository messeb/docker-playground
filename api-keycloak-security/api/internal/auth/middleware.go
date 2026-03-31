package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwt"
)

type contextKey string

const claimsKey contextKey = "claims"

// Claims holds the parsed JWT fields used by the API.
type Claims struct {
	Sub               string
	PreferredUsername string
	Email             string
	RealmRoles        []string
	// BankAccountNumber is a custom Keycloak user attribute mapped into the JWT.
	// It identifies which bank account row in PostgreSQL belongs to this user.
	BankAccountNumber string
}

// Middleware validates the Bearer JWT on every request.
// If encKey is non-nil, the token is expected to be a JWE (encrypted by Keycloak
// using the API's public key). The middleware decrypts it first, then validates
// the inner JWS using the cached JWKS — no Keycloak call per request.
func Middleware(cache *KeyCache, issuer string, encKey *EncryptionKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid Authorization header"})
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			// JWE compact tokens have 5 dot-separated parts; JWS tokens have 3.
			if strings.Count(tokenStr, ".") == 4 {
				if encKey == nil {
					writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "encrypted token received but no decryption key configured"})
					return
				}
				decrypted, err := encKey.Decrypt([]byte(tokenStr))
				if err != nil {
					writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "token decryption failed"})
					return
				}
				tokenStr = string(decrypted)
			}

			ks, err := cache.KeySet(r.Context())
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "key set unavailable"})
				return
			}

			token, err := jwt.Parse([]byte(tokenStr),
				jwt.WithKeySet(ks),
				jwt.WithValidate(true),
				jwt.WithIssuer(issuer),
			)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
				return
			}

			claims := &Claims{
				Sub:               token.Subject(),
				PreferredUsername: stringClaim(token, "preferred_username"),
				Email:             stringClaim(token, "email"),
				RealmRoles:        extractRealmRoles(token),
				BankAccountNumber: stringClaim(token, "bank_account_number"),
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns middleware that enforces a Keycloak realm role.
// Must be chained after Middleware (which injects Claims into context).
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "no claims in context"})
				return
			}
			for _, claimRole := range claims.RealmRoles {
				if claimRole == role {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "insufficient permissions"})
		})
	}
}

// ClaimsFromContext retrieves *Claims injected by Middleware.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*Claims)
	return c, ok
}

func stringClaim(token jwt.Token, key string) string {
	v, ok := token.Get(key)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// extractRealmRoles parses the Keycloak-specific realm_access.roles claim.
func extractRealmRoles(token jwt.Token) []string {
	v, ok := token.Get("realm_access")
	if !ok {
		return nil
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	rawRoles, ok := m["roles"].([]interface{})
	if !ok {
		return nil
	}
	roles := make([]string, 0, len(rawRoles))
	for _, r := range rawRoles {
		if s, ok := r.(string); ok {
			roles = append(roles, s)
		}
	}
	return roles
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
