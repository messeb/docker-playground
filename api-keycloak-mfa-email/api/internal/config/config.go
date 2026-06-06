package config

import (
	"fmt"
	"os"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	APIPort    string
	// Keycloak admin credentials — used only by the in-place MFA enrollment
	// flow to flip a user attribute. Demo-quality (master-realm admin via
	// password grant); production should use a confidential service account.
	KeycloakBaseURL     string
	KeycloakAdminUser   string
	KeycloakAdminPass   string
	KeycloakTargetRealm string
	// SMTP for the OTP mailer (Mailpit in dev).
	SMTPHost string
	SMTPFrom string
	// WrapperHMACSecret defends the API against direct calls: the wrapper
	// signs every set of X-* identity headers it emits; the API rejects any
	// inbound request that doesn't carry a fresh, matching signature.
	// Production should replace this with SPIFFE mTLS (see README).
	WrapperHMACSecret string
}

func Load() *Config {
	return &Config{
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBUser:              mustGetEnv("DB_USER"),
		DBPassword:          mustGetEnv("DB_PASSWORD"),
		DBName:              mustGetEnv("DB_NAME"),
		APIPort:             getEnv("API_PORT", "8080"),
		KeycloakBaseURL:     getEnv("KEYCLOAK_BASE_URL", ""),
		KeycloakAdminUser:   getEnv("KEYCLOAK_ADMIN_USER", ""),
		KeycloakAdminPass:   getEnv("KEYCLOAK_ADMIN_PASSWORD", ""),
		KeycloakTargetRealm: getEnv("KEYCLOAK_TARGET_REALM", ""),
		SMTPHost:            getEnv("SMTP_HOST", "mailpit:1025"),
		SMTPFrom:            getEnv("SMTP_FROM", "no-reply@banking.local"),
		WrapperHMACSecret:   mustGetEnv("WRAPPER_HMAC_SECRET"),
	}
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
