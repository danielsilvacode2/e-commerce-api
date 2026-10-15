package dto

import (
	fielderrors "e-commerce/adapters/controller/fieldErrors"
	"net/mail"
	"strings"
)

type UserUpdateRequestDTO struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (dto *UserUpdateRequestDTO) Validator() []fielderrors.FieldValidateErrors {

	var listErrCampo []fielderrors.FieldValidateErrors

	if len(strings.TrimSpace(dto.Name)) < 2 {
		listErrCampo = append(listErrCampo, fielderrors.FieldValidateErrors{
			Field:   "name",
			Message: "the name field is invalid and required",
		})
	}

	_, err := mail.ParseAddress(dto.Email)

	if err != nil {
		listErrCampo = append(listErrCampo, fielderrors.FieldValidateErrors{
			Field:   "e-mail",
			Message: "the email field is invalid and required",
		})
	}

	return listErrCampo
}