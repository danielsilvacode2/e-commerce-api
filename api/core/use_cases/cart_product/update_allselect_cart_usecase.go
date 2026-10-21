package use_cases

import entities "e-commerce/core/entities/cart_product"

type UpdateAllIsSelectUseCase struct {
	repository entities.CartProductRepository
}

func NewAllUpdateIsSelectUseCase(repository entities.CartProductRepository) *UpdateAllIsSelectUseCase {
	return &UpdateAllIsSelectUseCase{
		repository: repository,
	}
}

func (useCase UpdateAllIsSelectUseCase) UseCase(valIsSelect bool, useId string) error {
	if err := useCase.repository.UpdateAllIsSelect(valIsSelect, useId); err != nil {
		return err
	}
	return nil
}
