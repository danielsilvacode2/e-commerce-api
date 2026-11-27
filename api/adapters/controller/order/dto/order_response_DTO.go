package dto

import (
	"time"

	"github.com/google/uuid"
)

type PaginatedOrdersDTO struct {
	Orders     []OrderResponseDTO `json:"data"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	TotalRows  int64              `json:"total_rows"`
	TotalPages int                `json:"total_pages"`
}

type OrderAddressResponseDTO struct {
	Id           uuid.UUID `json:"id"`
	Cep          string    `json:"cep"`
	Street       string    `json:"street"`
	Number       string    `json:"number"`
	Complement   string    `json:"complement,omitempty"`
	Neighborhood string    `json:"neighborhood,omitempty"`
	City         string    `json:"city"`
	State        string    `json:"state"`
}

type OrderResponseDTO struct {
	Id                       uuid.UUID                `json:"id"`
	User                     UserOrderResponseDTO     `json:"user"`
	Address                  *OrderAddressResponseDTO `json:"address"`
	Status                   string                   `json:"status"`
	AddressValidationWarning string                   `json:"address_validation_warning,omitempty"`
	TrackingCode             string                   `json:"tracking_code,omitempty"`
	RefundReason             string                   `json:"refund_reason,omitempty"`
	RefundRejectionReason    string                   `json:"refund_rejection_reason,omitempty"`
	Total                    float64                  `json:"total"`
	Items                    []OrderItemResponseDTO   `json:"items"`
	UpdateAt                 time.Time                `json:"update_at"`
	CreateAt                 time.Time                `json:"created_at"`
}

type UserOrderResponseDTO struct {
	Id    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}
