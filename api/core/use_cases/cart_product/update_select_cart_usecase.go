package use_cases

import (
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/cart_product"
)

type UpdateIsSelectUseCase struct {
	repository      entities.CartProductRepository
	readCartProduct ReadCartProductUseCase
}

func NewUpdateIsSelectUseCase(repository entities.CartProductRepository,readCartProduct ReadCartProductUseCase) *UpdateIsSelectUseCase {
	return &UpdateIsSelectUseCase{
		repository: repository,
		readCartProduct: readCartProduct,
	}
}

func (useCase *UpdateIsSelectUseCase) UseCase(valueSelect bool, userID string, cartProductId string) error {

	cartProduct, err := useCase.readCartProduct.UseCase(cartProductId)

	if err != nil {
		return err
	}

	cartProductUserId := cartProduct.Cart.User.ID.String()

	if cartProductUserId != userID {
		return apperrors.ErrForbidden
	}

	if err = useCase.repository.UpdateIsSelect(valueSelect, cartProductId); err != nil {
		return err
	}

	return nil
}
