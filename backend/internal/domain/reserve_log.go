package domain

import "time"

type ReserveLogEntry struct {
	ID           int64     `db:"id" json:"id"`
	ProductID    string    `db:"product_id" json:"product_id"`
	Quantity     int       `db:"quantity" json:"quantity"`
	RequestID    string    `db:"request_id" json:"request_id"`
	UserID       string    `db:"user_id" json:"user_id"`
	Status       string    `db:"status" json:"status"`
	ErrorMessage *string   `db:"error_message" json:"error_message"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}
