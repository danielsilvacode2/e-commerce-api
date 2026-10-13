package dto

import (
	"time"
	"github.com/google/uuid"
)

type UserResponseDTO struct {
	Id         uuid.UUID
	Name       string    `json:"name" `
	Email      string    `json:"email" `
	Role       string    `json:"role"`
	Update_at  time.Time `json:"update_at"`
	Created_at time.Time `json:"created_at"`
}






