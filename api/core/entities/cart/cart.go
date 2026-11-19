package entities

import (
	entities "e-commerce/core/entities/user"
	"time"

	"github.com/google/uuid"
)

const (
	CartStatusActive   = "ACTIVE"
	CartStatusInactive = "INACTIVE"
)

type Cart struct {
	Id       uuid.UUID
	User     *entities.User
	Status   string
	UpdateAt time.Time
	CreateAt time.Time
}

type CartRepository interface {
	CreateCart(Cart) error
	ReadCartByUserId(userId string, status string) (*Cart, error)
}
