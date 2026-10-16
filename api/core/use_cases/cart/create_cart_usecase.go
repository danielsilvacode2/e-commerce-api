package use_cases

import entities "e-commerce/core/entities/cart"

type CreateCartUseCase struct {
	repository entities.CartRepository
}

func NewCreateCartUseCase(repository entities.CartRepository) *CreateCartUseCase {
	return &CreateCartUseCase{
		repository: repository,
	}
}

func (useCase *CreateCartUseCase) UseCase(cart entities.Cart) error {
	if err := useCase.repository.CreateCart(cart); err != nil {
		return err
	}
	return nil
}
