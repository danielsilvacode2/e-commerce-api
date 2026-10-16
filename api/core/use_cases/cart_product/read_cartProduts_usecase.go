package use_cases

import entities "e-commerce/core/entities/cart_product"

type ReadAllCartProductsUseCase struct {
	repository entities.CartProductRepository
}

func NewReadAllCartProductsUseCase(repository entities.CartProductRepository) *ReadAllCartProductsUseCase {
	return &ReadAllCartProductsUseCase{
		repository: repository,
	}
}

func (useCase *ReadAllCartProductsUseCase) UseCase(userId string) ([]entities.CartProduct, error) {

	listCartProducts, err := useCase.repository.ReadAllProductsCart(userId)

	if err != nil {
		return nil, err
	}

	return listCartProducts, err
}
