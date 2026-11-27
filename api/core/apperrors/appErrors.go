package apperrors

import (
	"errors"
	"strings"
)

var (
	ErrNotFound            = errors.New("This record was not found")
	ErrConflict            = errors.New("Record already exists")
	ErrInvalidInput        = errors.New("Invalid input")
	ErrUnauthorized        = errors.New("Unauthorized access")
	ErrForbidden           = errors.New("Forbidden access")
	ErrTooManyRequests     = errors.New("too many requests")
	ErrUnprocessableEntity = errors.New("unprocessable Entity ")
	ErrPayloadTooLarge     = errors.New("Payload too large")

	ErrCepNotFound              = errors.New("CEP não encontrado.")
	ErrAddressLookupUnavailable = errors.New("Serviço de consulta de CEP temporariamente indisponível. Tente novamente em instantes.")

	ErrServiceUnavailable     = errors.New("O assistente está temporariamente indisponível. Tente novamente em instantes.")
	ErrImageUploadUnavailable = errors.New("Serviço de upload de imagem temporariamente indisponível. Tente novamente em instantes.")

	ErrDatabaseUnavailable = errors.New("Database unavailable")
)

// FieldError representa uma violacao de validacao associada a um campo especifico.
type FieldError struct {
	Field   string
	Message string
}

// ValidationError transporta erros de validacao de dominio/aplicacao por campo
// preservando a compatibilidade com errors.Is sobre o erro sentinela Base (400 ou 422).
type ValidationError struct {
	Base    error
	Message string
	Fields  []FieldError
}

func (e *ValidationError) Error() string {
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if len(e.Fields) > 0 && strings.TrimSpace(e.Fields[0].Message) != "" {
		return e.Fields[0].Message
	}
	if e.Base != nil {
		return e.Base.Error()
	}
	return ErrInvalidInput.Error()
}

func (e *ValidationError) Unwrap() error {
	if e.Base != nil {
		return e.Base
	}
	return ErrInvalidInput
}

func NewValidationError(base error, message string, fields ...FieldError) *ValidationError {
	if base == nil {
		base = ErrInvalidInput
	}
	return &ValidationError{
		Base:    base,
		Message: message,
		Fields:  fields,
	}
}
