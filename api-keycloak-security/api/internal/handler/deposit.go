package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/messeb/docker-playground/api-keycloak-security/internal/auth"
	"github.com/messeb/docker-playground/api-keycloak-security/internal/repository"
	"github.com/messeb/docker-playground/api-keycloak-security/internal/service"
)

type amountRequest struct {
	Amount float64 `json:"amount"`
}

// Deposit handles POST /api/v1/accounts/me/deposit
// Body: { "amount": 100.00 }
func Deposit(svc *service.AccountService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, _ := auth.ClaimsFromContext(r.Context())

		var req amountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount <= 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse("amount must be a positive number"))
			return
		}

		account, err := svc.Deposit(r.Context(), claims.BankAccountNumber, req.Amount)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, errorResponse("no bank account found for this user"))
				return
			}
			writeJSON(w, http.StatusInternalServerError, errorResponse("internal error"))
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"account_number": account.AccountNumber,
			"balance":        account.Balance,
		})
	}
}
