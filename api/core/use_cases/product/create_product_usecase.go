package use_cases

import (
	entities "e-commerce/core/entities/product"

	"github.com/google/uuid"
)

func CreateProduct( product entities.Product, repository entities.ProductRepository) ( *entities.Product, error) {


	product.Id = uuid.New()

	if err := repository.CreateProduct(product); err != nil {
		return nil,err
	}

	return &product ,nil
}