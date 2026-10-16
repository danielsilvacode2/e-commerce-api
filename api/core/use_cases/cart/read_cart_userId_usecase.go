package use_cases

import entities "e-commerce/core/entities/cart"

type ReadCartByUserId struct {
	repository entities.CartRepository
}


func NeweadCartByUserId (repository entities.CartRepository) *ReadCartByUserId{
	return &ReadCartByUserId{
		repository: repository,
	}
}

func (useCase *ReadCartByUserId) UseCase(userId string) (*entities.Cart, error) {

	cart, err := useCase.repository.ReadCartByUserId(userId)

	if err != nil {
		return nil, err
	}

	return cart, nil

}
