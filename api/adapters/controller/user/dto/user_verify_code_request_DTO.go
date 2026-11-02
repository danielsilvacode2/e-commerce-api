package dto

import(
	fielderrors "e-commerce/adapters/controller/fieldErrors"
	"net/mail"
)

type UserVerifyCodeRequestDTO struct {
	Email            string `json:"email"`
	VerificationCode string `json:"verification_code"`
}

func (dto *UserVerifyCodeRequestDTO) Validator() []fielderrors.FieldValidateErrors {
	var listErrCampo []fielderrors.FieldValidateErrors
	_, err := mail.ParseAddress(dto.Email)

	if err != nil {
		listErrCampo = append(listErrCampo, fielderrors.FieldValidateErrors{
			Field:   "e-mail",
			Message: "the email field is invalid and required",
		})
	}

	if code := dto.VerificationCode; len(code) != 6 {
		listErrCampo = append(listErrCampo, fielderrors.FieldValidateErrors{
			Field:   "verification_code",
			Message: "the verification code field is invalid and required",
		})
	}

	return listErrCampo
}
