package model

import "time"

type Transaction struct {
	ID          int
	AccountID   int
	Type        string
	Amount      float64
	Description string
	CreatedAt   time.Time
}
