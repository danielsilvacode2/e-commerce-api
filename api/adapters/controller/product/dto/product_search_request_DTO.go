package dto

import (
	fielderrors "e-commerce/adapters/controller/fieldErrors"
	"strings"
	"unicode/utf8"
)

const (
	SearchQueryMinLength = 2
	SearchQueryMaxLength = 100
)

type ProductSearchRequestDTO struct {
	Query string
}

// Validator normaliza o termo (trim) e valida tamanho minimo e maximo
func (dto *ProductSearchRequestDTO) Validator() []fielderrors.FieldValidateErrors {

	var listFieldErrors []fielderrors.FieldValidateErrors

	dto.Query = strings.TrimSpace(dto.Query)

	size := utf8.RuneCountInString(dto.Query)

	if size < SearchQueryMinLength || size > SearchQueryMaxLength {
		listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
			Field:   "q",
			Message: "the q field is required and must have between 2 and 100 characters",
		})
	}

	return listFieldErrors
}
