package apperrors

import "errors"

var (
	RegisterNotFound = errors.New("This register not found")
	RegisterConflict = errors.New("This register already exists")
)