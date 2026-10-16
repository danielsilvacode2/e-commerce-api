package dto

import (
	fielderrors "e-commerce/adapters/controller/fieldErrors"

	"github.com/google/uuid"
)

type CartRequestDTO struct {
	ProductId string `json:"product_id"`
	Quantity  int64  `json:"quantity"`
}

func (dto *CartRequestDTO) Validator() []fielderrors.FieldValidateErrors {

	var listFieldErrors []fielderrors.FieldValidateErrors

	if dto.Quantity < 1 {
		listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
			Field:   "quantity",
			Message: "The quantity is mandatory, and the minimum required is 1 product.",
		})
	}

	if err := uuid.Validate(dto.ProductId); err != nil {
		listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
			Field:   "product_id",
			Message: "The product is mandatory",
		})
	}

	return listFieldErrors

}
