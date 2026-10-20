package use_cases

import entities "e-commerce/core/entities/cart_product"

type ReadCartProductUseCase struct {
	repository entities.CartProductRepository
}

func NewReadCartProductUseCase(repository entities.CartProductRepository) *ReadCartProductUseCase {
	return &ReadCartProductUseCase{
		repository: repository,
	}
}

func (useCase *ReadCartProductUseCase) UseCase(cartProductID string) (*entities.CartProduct, error) {
	cartProduct, err := useCase.repository.ReadProductsCart(cartProductID)
	if err != nil {
		return nil, err
	}
	return cartProduct, nil
}
