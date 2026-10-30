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

func (useCase *ReadAllProductsUseCase) UseCase(page int, limit int) (*entities.PaginatedProducts, error) {
	listProduct, err := useCase.repository.ReadAllProduct(page, limit)

	if err != nil {
		return nil, err
	}

	return listProduct, nil
}
