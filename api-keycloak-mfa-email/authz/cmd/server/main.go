// authz is the nginx auth_request sidecar. It performs the BFF re-encryption hop:
//
//   1. The client sends an encrypted access token (JWE) issued by Keycloak.
//      Keycloak fetched our public key from GET /.well-known/jwks.json on this
//      service and wrapped the access token with it.
//   2. authz decrypts the JWE with its private key, validates the inner JWS
//      against the Keycloak JWKS, and reads the standard claims.
//   3. authz re-encrypts the same JWS as a fresh JWE for the API's public key
//      (fetched from the API's own /.well-known/jwks.json at startup) and
//      returns it to nginx in the X-Auth-JWE header.
//   4. nginx forwards the request to the API with Authorization: Bearer <JWE>.
//
// MFA enforcement (mfa_verified claim) is NOT done here — the API applies it
// per-endpoint via RequireMFA so non-MFA users (carol) can still reach the
// non-sensitive routes.
package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/messeb/docker-playground/api-keycloak-mfa-email-authz/internal/config"
	"github.com/messeb/docker-playground/api-keycloak-mfa-email-authz/internal/jwe"
	"github.com/messeb/docker-playground/api-keycloak-mfa-email-authz/internal/jwks"
	jwsutil "github.com/messeb/docker-playground/api-keycloak-mfa-email-authz/internal/jws"
	"github.com/messeb/docker-playground/api-keycloak-mfa-email-authz/internal/wrapsig"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	cache, err := jwks.NewKeyCache(ctx, cfg.JWKSUrl, cfg.JWKSRefreshInterval)
	if err != nil {
		log.Fatalf("initialize JWKS cache: %v", err)
	}
	log.Printf("JWKS loaded from %s", cfg.JWKSUrl)

	pemBytes, err := cfg.BFFPrivateKeyPEM()
	if err != nil {
		log.Fatalf("decode BFF private key: %v", err)
	}
	bffKey, err := jwe.NewBFFKey(pemBytes)
	if err != nil {
		log.Fatalf("load BFF private key: %v", err)
	}
	jwksJSON, err := bffKey.PublicJWKSetJSON()
	if err != nil {
		log.Fatalf("serialize BFF JWKS: %v", err)
	}
	log.Println("BFF keypair loaded; JWKS available at /.well-known/jwks.json")

	// The wrapper no longer re-encrypts tokens for the API — claims are passed
	// as plain X-* headers instead — so the API's public key is not fetched.
	_ = ctx
	_ = cfg.APIPublicKeyURL

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Keycloak fetches this at token-issue time to encrypt the access token.
	mux.HandleFunc("GET /.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jwksJSON)
	})

	secret := []byte(cfg.WrapperHMACSecret)
	mux.HandleFunc("POST /token-proxy", tokenProxyHandler(cache, cfg.JWTIssuer, bffKey, cfg.KeycloakTokenURL))
	mux.HandleFunc("/verify", verifyHandler(cache, cfg.JWTIssuer, bffKey, secret))

	addr := ":" + cfg.Port
	log.Printf("authz listening on %s", addr)
	if err := http.ListenAndServe(addr, logRequests(mux)); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("→ %s %s ua=%q", r.Method, r.URL.Path, r.Header.Get("User-Agent"))
		next.ServeHTTP(w, r)
	})
}

