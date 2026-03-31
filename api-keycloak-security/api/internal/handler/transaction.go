package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/messeb/docker-playground/api-keycloak-security/internal/auth"
	"github.com/messeb/docker-playground/api-keycloak-security/internal/repository"
	"github.com/messeb/docker-playground/api-keycloak-security/internal/service"
)

type transactionResponse struct {
	ID          int       `json:"id"`
	Type        string    `json:"type"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ListTransactions handles GET /api/v1/accounts/me/transactions?limit=20&offset=0
func ListTransactions(svc *service.AccountService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, _ := auth.ClaimsFromContext(r.Context())

		limit := queryInt(r, "limit", 20)
		offset := queryInt(r, "offset", 0)
		if limit < 1 || limit > 100 {
			limit = 20
		}

		txs, err := svc.ListTransactions(r.Context(), claims.BankAccountNumber, limit, offset)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, errorResponse("no bank account found for this user"))
				return
			}
			writeJSON(w, http.StatusInternalServerError, errorResponse("internal error"))
			return
		}

		resp := make([]transactionResponse, 0, len(txs))
		for _, t := range txs {
			resp = append(resp, transactionResponse{
				ID:          t.ID,
				Type:        t.Type,
				Amount:      t.Amount,
				Description: t.Description,
				CreatedAt:   t.CreatedAt,
			})
		}

		writeJSON(w, http.StatusOK, map[string]any{"transactions": resp})
	}
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return defaultVal
	}
	return v
}
