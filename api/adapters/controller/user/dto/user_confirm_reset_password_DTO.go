package dto

import (
	fielderrors "e-commerce/adapters/controller/fieldErrors"
	"net/mail"
	"strings"
)

type UserConfirmResetPasswordDTO struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"newpass"`
}

func (dto *UserConfirmResetPasswordDTO) Validator() []fielderrors.FieldValidateErrors {
	var listErr []fielderrors.FieldValidateErrors

	if _, err := mail.ParseAddress(dto.Email); err != nil {
		listErr = append(listErr, fielderrors.FieldValidateErrors{
			Field:   "email",
			Message: "the email field is invalid and required",
		})
	}

	if len(strings.TrimSpace(dto.Code)) != 6 {
		listErr = append(listErr, fielderrors.FieldValidateErrors{
			Field:   "code",
			Message: "the verification code is invalid",
		})
	}

	if len(dto.NewPassword) < 8 {
		listErr = append(listErr, fielderrors.FieldValidateErrors{
			Field:   "newpass",
			Message: "the password field is very short and requires at least 8 characters",
		})
	} else if len([]byte(dto.NewPassword)) > 72 {
		listErr = append(listErr, fielderrors.FieldValidateErrors{
			Field:   "newpass",
			Message: "the password field must be at most 72 bytes",
		})
	} else if strings.Contains(dto.NewPassword, " ") {
		listErr = append(listErr, fielderrors.FieldValidateErrors{
			Field:   "newpass",
			Message: "the password field cannot contain spaces",
		})
	}

	return listErr
}
