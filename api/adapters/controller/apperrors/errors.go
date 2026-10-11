package apperrors

import "errors"

var (
	ErrNotFound = errors.New("This record was not found")
	ErrConflict = errors.New("Record already exists")
	ErrInvalidInput = errors.New("Invalid input")
)