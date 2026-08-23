package service

import "errors"

var (
	ErrDuplicateRequest = errors.New("duplicate request")
	ErrProductNotFound  = errors.New("product not found")
	ErrNotEnoughStock   = errors.New("not enough stock")
)
