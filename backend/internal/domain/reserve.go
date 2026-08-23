package domain

type ReserveRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required,min=1"`
	RequestID string `json:"request_id" binding:"required"`
}

type ReserveResponse struct {
	Status    string `json:"status"`
	ProductID string `json:"product_id"`
	Reserved  int    `json:"reserved"`
}

type StockReservedEvent struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	RequestID string `json:"request_id"`
	Timestamp int64  `json:"timestamp"`
}
