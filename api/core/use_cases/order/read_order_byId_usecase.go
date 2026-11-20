package use_cases

import (
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/order"
)

const roleAdmin = "admin"

type ReadOrderByIdUseCase struct {
	repository entities.OrderRepository
}

func NewReadOrderByIdUseCase(repository entities.OrderRepository) *ReadOrderByIdUseCase {
	return &ReadOrderByIdUseCase{
		repository: repository,
	}
}

// UseCase retorna o pedido (com itens) se o usuario for o dono ou admin
func (useCase *ReadOrderByIdUseCase) UseCase(orderId string, userId string, role string) (*entities.Order, error) {

	order, err := useCase.repository.ReadOrderById(orderId)

	if err != nil {
		return nil, err
	}

	if role != roleAdmin && !order.BelongsTo(userId) {
		return nil, apperrors.ErrForbidden
	}

	return order, nil
}
