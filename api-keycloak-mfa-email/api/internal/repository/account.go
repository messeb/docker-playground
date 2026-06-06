package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/messeb/docker-playground/api-keycloak-mfa-email/internal/model"
)

var ErrNotFound = errors.New("account not found")

type AccountRepository struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{pool: pool}
}

// GetByAccountNumber fetches an account by its account number (read-only).
// The account number comes from the 'bank_account_number' JWT claim.
func (r *AccountRepository) GetByAccountNumber(ctx context.Context, accountNumber string) (*model.BankAccount, error) {
	return scanAccount(r.pool.QueryRow(ctx,
		`SELECT id, account_number, owner_name, balance, created_at
		 FROM bank_accounts WHERE account_number = $1`, accountNumber))
}

// GetByAccountNumberForUpdate fetches and locks the account row within a transaction.
// Use this before any balance modification to prevent concurrent race conditions.
func (r *AccountRepository) GetByAccountNumberForUpdate(ctx context.Context, tx pgx.Tx, accountNumber string) (*model.BankAccount, error) {
	return scanAccount(tx.QueryRow(ctx,
		`SELECT id, account_number, owner_name, balance, created_at
		 FROM bank_accounts WHERE account_number = $1 FOR UPDATE`, accountNumber))
}

// UpdateBalance sets a new balance within an open transaction.
func (r *AccountRepository) UpdateBalance(ctx context.Context, tx pgx.Tx, id int, newBalance float64) error {
	_, err := tx.Exec(ctx,
		`UPDATE bank_accounts SET balance = $1 WHERE id = $2`, newBalance, id)
	return err
}

func scanAccount(row pgx.Row) (*model.BankAccount, error) {
	var a model.BankAccount
	err := row.Scan(&a.ID, &a.AccountNumber, &a.OwnerName, &a.Balance, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}
