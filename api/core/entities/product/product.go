package product

import (
	"time"

	"github.com/google/uuid"
)

type Product struct {
	Id           uuid.UUID
	Sku          string
	Name         string
	Description  string
	ProductImage string
	Price        float64
	Quantity     int64
	CreateAt     time.Time
	UpdateAt     time.Time
}

type ProductInterface interface{
	CreateProduct( product Product) error
	ReadProdutById( Id string) (Product, error)
	ReadAllProduct() ([]Product, error)
	DeleteProductById(Id string)
}
