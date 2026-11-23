package use_cases

import (
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/cart_product"
)

type RemoveProductCartUseCase struct {
	ReadProductCart ReadCartProductUseCase
	repository      entities.CartProductRepository
}

func NewRemoveProductCartUseCase(repository entities.CartProductRepository, ReadProductCart ReadCartProductUseCase) *RemoveProductCartUseCase {
	return &RemoveProductCartUseCase{
		repository:      repository,
		ReadProductCart: ReadProductCart,
	}
}

func (useCase RemoveProductCartUseCase) UseCase(ProductCartId string, userId string) error {

	cartProduct, err := useCase.ReadProductCart.UseCase(ProductCartId)

	if err != nil {
		return err
	}

	userCartID := cartProduct.Cart.User.ID.String()

	if userCartID != userId {
		return apperrors.ErrForbidden
	}

	if err = useCase.repository.RemoveProductCart(ProductCartId); err != nil {
		return err
	}
	return nil
}
