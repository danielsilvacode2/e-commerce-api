package entities

import (
	entitiesCart "e-commerce/core/entities/cart"
	entitiesProduct "e-commerce/core/entities/product"
	"time"

	"github.com/google/uuid"
)

type CartProduct struct {
	Id       uuid.UUID
	Cart     *entitiesCart.Cart
	Product  *entitiesProduct.Product
	Quantity int64
	IsSelect bool
	CreateAt time.Time
	UpdateAt time.Time
}

type CartProductRepository interface {
	AddProductCart(cartProduct  CartProduct) error
	RemoveProduct( cartProduct CartProduct) error
	ReadAllProductsCart(userId string) ([]CartProduct, error)
	UpdateIsSelect(bool, cartProductId string) error
}
