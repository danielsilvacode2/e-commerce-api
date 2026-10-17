package use_cases

import (
	entitiesCart "e-commerce/core/entities/cart"
	entitiesCartProduct "e-commerce/core/entities/cart_product"
	use_cases "e-commerce/core/use_cases/cart"

	"github.com/google/uuid"
)

type AddProductUseCase struct {
	repository       entitiesCartProduct.CartProductRepository
	readOrCreateCart use_cases.ReadOrCreateCartUseCase
}

func NewAddProductUseCase(repository entitiesCartProduct.CartProductRepository, readOrCreateCart use_cases.ReadOrCreateCartUseCase) *AddProductUseCase {
	return &AddProductUseCase{
		repository:       repository,
		readOrCreateCart: readOrCreateCart,
	}
}

func (useCase *AddProductUseCase) UseCase(cartProduct entitiesCartProduct.CartProduct, userId string) error {

	cart, err := useCase.readOrCreateCart.UseCase(userId)

	if err != nil {
		return err
	}

	cartProduct.Cart = &entitiesCart.Cart{Id: cart.Id}

	cartProduct.Id = uuid.New()
	cartProduct.IsSelect = true

	if err = useCase.repository.AddProductCart(cartProduct); err != nil {
		return err
	}

	return nil
}
