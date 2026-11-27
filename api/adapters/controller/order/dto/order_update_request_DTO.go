package dto

import (
	fielderrors "e-commerce/adapters/controller/fieldErrors"
)

type OrderUpdateRequestDTO struct {
	AddressId string                  `json:"address_id,omitempty"`
	Address   *OrderAddressRequestDTO `json:"address,omitempty"`
}

func (dto *OrderUpdateRequestDTO) Validator() []fielderrors.FieldValidateErrors {
	return validateAddressSelection(dto.AddressId, dto.Address)
}
