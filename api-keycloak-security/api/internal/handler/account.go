package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/messeb/docker-playground/api-keycloak-security/internal/auth"
	"github.com/messeb/docker-playground/api-keycloak-security/internal/repository"
	"github.com/messeb/docker-playground/api-keycloak-security/internal/service"
)

type accountResponse struct {
	AccountNumber string    `json:"account_number"`
	OwnerName     string    `json:"owner_name"`
	Balance       float64   `json:"balance"`
	CreatedAt     time.Time `json:"created_at"`
}

// GetAccount handles GET /api/v1/accounts/me
func GetAccount(svc *service.AccountService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, _ := auth.ClaimsFromContext(r.Context())

		account, err := svc.GetAccount(r.Context(), claims.BankAccountNumber)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, errorResponse("no bank account found for this user"))
				return
			}
			writeJSON(w, http.StatusInternalServerError, errorResponse("internal error"))
			return
		}

		writeJSON(w, http.StatusOK, accountResponse{
			AccountNumber: account.AccountNumber,
			OwnerName:     account.OwnerName,
			Balance:       account.Balance,
			CreatedAt:     account.CreatedAt,
		})
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func errorResponse(msg string) map[string]string {
	return map[string]string{"error": msg}
}
