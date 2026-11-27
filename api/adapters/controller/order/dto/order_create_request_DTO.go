package dto

import (
	fielderrors "e-commerce/adapters/controller/fieldErrors"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var cepRegex = regexp.MustCompile(`^[0-9]{5}-?[0-9]{3}$`)

type OrderAddressRequestDTO struct {
	Cep          string `json:"cep"`
	Street       string `json:"street"`
	Number       string `json:"number"`
	Complement   string `json:"complement,omitempty"`
	Neighborhood string `json:"neighborhood,omitempty"`
	City         string `json:"city"`
	State        string `json:"state"`
}

func (dto *OrderAddressRequestDTO) Validator() []fielderrors.FieldValidateErrors {
	var listFieldErrors []fielderrors.FieldValidateErrors

	if !cepRegex.MatchString(strings.TrimSpace(dto.Cep)) {
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

func validateAddressSelection(addressId string, address *OrderAddressRequestDTO) []fielderrors.FieldValidateErrors {
	trimmedId := strings.TrimSpace(addressId)

	if trimmedId != "" && address != nil {
		return []fielderrors.FieldValidateErrors{
			{
				Field:   "address_id",
				Message: "Informe um endereço salvo (address_id) ou um novo endereço (address), não ambos.",
			},
		}
	}

	if trimmedId == "" && address == nil {
		return []fielderrors.FieldValidateErrors{
			{
				Field:   "address_id",
				Message: "Selecione um endereço salvo ou informe um novo endereço para entrega.",
			},
		}
	}

	if trimmedId != "" {
		if _, err := uuid.Parse(trimmedId); err != nil {
			return []fielderrors.FieldValidateErrors{
				{
					Field:   "address_id",
					Message: "Identificador de endereço salvo inválido.",
				},
			}
		}
		return nil
	}

	return address.Validator()
}

type OrderCreateRequestDTO struct {
	AddressId string                  `json:"address_id,omitempty"`
	Address   *OrderAddressRequestDTO `json:"address,omitempty"`
}

func (dto *OrderCreateRequestDTO) Validator() []fielderrors.FieldValidateErrors {
	return validateAddressSelection(dto.AddressId, dto.Address)
}
