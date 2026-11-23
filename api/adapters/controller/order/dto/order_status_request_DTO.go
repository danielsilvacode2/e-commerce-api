package dto

import (
	fielderrors "e-commerce/adapters/controller/fieldErrors"
	entities "e-commerce/core/entities/order"
	"strings"
)

type OrderStatusRequestDTO struct {
	Status       string `json:"status"`
	TrackingCode string `json:"tracking_code"`
}

func (dto *OrderStatusRequestDTO) Validator() []fielderrors.FieldValidateErrors {

	var listFieldErrors []fielderrors.FieldValidateErrors

	if !entities.IsValidOrderStatus(strings.ToUpper(strings.TrimSpace(dto.Status))) {
		listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
			Field:   "status",
			Message: "the status field is invalid (PENDING_PAYMENT, PREPARING, SHIPPED, DELIVERED, CANCELED, REFUND_REQUESTED, REFUNDED)",
		})
	}

	return listFieldErrors
}
