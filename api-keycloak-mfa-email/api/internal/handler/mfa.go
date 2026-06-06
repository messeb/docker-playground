package handler

import (
	"net/http"

	"github.com/messeb/docker-playground/api-keycloak-mfa-email/internal/auth"
)

type mfaStatusResponse struct {
	Verified bool   `json:"verified"`
	Method   string `json:"method"`
	Sub      string `json:"sub"`
	Email    string `json:"email"`
}

// MFAStatus handles GET /api/v1/mfa/status — echoes the JWT-derived MFA claims.
// The token will only reach this handler if the authz BFF accepted it,
// meaning mfa_verified is already enforced upstream. This endpoint exists so
// clients can read the verification state without parsing the token themselves.
func MFAStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse("no claims in context"))
			return
		}
		method := claims.MFAMethod
		if method == "" {
			method = "none"
		}
		writeJSON(w, http.StatusOK, mfaStatusResponse{
			Verified: claims.MFAVerified,
			Method:   method,
			Sub:      claims.Sub,
			Email:    claims.Email,
		})
	}
}
