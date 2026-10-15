package use_cases

import (
	entities "e-commerce/core/entities/product"

	"github.com/google/uuid"
)

type CreateProductUseCase struct {
	repository entities.ProductRepository
}

func NewCreateProductUseCase(repository entities.ProductRepository) *CreateProductUseCase {
	return &CreateProductUseCase{
		repository: repository,
	}
}

func (useCase *CreateProductUseCase) UseCase(product entities.Product) (*entities.Product, error) {

	product.Id = uuid.New()

	if err := useCase.repository.CreateProduct(product); err != nil {
		return nil, err
	}

	return &product, nil
}
