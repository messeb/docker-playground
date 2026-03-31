package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/messeb/docker-playground/api-keycloak-security/internal/model"
)

type TransactionRepository struct {
	pool *pgxpool.Pool
}

func NewTransactionRepository(pool *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{pool: pool}
}

// Create inserts a transaction record within an open database transaction.
func (r *TransactionRepository) Create(ctx context.Context, tx pgx.Tx, accountID int, txType string, amount float64, description string) (*model.Transaction, error) {
	var t model.Transaction
	err := tx.QueryRow(ctx,
		`INSERT INTO transactions (account_id, type, amount, description)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, account_id, type, amount, description, created_at`,
		accountID, txType, amount, description,
	).Scan(&t.ID, &t.AccountID, &t.Type, &t.Amount, &t.Description, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ListByAccountID returns transactions for an account, newest first.
func (r *TransactionRepository) ListByAccountID(ctx context.Context, accountID, limit, offset int) ([]model.Transaction, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, account_id, type, amount, description, created_at
		 FROM transactions
		 WHERE account_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		accountID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []model.Transaction
	for rows.Next() {
		var t model.Transaction
		if err := rows.Scan(&t.ID, &t.AccountID, &t.Type, &t.Amount, &t.Description, &t.CreatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, t)
	}
	return txs, rows.Err()
}
