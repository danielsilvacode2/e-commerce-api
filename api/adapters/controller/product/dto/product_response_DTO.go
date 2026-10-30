package dto

import (
	"time"

	"github.com/google/uuid"
)

type PaginatedProducts struct {
	Products   []ProcuctResponseDTO `json:"data"`
	Page       int                  `json:"page"`
	Limit      int                  `json:"limit"`
	TotalRows  int64                `json:"total_rows"`
	TotalPages int                  `json:"total_pages"`
}

type ProcuctResponseDTO struct {
	Id          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Sku         string    `json:"sku"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Quantity    int64     `json:"quantity"`
	Image       string    `json:"image"`
	UpdateAt    time.Time `json:"update_at"`
	CreateAt    time.Time `json:"created_at"`
}
