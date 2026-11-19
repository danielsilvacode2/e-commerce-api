package use_cases

import (
	"e-commerce/core/apperrors"
	entitiesCart "e-commerce/core/entities/cart"
	entities "e-commerce/core/entities/user"

	"github.com/google/uuid"
)

type ReadOrCreateCartUseCase struct {
	repository entitiesCart.CartRepository
}

func NewReadOrCreateCartUseCase (repository entitiesCart.CartRepository) *ReadOrCreateCartUseCase{
	return &ReadOrCreateCartUseCase{
		repository: repository,
	}
}


func (useCase *ReadOrCreateCartUseCase) UseCase(userId string) (*entitiesCart.Cart, error) {

	potentialCart, err := useCase.repository.ReadCartByUserId(userId, entitiesCart.CartStatusActive)

	if err != nil && err != apperrors.ErrNotFound {
		return nil, err
	}

	if potentialCart != nil {
		return potentialCart, nil
	}

	userUUID, err := uuid.Parse(userId)

	if err != nil {
		return nil, err
	}

	cart := entitiesCart.Cart{
		Id:     uuid.New(),
		Status: entitiesCart.CartStatusActive,
		User: &entities.User{
			ID: userUUID,
		},
	}

	if err = useCase.repository.CreateCart(cart); err != nil {
		return nil, err
	}

	return &cart, nil
}
