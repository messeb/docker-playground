package model

import "time"

type BankAccount struct {
	ID            int
	AccountNumber string
	OwnerName     string
	Balance       float64
	CreatedAt     time.Time
}
