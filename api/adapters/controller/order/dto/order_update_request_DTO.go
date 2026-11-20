package dto

import (
	fielderrors "e-commerce/adapters/controller/fieldErrors"
	"strings"
)

type OrderUpdateRequestDTO struct {
	ZipCode string `json:"zip_code"`
}

func (dto *OrderUpdateRequestDTO) Validator() []fielderrors.FieldValidateErrors {

	var listFieldErrors []fielderrors.FieldValidateErrors

	if !zipCodeRegex.MatchString(strings.TrimSpace(dto.ZipCode)) {
		listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
			Field:   "zip_code",
			Message: "the zip_code field is invalid and required (format 00000-000 or 00000000)",
		})
	}

	return listFieldErrors
}
