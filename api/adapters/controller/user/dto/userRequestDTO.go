package dto

import (
	"e-commerce/adapters/controller/apperrors"
	"net/mail"
	"strings"
)

type UserRequestDTO struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (dto *UserRequestDTO) Validator() []apperrors.FieldValidateErrors {

	var listErrCampo []apperrors.FieldValidateErrors

	if len(strings.TrimSpace(dto.Name)) < 2 {
		listErrCampo = append(listErrCampo, apperrors.FieldValidateErrors{
			Field:   "name",
			Message: "the name field is invalid and required",
		})
	}

	_, err := mail.ParseAddress(dto.Email)

	if err != nil {
		listErrCampo = append(listErrCampo, apperrors.FieldValidateErrors{
			Field:   "e-mail",
			Message: "the email field is invalid and required",
		})
	}

	if len(dto.Password) < 8 {
		listErrCampo = append(listErrCampo, apperrors.FieldValidateErrors{
			Field:   "password",
			Message: "the password field is and very short and required 8 caracters",
		})
	} else {

		if strings.Contains(dto.Password, " ") {
			listErrCampo = append(listErrCampo, apperrors.FieldValidateErrors{
				Message: "the password field cannot	contain spaces",
			})
		}

	}

	return listErrCampo
}