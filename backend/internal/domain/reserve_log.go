package domain

import "time"

type ReserveLogEntry struct {
	ID           int64     `json:"id"`
	ProductID    string    `json:"product_id"`
	Quantity     int       `json:"quantity"`
	RequestID    string    `json:"request_id"`
	UserID       string    `json:"user_id"`
	Status       string    `json:"status"`
	ErrorMessage *string   `json:"error_message"`
	CreatedAt    time.Time `json:"created_at"`
}
