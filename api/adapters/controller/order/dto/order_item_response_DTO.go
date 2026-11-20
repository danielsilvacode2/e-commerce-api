package dto

import "github.com/google/uuid"

type OrderItemResponseDTO struct {
	Id        uuid.UUID `json:"id"`
	ProductId uuid.UUID `json:"product_id"`
	Name      string    `json:"name"`
	Sku       string    `json:"sku"`
	Image     string    `json:"image"`
	Quantity  int64     `json:"quantity"`
	UnitPrice float64   `json:"unit_price"`
	Subtotal  float64   `json:"subtotal"`
}
