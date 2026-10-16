package dto

import (


	"github.com/google/uuid"
)

type CartResponseDTO struct {
	Id          uuid.UUID
	User        UserCartResponseDTO
	ListProduct []ProcuctCartResponseDTO
}


