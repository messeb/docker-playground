package jws

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"

	"github.com/messeb/docker-playground/api-keycloak-mfa-email-authz/internal/jwks"
)

// Result is what /verify returns to nginx via headers.
type Result struct {
	Sub               string
	Email             string
	PreferredUsername string
	RealmRoles        []string
	BankAccountNumber string
	MFAVerified       bool
	MFAMethod         string
}

// Validate parses the JWS, validates signature/issuer/expiry.
// MFA enforcement is intentionally NOT done here — the BFF accepts any valid
// session and the API enforces step-up per-endpoint via the mfa_verified
// claim, which is carried through inside the JWE that this BFF re-emits.
//
// On signature failure (typical when Keycloak rotated keys, e.g. after a
// dev-mode restart) the JWKS is force-refreshed and parsing retried once.
func Validate(ctx context.Context, cache *jwks.KeyCache, issuer, tokenStr string) (*Result, error) {
	ks, err := cache.KeySet(ctx)
	if err != nil {
		return nil, fmt.Errorf("keyset unavailable: %w", err)
	}

	parse := func(ks jwk.Set) (jwt.Token, error) {
		return jwt.Parse([]byte(tokenStr),
			jwt.WithKeySet(ks),
			jwt.WithValidate(true),
			jwt.WithIssuer(issuer),
		)
	}

	token, err := parse(ks)
	if err != nil {
		// Force refresh once — covers key rotation after Keycloak restart.
		ks2, rerr := cache.Refresh(ctx)
		if rerr == nil {
			token, err = parse(ks2)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	return &Result{
		Sub:               token.Subject(),
		Email:             stringClaim(token, "email"),
		PreferredUsername: stringClaim(token, "preferred_username"),
		RealmRoles:        extractRealmRoles(token),
		BankAccountNumber: stringClaim(token, "bank_account_number"),
		MFAVerified:       boolClaim(token, "mfa_verified"),
		MFAMethod:         stringClaim(token, "mfa_method"),
	}, nil
}

func stringClaim(token jwt.Token, key string) string {
	v, ok := token.Get(key)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func boolClaim(token jwt.Token, key string) bool {
	v, ok := token.Get(key)
	if !ok {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true"
	}
	return false
}

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
