package main

import (
	"context"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/messeb/docker-playground/api-keycloak-mfa-email/internal/admin"
	"github.com/messeb/docker-playground/api-keycloak-mfa-email/internal/auth"
	"github.com/messeb/docker-playground/api-keycloak-mfa-email/internal/config"
	"github.com/messeb/docker-playground/api-keycloak-mfa-email/internal/enrollment"
	"github.com/messeb/docker-playground/api-keycloak-mfa-email/internal/handler"
	"github.com/messeb/docker-playground/api-keycloak-mfa-email/internal/repository"
	"github.com/messeb/docker-playground/api-keycloak-mfa-email/internal/service"
)

// The API has no knowledge of JWT/JWE/Keycloak. Identity arrives as plain
// X-* headers from the wrapper (nginx + authz). Authenticity is proven by an
// HMAC over those headers using a secret shared with the wrapper. The API
// container is only reachable through that wrapper; the HMAC is the
// belt-and-braces against direct calls.
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

	// ── Wiring ────────────────────────────────────────────────────────────────
	accountRepo := repository.NewAccountRepository(pool)
	txRepo := repository.NewTransactionRepository(pool)
	accountSvc := service.NewAccountService(accountRepo, txRepo, pool)

	enrollStore := enrollment.NewStore()
	mailer := enrollment.NewMailer(cfg.SMTPHost, cfg.SMTPFrom)

	// ── Middleware ────────────────────────────────────────────────────────────
	wrapperSig := auth.VerifyWrapperSignature([]byte(cfg.WrapperHMACSecret))
	protected := func(h http.Handler) http.Handler { return wrapperSig(auth.Middleware()(h)) }
	mfaOnly := auth.RequireMFA(enrollStore)

	// ── Routes ────────────────────────────────────────────────────────────────
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	mux.Handle("GET /api/v1/mfa/status",
		protected(http.HandlerFunc(handler.MFAStatus())))

	if cfg.KeycloakBaseURL != "" {
		kcAdmin := admin.New(cfg.KeycloakBaseURL, cfg.KeycloakTargetRealm, cfg.KeycloakAdminUser, cfg.KeycloakAdminPass)
		if err := kcAdmin.EnsureUnmanagedAttributesEnabled(ctx); err != nil {
			log.Printf("WARN: could not enable unmanaged user attributes: %v", err)
		} else {
			log.Println("unmanagedAttributePolicy=ENABLED on realm")
		}
		mux.Handle("POST /api/v1/mfa/enroll/start",
			protected(http.HandlerFunc(handler.EnrollStart(enrollStore, mailer))))
		mux.Handle("POST /api/v1/mfa/enroll/verify",
			protected(http.HandlerFunc(handler.EnrollVerify(enrollStore, kcAdmin))))
		log.Printf("MFA self-enroll endpoints enabled (Keycloak admin %s/realms/%s, SMTP %s)", cfg.KeycloakBaseURL, cfg.KeycloakTargetRealm, cfg.SMTPHost)
	}

	mux.Handle("GET /api/v1/accounts/me",
		protected(http.HandlerFunc(handler.GetAccount(accountSvc))))
	mux.Handle("POST /api/v1/accounts/me/deposit",
		protected(mfaOnly(http.HandlerFunc(handler.Deposit(accountSvc)))))
	mux.Handle("POST /api/v1/accounts/me/withdraw",
		protected(mfaOnly(http.HandlerFunc(handler.Withdraw(accountSvc)))))
	mux.Handle("GET /api/v1/accounts/me/transactions",
		protected(http.HandlerFunc(handler.ListTransactions(accountSvc))))

	addr := ":" + cfg.APIPort
	log.Printf("API listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
