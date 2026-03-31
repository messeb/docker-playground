package handler

import (
	"encoding/json"
	"net/http"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/messeb/docker-playground/api-keycloak-security/internal/auth"
)

// ServeJWKS handles GET /.well-known/jwks.json
// Keycloak fetches this endpoint at login time to get the API's RSA public key,
// then uses it to encrypt the access token (JWE). The response is pre-serialized
// at startup since the key never changes at runtime.
func ServeJWKS(encKey *auth.EncryptionKey) http.HandlerFunc {
	ks := jwk.NewSet()
	ks.AddKey(encKey.PublicJWK())
	body, _ := json.Marshal(ks)

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}
}
