package dto

import (
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

type UserRequestDTO struct {
	Name     string `json:"name" `
	Email    string `json:"email" `
	Password string `json:"password"`
}

type CampoValidate struct {
	Field   string `json:"campo"`
	Message string `json:"mensagem"`
}

func (dto *UserRequestDTO) Validator() []CampoValidate {

	var listErrCampo []CampoValidate

	if len(strings.TrimSpace(dto.Name)) < 2 {
		listErrCampo = append(listErrCampo, CampoValidate{
			Field:   "name",
			Message: "the name field is invalid and required",
		})
	}

	_, err := mail.ParseAddress(dto.Email)

	if err != nil {
		listErrCampo = append(listErrCampo, CampoValidate{
			Field:   "e-mail",
			Message: "the email field is invalid and required",
		})
	}

	if len(dto.Password) < 8 {
		listErrCampo = append(listErrCampo, CampoValidate{
			Field:   "password",
			Message: "the password field is and very short and required 8 caracters",
		})
	} else {

		if strings.Contains(dto.Password, " ") {
			listErrCampo = append(listErrCampo, CampoValidate{
				Message: "the password field cannot	contain spaces",
			})
		}

	}

	return listErrCampo
}

type UserResponseDTO struct {
	Id         uuid.UUID
	Name       string    `json:"name" `
	Email      string    `json:"email" `
	Role       string    `json:"role"`
	Update_at  time.Time `json:"update_at"`
	Created_at time.Time `json:"created_at"`
}
