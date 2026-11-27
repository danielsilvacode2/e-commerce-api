package dto

import (
	fielderrors "e-commerce/adapters/controller/fieldErrors"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var addressCepRegex = regexp.MustCompile(`^[0-9]{5}-?[0-9]{3}$`)

type CreateAddressRequestDTO struct {
	Cep          string `json:"cep"`
	Street       string `json:"street"`
	Number       string `json:"number"`
	Complement   string `json:"complement,omitempty"`
	Neighborhood string `json:"neighborhood,omitempty"`
	City         string `json:"city"`
	State        string `json:"state"`
}

func (dto *CreateAddressRequestDTO) Validator() []fielderrors.FieldValidateErrors {
	var listFieldErrors []fielderrors.FieldValidateErrors

	if !addressCepRegex.MatchString(strings.TrimSpace(dto.Cep)) {
		listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
			Field:   "cep",
			Message: "Informe um CEP válido com 8 dígitos (ex.: 00000-000).",
		})
	}
	if strings.TrimSpace(dto.Street) == "" {
		listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
			Field:   "street",
			Message: "A rua é obrigatória.",
		})
	}
	if strings.TrimSpace(dto.Number) == "" {
		listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
			Field:   "number",
			Message: "O número é obrigatório (informe S/N se não houver).",
		})
	}
	if strings.TrimSpace(dto.City) == "" {
		listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
			Field:   "city",
			Message: "A cidade é obrigatória.",
		})
	}
	if strings.TrimSpace(dto.State) == "" {
		listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
			Field:   "state",
			Message: "O estado (UF) é obrigatório.",
		})
	}

	return listFieldErrors
}

type AddressLookupResponseDTO struct {
	Cep          string `json:"cep"`
	Street       string `json:"street"`
	Neighborhood string `json:"neighborhood,omitempty"`
	City         string `json:"city"`
	State        string `json:"state"`
}

type SavedAddressResponseDTO struct {
	Id                       uuid.UUID `json:"id"`
	Cep                      string    `json:"cep"`
	Street                   string    `json:"street"`
	Number                   string    `json:"number"`
	Complement               string    `json:"complement,omitempty"`
	Neighborhood             string    `json:"neighborhood,omitempty"`
	City                     string    `json:"city"`
	State                    string    `json:"state"`
	AddressValidationWarning string    `json:"address_validation_warning,omitempty"`
}
