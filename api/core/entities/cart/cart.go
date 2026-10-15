package entities

import (
	entities "e-commerce/core/entities/user"
	"time"

	"github.com/google/uuid"
)

type Cart struct {
	Id       uuid.UUID
	User     *entities.User
	UpdateAt time.Time
	CreateAt time.Time
}

type CartRepository interface{
	CreateCart(Cart) error
	ReadCartByUserId(userId string) (*Cart, error)
}