func verifyHandler(cache *jwks.KeyCache, issuer string, bffKey *jwe.BFFKey, hmacSecret []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid Authorization header"})
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		switch strings.Count(tokenStr, ".") {
		case 4:
			plain, err := bffKey.Decrypt([]byte(tokenStr))
			if err != nil {
				log.Printf("decrypt incoming JWE: %v", err)
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "token decryption failed"})
				return
			}
			tokenStr = string(plain)
		case 2:
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "expected encrypted access token (JWE)"})
			return
		default:
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "malformed bearer token"})
			return
		}

		res, err := jwsutil.Validate(r.Context(), cache, issuer, tokenStr)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
			return
		}

		// The wrapper terminates token crypto. The upstream API receives plain
		// X-* headers and knows nothing about JWT/JWE/Keycloak.
		w.Header().Set("X-User-Sub", res.Sub)
		w.Header().Set("X-User-Email", res.Email)
		w.Header().Set("X-User-Username", res.PreferredUsername)
		w.Header().Set("X-User-Roles", strings.Join(res.RealmRoles, ","))
		w.Header().Set("X-Bank-Account-Number", res.BankAccountNumber)
		if res.MFAVerified {
			w.Header().Set("X-MFA-Verified", "true")
		} else {
			w.Header().Set("X-MFA-Verified", "false")
		}
		if res.MFAMethod != "" {
			w.Header().Set("X-MFA-Method", res.MFAMethod)
		}

		// HMAC-sign the X-* set so the upstream API can prove the request
		// arrived via this wrapper and not from a peer on the compose network.
		w.Header().Set(wrapsig.HeaderTimestamp, strconv.FormatInt(time.Now().Unix(), 10))
		w.Header().Set(wrapsig.HeaderSignature, wrapsig.Sign(w.Header().Get, hmacSecret))

		w.WriteHeader(http.StatusOK)
	}
}

// tokenProxyHandler turns the BFF into the SPA's *only* token endpoint.
//
// The SPA POSTs the OIDC token-endpoint form (grant_type=authorization_code +
// PKCE verifier, OR grant_type=refresh_token) here. We forward that body to
// Keycloak's real /token endpoint over the internal compose network, parse
// the response, replace `access_token` with its JWE-for-BFF wrap, and return
// the modified JSON. The plain Keycloak JWS never leaves this container.
//
// This closes the gap that `/wrap-token` had: previously the SPA exchanged
// the auth code at Keycloak directly, briefly held the cleartext JWS in
// memory, then called /wrap-token. With this proxy the JWS exists only
// inside authz; the SPA only ever sees the JWE.
//
// Errors from Keycloak (invalid_grant, expired refresh_token, etc.) are
// forwarded verbatim so the SPA's error handling keeps working unchanged.
func tokenProxyHandler(cache *jwks.KeyCache, issuer string, bffKey *jwe.BFFKey, keycloakTokenURL string) http.HandlerFunc {
	client := &http.Client{Timeout: 15 * time.Second}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")

		// Read raw form body and forward as-is.
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "could not read request body"})
			return
		}
		req, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, keycloakTokenURL, strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("token proxy upstream error: %v", err)
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "keycloak unreachable"})
			return
		}
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)

		// Pass non-2xx responses straight through — they carry Keycloak's
		// own error structure (invalid_grant etc.) and the SPA expects it.
		if resp.StatusCode >= 400 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(resp.StatusCode)
			w.Write(respBody)
			return
		}

		var payload map[string]any
		if err := json.Unmarshal(respBody, &payload); err != nil {
			log.Printf("token proxy: non-JSON response from KC: %v", err)
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "unexpected upstream response"})
			return
		}
		raw, ok := payload["access_token"].(string)
		if !ok || strings.Count(raw, ".") != 2 {
			log.Printf("token proxy: KC response missing JWS access_token")
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "no access_token in upstream response"})
			return
		}
		// Validate the JWS before wrapping (defense in depth — should always
		// succeed because Keycloak just issued it).
		if _, err := jwsutil.Validate(r.Context(), cache, issuer, raw); err != nil {
			log.Printf("token proxy: KC-issued JWS failed validation: %v", err)
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "upstream JWS rejected"})
			return
		}
		jweToken, err := bffKey.Encrypt([]byte(raw))
		if err != nil {
			log.Printf("token proxy encrypt: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encryption failure"})
			return
		}
		payload["access_token"] = jweToken

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(payload)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
