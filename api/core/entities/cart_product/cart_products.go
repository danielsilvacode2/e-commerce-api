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
	DecreaseProductCart(cartProductId string, quantity int64) error
	RemoveProductCart( ID string) error
	ReadAllProductsCart(userId string) ([]CartProduct, error)
	ReadProductsCart(cartProductId string) (*CartProduct, error)
	UpdateIsSelect(valueSelect bool, cartProductId string) error
	UpdateAllIsSelect(valueSelect bool, userID string) error
}
