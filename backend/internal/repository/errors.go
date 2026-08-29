package repository

import "errors"

var (
	ErrProductNotFound = errors.New("product not found")
	ErrNotEnoughStock  = errors.New("not enough stock")
	ErrVersionConflict = errors.New("version conflict, please retry")
)
