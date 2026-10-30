package use_cases

import (
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/cart_product"

)

type DecreaseProductCartUseCase struct {
	readCartUseCase ReadCartProductUseCase
	repository      entities.CartProductRepository
}

func NewDecreaseProductCartUseCase(repository entities.CartProductRepository, readCartUseCase ReadCartProductUseCase) *DecreaseProductCartUseCase {
	return &DecreaseProductCartUseCase{
		repository:      repository,
		readCartUseCase: readCartUseCase,
	}
}

func (useCase *DecreaseProductCartUseCase) UseCase(cartProductId string, quantity int64, userId string) error {

	cart, err := useCase.readCartUseCase.UseCase(cartProductId)

	if err != nil {
		return err
	}

	idUseCart := cart.Cart.User.ID.String()

	if idUseCart != userId {
		return apperrors.ErrForbidden
	}

	
	if (cart.Quantity - quantity) <= 1 {
		return apperrors.ErrUnprocessableEntity
	}

	if err = useCase.repository.DecreaseProductCart(cartProductId, quantity); err != nil {
		return err
	}

	return nil
}
