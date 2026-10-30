package dto

import (


	"github.com/google/uuid"
)

type CartResponseDTO struct {
	Id          uuid.UUID  `json:"id"`
	User        UserCartResponseDTO `json:"user"`
	ListProduct []ProcuctCartResponseDTO `json:"list"`
}


