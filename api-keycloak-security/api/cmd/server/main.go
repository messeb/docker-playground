package main

import (
	"context"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/messeb/docker-playground/api-keycloak-security/internal/auth"
	"github.com/messeb/docker-playground/api-keycloak-security/internal/config"
	"github.com/messeb/docker-playground/api-keycloak-security/internal/handler"
	"github.com/messeb/docker-playground/api-keycloak-security/internal/repository"
	"github.com/messeb/docker-playground/api-keycloak-security/internal/service"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()

	// ── Database ──────────────────────────────────────────────────────────────
	pool, err := pgxpool.New(ctx, cfg.DBConnString())
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping database: %v", err)
	}
	log.Println("database connected")

	// ── JWKS cache ────────────────────────────────────────────────────────────
	// Performs an initial blocking fetch. Startup fails if Keycloak is unreachable.
	// After startup, keys refresh in the background — zero Keycloak calls per request.
	jwksCache, err := auth.NewKeyCache(ctx, cfg.JWKSUrl, cfg.JWKSRefreshInterval)
	if err != nil {
		log.Fatalf("initialize JWKS cache: %v", err)
	}
	log.Printf("JWKS loaded from %s (refresh every %s)", cfg.JWKSUrl, cfg.JWKSRefreshInterval)

	// ── JWE encryption key (optional) ─────────────────────────────────────────
	// When API_PRIVATE_KEY_BASE64 is set, the API decrypts JWE tokens before
	// validating the inner JWS. Keycloak fetches the matching public key from
	// GET /.well-known/jwks.json at login time.
	var encKey *auth.EncryptionKey
	pemBytes, err := cfg.PrivateKeyPEM()
	if err != nil {
		log.Fatalf("decode private key base64: %v", err)
	}
	if pemBytes != nil {
		encKey, err = auth.NewEncryptionKeyFromPEM(pemBytes)
		if err != nil {
			log.Fatalf("load encryption key: %v", err)
		}
		log.Println("JWE token encryption enabled")
	}

	// ── Wiring ────────────────────────────────────────────────────────────────
	accountRepo := repository.NewAccountRepository(pool)
	txRepo := repository.NewTransactionRepository(pool)
	accountSvc := service.NewAccountService(accountRepo, txRepo, pool)

	// ── Middleware ────────────────────────────────────────────────────────────
	protected := auth.Middleware(jwksCache, cfg.JWTIssuer, encKey)
	adminOnly := auth.RequireRole("admin")

	// ── Routes ────────────────────────────────────────────────────────────────
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Keycloak fetches this at login time to encrypt the access token.
	// Only registered when an encryption key is configured.
	if encKey != nil {
		mux.HandleFunc("GET /.well-known/jwks.json", handler.ServeJWKS(encKey))
		log.Println("JWKS endpoint: GET /.well-known/jwks.json")
	}

	mux.Handle("GET /api/v1/accounts/me",
		protected(http.HandlerFunc(handler.GetAccount(accountSvc))))

	mux.Handle("POST /api/v1/accounts/me/deposit",
		protected(http.HandlerFunc(handler.Deposit(accountSvc))))

	mux.Handle("POST /api/v1/accounts/me/withdraw",
		protected(http.HandlerFunc(handler.Withdraw(accountSvc))))

	mux.Handle("GET /api/v1/accounts/me/transactions",
		protected(http.HandlerFunc(handler.ListTransactions(accountSvc))))

	mux.Handle("POST /api/v1/admin/accounts",
		protected(adminOnly(http.HandlerFunc(handler.CreateAccount(accountSvc)))))

	// ── Server ────────────────────────────────────────────────────────────────
	addr := ":" + cfg.APIPort
	log.Printf("API listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
