package apperrors

import "errors"

var (
	ErrNotFound = errors.New("This record was not found")
	ErrConflict = errors.New("Record already exists")
	ErrInvalidInput = errors.New("Invalid input")
	ErrUnauthorized = errors.New("Unauthorized access")
	ErrForbidden = errors.New("Forbidden access")
)

type FieldValidateErrors struct {
	Field   string `json:"campo"`
	Message string `json:"mensagem"`
}