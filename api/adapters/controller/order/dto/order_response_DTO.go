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

type OrderResponseDTO struct {
	Id                    uuid.UUID              `json:"id"`
	User                  UserOrderResponseDTO   `json:"user"`
	Status                string                 `json:"status"`
	ZipCode               string                 `json:"zip_code"`
	TrackingCode          string                 `json:"tracking_code,omitempty"`
	RefundReason          string                 `json:"refund_reason,omitempty"`
	RefundRejectionReason string                 `json:"refund_rejection_reason,omitempty"`
	Total                 float64                `json:"total"`
	Items                 []OrderItemResponseDTO `json:"items"`
	UpdateAt              time.Time              `json:"update_at"`
	CreateAt              time.Time              `json:"created_at"`
}

type UserOrderResponseDTO struct {
	Id    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}
