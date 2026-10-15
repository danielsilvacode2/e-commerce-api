package use_cases

import entities "e-commerce/core/entities/product"

type ReadAllProductsUseCase struct {
	repository entities.ProductRepository
}

func NewReadAllProductsUseCase(repository entities.ProductRepository) *ReadAllProductsUseCase {
	return &ReadAllProductsUseCase{
		repository: repository,
	}
}

func (useCase *ReadAllProductsUseCase) UseCase() ([]entities.Product, error) {
	listProduct, err := useCase.repository.ReadAllProduct()

	if err != nil {
		return nil, err
	}

	return listProduct, nil
}
