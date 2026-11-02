package dto

import(
	fielderrors "e-commerce/adapters/controller/fieldErrors"
	"net/mail"
)

type UserNewVerifyCodeDTO struct {
	Email string `json:"email"`
}

func (dto *UserNewVerifyCodeDTO) Validator() []fielderrors.FieldValidateErrors {
	var listErrCampo []fielderrors.FieldValidateErrors
	_, err := mail.ParseAddress(dto.Email)

	if err != nil {
		listErrCampo = append(listErrCampo, fielderrors.FieldValidateErrors{
			Field:   "e-mail",
			Message: "the email field is invalid and required",
		})
	}

	return listErrCampo
}
