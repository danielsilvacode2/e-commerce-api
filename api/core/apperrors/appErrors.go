package apperrors

import "errors"

var (
	ErrNotFound            = errors.New("This record was not found")
	ErrConflict            = errors.New("Record already exists")
	ErrInvalidInput        = errors.New("Invalid input")
	ErrUnauthorized        = errors.New("Unauthorized access")
	ErrForbidden           = errors.New("Forbidden access")
	ErrTooManyRequests     = errors.New("too many requests")
	ErrUnprocessableEntity = errors.New("unprocessable Entity ")
	ErrPayloadTooLarge     = errors.New("Payload too large")

	ErrServiceUnavailable     = errors.New("O assistente está temporariamente indisponível. Tente novamente em instantes.")
	ErrImageUploadUnavailable = errors.New("Serviço de upload de imagem temporariamente indisponível. Tente novamente em instantes.")

	ErrDatabaseUnavailable = errors.New("Database unavailable")
	
)
