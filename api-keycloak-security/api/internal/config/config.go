package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DBHost              string
	DBPort              string
	DBUser              string
	DBPassword          string
	DBName              string
	JWKSUrl             string
	JWTIssuer           string
	JWKSRefreshInterval time.Duration
	APIPort             string
	// PrivateKeyBase64 holds a base64-encoded PEM RSA private key.
	// When set, the API decrypts JWE access tokens issued by Keycloak.
	PrivateKeyBase64    string
}

func Load() *Config {
	secs, err := strconv.Atoi(getEnv("JWKS_REFRESH_INTERVAL", "300"))
	if err != nil {
		secs = 300
	}

	return &Config{
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBUser:              mustGetEnv("DB_USER"),
		DBPassword:          mustGetEnv("DB_PASSWORD"),
		DBName:              mustGetEnv("DB_NAME"),
		JWKSUrl:             mustGetEnv("KEYCLOAK_JWKS_URL"),
		JWTIssuer:           mustGetEnv("KEYCLOAK_ISSUER"),
		JWKSRefreshInterval: time.Duration(secs) * time.Second,
		APIPort:             getEnv("API_PORT", "8080"),
		PrivateKeyBase64:    os.Getenv("API_PRIVATE_KEY_BASE64"),
	}
}

// PrivateKeyPEM decodes the base64 private key and returns the raw PEM bytes.
// Returns nil, nil when no key is configured.
func (c *Config) PrivateKeyPEM() ([]byte, error) {
	if c.PrivateKeyBase64 == "" {
		return nil, nil
	}
	return base64.StdEncoding.DecodeString(c.PrivateKeyBase64)
}

func (c *Config) DBConnString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
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
