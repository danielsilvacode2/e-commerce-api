package dto

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatusHistoryResponseDTO struct {
	Id       uuid.UUID `json:"id"`
	Status   string    `json:"status"`
	CreateAt time.Time `json:"created_at"`
}
