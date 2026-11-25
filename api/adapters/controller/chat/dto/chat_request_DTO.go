package dto

import (
	fielderrors "e-commerce/adapters/controller/fieldErrors"
	"strings"
	"unicode/utf8"
)

const maxMessageLength = 1000

type ChatRequestDTO struct {
	Message string `json:"message"`
}

func (dto *ChatRequestDTO) Validator() []fielderrors.FieldValidateErrors {

	var listFieldErrors []fielderrors.FieldValidateErrors

	message := strings.TrimSpace(dto.Message)

	if message == "" || utf8.RuneCountInString(message) > maxMessageLength {
		listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
			Field:   "message",
			Message: "the message field is required and must have at most 1000 characters",
		})
	}

	return listFieldErrors
}
