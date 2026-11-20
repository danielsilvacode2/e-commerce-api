package dto

import (
	fielderrors "e-commerce/adapters/controller/fieldErrors"
)

type OrderItemUpdateRequestDTO struct {
	Quantity int64 `json:"quantity"`
}

func (dto *OrderItemUpdateRequestDTO) Validator() []fielderrors.FieldValidateErrors {

	var listFieldErrors []fielderrors.FieldValidateErrors

	if dto.Quantity < 1 {
		listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
			Field:   "quantity",
			Message: "The quantity is mandatory, and the minimum required is 1 product.",
		})
	}

	return listFieldErrors
}
