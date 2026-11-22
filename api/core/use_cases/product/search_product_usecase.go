package use_cases

import (
	"context"
	entities "e-commerce/core/entities/product"
)

type SearchProductsUseCase struct {
	repository entities.ProductRepository
}

func NewSearchProductsUseCase(repository entities.ProductRepository) *SearchProductsUseCase {
	return &SearchProductsUseCase{
		repository: repository,
	}
}

func (useCase *SearchProductsUseCase) UseCase(ctx context.Context, term string, page int, limit int) (*entities.PaginatedProducts, error) {
	return useCase.repository.SearchProducts(ctx, term, page, limit)
}
