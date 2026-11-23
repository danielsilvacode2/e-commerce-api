package dto

import (
	fielderrors "e-commerce/adapters/controller/fieldErrors"
	"strings"
)

type OrderRefundRequestDTO struct {
	Reason string `json:"reason"`
}

type OrderRefundRejectRequestDTO struct {
	Reason string `json:"reason"`
}

func (dto *OrderRefundRejectRequestDTO) Validator() []fielderrors.FieldValidateErrors {
	var listFieldErrors []fielderrors.FieldValidateErrors

	if strings.TrimSpace(dto.Reason) == "" {
		listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
			Field:   "reason",
			Message: "a justificativa da recusa do estorno é obrigatória",
		})
	}

	return listFieldErrors
}
