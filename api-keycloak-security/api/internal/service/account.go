package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/messeb/docker-playground/api-keycloak-security/internal/model"
	"github.com/messeb/docker-playground/api-keycloak-security/internal/repository"
)

var ErrInsufficientFunds    = errors.New("insufficient funds")
var ErrAccountAlreadyExists = errors.New("account already exists")

type AccountService struct {
	accounts     *repository.AccountRepository
	transactions *repository.TransactionRepository
	pool         *pgxpool.Pool
}

func NewAccountService(
	accounts *repository.AccountRepository,
	transactions *repository.TransactionRepository,
	pool *pgxpool.Pool,
) *AccountService {
	return &AccountService{
		accounts:     accounts,
		transactions: transactions,
		pool:         pool,
	}
}

// GetAccount returns the bank account identified by accountNumber.
// accountNumber comes from the 'bank_account_number' JWT claim.
func (s *AccountService) GetAccount(ctx context.Context, accountNumber string) (*model.BankAccount, error) {
	return s.accounts.GetByAccountNumber(ctx, accountNumber)
}

// Deposit adds amount to the account balance inside a DB transaction.
// A SELECT FOR UPDATE row lock prevents concurrent balance corruption.
func (s *AccountService) Deposit(ctx context.Context, accountNumber string, amount float64) (*model.BankAccount, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	account, err := s.accounts.GetByAccountNumberForUpdate(ctx, tx, accountNumber)
	if err != nil {
		return nil, err
	}

	newBalance := account.Balance + amount
	if err := s.accounts.UpdateBalance(ctx, tx, account.ID, newBalance); err != nil {
		return nil, err
	}

	if _, err := s.transactions.Create(ctx, tx, account.ID, "deposit", amount, ""); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	account.Balance = newBalance
	return account, nil
}

// Withdraw subtracts amount from the account balance.
// Returns ErrInsufficientFunds if balance would go negative.
func (s *AccountService) Withdraw(ctx context.Context, accountNumber string, amount float64) (*model.BankAccount, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	account, err := s.accounts.GetByAccountNumberForUpdate(ctx, tx, accountNumber)
	if err != nil {
		return nil, err
	}

	if account.Balance < amount {
		return nil, ErrInsufficientFunds
	}

	newBalance := account.Balance - amount
	if err := s.accounts.UpdateBalance(ctx, tx, account.ID, newBalance); err != nil {
		return nil, err
	}

	if _, err := s.transactions.Create(ctx, tx, account.ID, "withdrawal", amount, ""); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	account.Balance = newBalance
	return account, nil
}

// ListTransactions returns paginated transaction history for the account.
func (s *AccountService) ListTransactions(ctx context.Context, accountNumber string, limit, offset int) ([]model.Transaction, error) {
	account, err := s.accounts.GetByAccountNumber(ctx, accountNumber)
	if err != nil {
		return nil, err
	}
	return s.transactions.ListByAccountID(ctx, account.ID, limit, offset)
}

// CreateAccount creates a new bank account entry in the database.
// accountNumber must match the 'bank_account_number' attribute set in Keycloak for the user.
// This is an admin operation — the handler enforces the admin role.
func (s *AccountService) CreateAccount(ctx context.Context, accountNumber, ownerName string) (*model.BankAccount, error) {
	account, err := s.accounts.Create(ctx, accountNumber, ownerName)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrAccountAlreadyExists
		}
		return nil, err
	}
	return account, nil
}
