package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                string
	JWKSUrl             string
	JWTIssuer           string
	JWKSRefreshInterval time.Duration
	APIPublicKeyURL     string
	// BFFPrivateKeyBase64 is the RSA private key used to decrypt the JWE
	// Keycloak emits. The matching public key is served at
	// GET /.well-known/jwks.json on this authorizer.
	BFFPrivateKeyBase64 string
	// WrapperHMACSecret is shared with the upstream API. The wrapper signs
	// every X-* identity header set it emits so the API can reject any
	// request that didn't come through it.
	WrapperHMACSecret string
	// KeycloakTokenURL is the *internal* Keycloak token endpoint the BFF
	// proxies to. The SPA never calls Keycloak directly for token exchange —
	// /api/token on nginx hits /token-proxy here, this URL is the upstream.
	KeycloakTokenURL string
}

func Load() *Config {
	secs, err := strconv.Atoi(getEnv("JWKS_REFRESH_INTERVAL", "300"))
	if err != nil {
		secs = 300
	}
	return &Config{
		Port:                getEnv("AUTHZ_PORT", "9000"),
		JWKSUrl:             mustGetEnv("KEYCLOAK_JWKS_URL"),
		JWTIssuer:           mustGetEnv("KEYCLOAK_ISSUER"),
		JWKSRefreshInterval: time.Duration(secs) * time.Second,
		APIPublicKeyURL:     mustGetEnv("API_PUBLIC_KEY_URL"),
		BFFPrivateKeyBase64: mustGetEnv("AUTHZ_PRIVATE_KEY_BASE64"),
		WrapperHMACSecret:   mustGetEnv("WRAPPER_HMAC_SECRET"),
		KeycloakTokenURL:    mustGetEnv("KEYCLOAK_TOKEN_URL"),
	}
}

// BFFPrivateKeyPEM decodes the base64 private key into raw PEM bytes.
func (c *Config) BFFPrivateKeyPEM() ([]byte, error) {
	return base64.StdEncoding.DecodeString(c.BFFPrivateKeyBase64)
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
}
