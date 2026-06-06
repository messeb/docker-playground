package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/messeb/docker-playground/api-keycloak-mfa-email/internal/admin"
	"github.com/messeb/docker-playground/api-keycloak-mfa-email/internal/auth"
	"github.com/messeb/docker-playground/api-keycloak-mfa-email/internal/enrollment"
)

var enrollEmailRE = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type startReq struct {
	Email string `json:"email"`
}

type startResp struct {
	Status   string `json:"status"`
	Email    string `json:"email"`
	ExpireIn int    `json:"expires_in_seconds"`
	Note     string `json:"note"`
}

// EnrollStart handles POST /api/v1/mfa/enroll/start.
// Generates a 6-digit OTP, caches it server-side keyed by the caller's sub,
// and emails it to the provided address (Mailpit in dev).
func EnrollStart(store *enrollment.Store, mailer *enrollment.Mailer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok || claims.Sub == "" {
			writeJSON(w, http.StatusUnauthorized, errorResponse("no claims in context"))
			return
		}
		var req startReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
			return
		}
		if !enrollEmailRE.MatchString(req.Email) {
			writeJSON(w, http.StatusBadRequest, errorResponse("invalid email address"))
			return
		}
		code, err := store.Start(claims.Sub, req.Email)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse("could not allocate OTP"))
			return
		}
		if err := mailer.Send(req.Email, code); err != nil {
			log.Printf("send OTP mail: %v", err)
			writeJSON(w, http.StatusBadGateway, errorResponse("could not send OTP email"))
			return
		}
		writeJSON(w, http.StatusOK, startResp{
			Status:   "code_sent",
			Email:    req.Email,
			ExpireIn: 300,
			Note:     "check Mailpit at http://localhost:8025 for the 6-digit code",
		})
	}
}

type verifyReq struct {
	Code string `json:"code"`
}

type verifyResp struct {
	Status      string `json:"status"`
	StepUpUntil string `json:"step_up_verified_until"`
	Note        string `json:"note"`
}

// EnrollVerify handles POST /api/v1/mfa/enroll/verify.
// On success: marks the caller's sub as MFA-verified for the step-up window,
// and writes mfa_enabled=true on the Keycloak user so subsequent logins go
// through the proper email-OTP authenticator.
func EnrollVerify(store *enrollment.Store, kc *admin.KeycloakAdmin) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok || claims.Sub == "" {
			writeJSON(w, http.StatusUnauthorized, errorResponse("no claims in context"))
			return
		}
		var req verifyReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
			return
		}
		if err := store.Verify(claims.Sub, req.Code); err != nil {
			status := http.StatusUnauthorized
			if errors.Is(err, enrollment.ErrCodeExpired) || errors.Is(err, enrollment.ErrTooManyTries) {
				status = http.StatusForbidden
			}
			writeJSON(w, status, errorResponse(err.Error()))
			return
		}
		// Persist the attribute change in Keycloak so future logins use the
		// proper OTP authenticator. Picks up the email passed at /start.
		email, _ := store.PendingEmail(claims.Sub)
		if email == "" {
			email = claims.Email
		}
		if email != "" {
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()
			if err := kc.EnableEmailMFA(ctx, claims.Sub, email); err != nil {
				log.Printf("EnableEmailMFA after step-up verify: %v", err)
				// Don't fail the request — step-up already succeeded; the
				// Keycloak attribute is best-effort for *next* login.
			}
		}
		writeJSON(w, http.StatusOK, verifyResp{
			Status:      "mfa_verified",
			StepUpUntil: time.Now().Add(enrollment.StepUpTTL).UTC().Format(time.RFC3339),
			Note:        "current session is now MFA-verified; retry the protected endpoint without re-login",
		})
	}
}
