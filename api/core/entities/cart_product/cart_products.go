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
	AddProductCart(product entitiesProduct.Product, cartProductId string) error
	RemoveProduct(product entitiesProduct.Product, cartProductId string) error
	ReadAllProductsCart(cartId string) []CartProduct
	UpdateIsSelect(bool, cartProductId string) error
}
