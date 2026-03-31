package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/messeb/docker-playground/api-keycloak-security/internal/service"
)

type createAccountRequest struct {
	AccountNumber string `json:"account_number"`
	OwnerName     string `json:"owner_name"`
}

// CreateAccount handles POST /api/v1/admin/accounts (admin role required).
// Creates a bank account DB entry for a given account_number.
// The account_number must match the 'bank_account_number' attribute set for the
// user in Keycloak — that attribute is what gets included in their JWT and used
// for all subsequent API lookups.
func CreateAccount(svc *service.AccountService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createAccountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
			return
		}
		if req.AccountNumber == "" || req.OwnerName == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse("account_number and owner_name are required"))
			return
		}

		account, err := svc.CreateAccount(r.Context(), req.AccountNumber, req.OwnerName)
		if err != nil {
			if errors.Is(err, service.ErrAccountAlreadyExists) {
				writeJSON(w, http.StatusConflict, errorResponse("an account with this number already exists"))
				return
			}
			writeJSON(w, http.StatusInternalServerError, errorResponse("internal error"))
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"account_number": account.AccountNumber,
			"owner_name":     account.OwnerName,
			"balance":        account.Balance,
		})
	}
}
