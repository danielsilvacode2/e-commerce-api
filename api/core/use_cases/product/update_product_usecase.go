package use_cases

import entities "e-commerce/core/entities/product"

type UpdateProductUseCase struct {
	repository entities.ProductRepository
}

func NewUpdateProductUseCase(repository entities.ProductRepository) *UpdateProductUseCase {
	return &UpdateProductUseCase{
		repository: repository,
	}
}

func (useCase * UpdateProductUseCase)  UseCase(product entities.Product, ID string) error {
	if err :=useCase.repository.UpdateProduct(product, ID); err != nil {
		return err
	}
	return nil
}
