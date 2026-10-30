package entities

import (
	"time"

	"github.com/google/uuid"
)

type Product struct {
	Id          uuid.UUID
	Sku         string
	Name        string
	Description string
	Image       string
	Price       float64
	Quantity    int64
	CreateAt    time.Time
	UpdateAt    time.Time
}

type PaginatedProducts struct {
	Data       []Product 
	Page       int               
	Limit      int               
	TotalRows  int64              
	TotalPages int                
}

type ProductRepository interface {
	CreateProduct(product Product) error
	ReadProdutById(Id string) (*Product, error)
	ReadAllProduct(page, limit int) (*PaginatedProducts, error)
	DeleteProductById(Id string) error
	UpdateProduct(product Product, ID string) error
}
