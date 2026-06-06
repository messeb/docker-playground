// Package auth here is intentionally trivial.
//
// The API is fronted by an nginx + Go BFF wrapper that terminates all token
// crypto. The wrapper signs the X-* identity headers it emits with an HMAC
// shared with the API (WRAPPER_HMAC_SECRET). The API rejects any request
// without a valid+fresh signature.
//
// In production this HMAC is the demo-grade substitute for SPIFFE mTLS;
// see README "Going production-ready" for how to swap it for SPIRE-issued
// SVIDs and proxy_ssl_certificate.
package auth

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/messeb/docker-playground/api-keycloak-mfa-email/internal/wrapsig"
)

type contextKey string

const claimsKey contextKey = "claims"

// Claims is the identity + step-up state the wrapper hands to the API.
type Claims struct {
	Sub               string
	PreferredUsername string
	Email             string
	BankAccountNumber string
	MFAVerified       bool
	MFAMethod         string
}

// VerifyWrapperSignature is the gate that proves the request came through the
// wrapper. It checks the HMAC over the X-* identity headers + a timestamp.
// Rejects requests that lack the signature, have a stale timestamp, or whose
// signature doesn't match under the shared secret. Apply this as the *first*
// middleware on any route that reads X-* identity headers.
func VerifyWrapperSignature(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := wrapsig.Verify(r.Header.Get, secret, time.Now()); err != nil {
				log.Printf("reject untrusted upstream: %v (path=%s)", err, r.URL.Path)
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"error":  "untrusted upstream",
					"detail": err.Error(),
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Middleware turns the wrapper's X-* headers into a Claims value in context.
// No crypto, no library calls — just header reads. Always chain after
// VerifyWrapperSignature so spoofed headers can't reach this code path.
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sub := r.Header.Get("X-User-Sub")
			if sub == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing identity headers"})
				return
			}
			claims := &Claims{
				Sub:               sub,
				PreferredUsername: r.Header.Get("X-User-Username"),
				Email:             r.Header.Get("X-User-Email"),
				BankAccountNumber: r.Header.Get("X-Bank-Account-Number"),
				MFAVerified:       r.Header.Get("X-MFA-Verified") == "true",
				MFAMethod:         r.Header.Get("X-MFA-Method"),
			}
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// StepUpChecker is the contract RequireMFA uses to ask whether a caller has
// completed an in-place OTP enrollment recently. Implemented by enrollment.Store.
type StepUpChecker interface {
	IsStepUpVerified(sub string) bool
}

// RequireMFA passes if the X-MFA-Verified header says "true" OR the caller has
// an active server-side step-up session from a recent in-place OTP enrollment.
func RequireMFA(stepUp StepUpChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "no claims in context"})
				return
			}
			if claims.MFAVerified {
				next.ServeHTTP(w, r)
				return
			}
			if stepUp != nil && stepUp.IsStepUpVerified(claims.Sub) {
				next.ServeHTTP(w, r)
				return
			}
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error":  "mfa required for this endpoint",
				"hint":   "enroll an OTP via POST /api/v1/mfa/enroll/start to step up your current session",
				"method": "email",
			})
		})
	}
}

// ClaimsFromContext retrieves *Claims injected by Middleware.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*Claims)
	return c, ok
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
