package dto

import "github.com/google/uuid"

type UserCartResponseDTO struct {
	Id    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Role  string    `json:"role"`
}
