package dto

import "github.com/google/uuid"

type ProcuctCartResponseDTO struct {
	Id          uuid.UUID `json:"id"`
	IsSelect    bool      `json:"isSelect"`
	Quantity    int64     `json:"quantity"`
	Stock       int64     `json:"stock"`
	ProductId   uuid.UUID `json:"product_id"`
	Name        string    `json:"name"`
	Sku         string    `json:"sku"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Image       string    `json:"image"`
}
