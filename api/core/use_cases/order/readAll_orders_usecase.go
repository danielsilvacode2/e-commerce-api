package use_cases

import (
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/order"
)

type ReadAllOrdersUseCase struct {
	repository entities.OrderRepository
}

func NewReadAllOrdersUseCase(repository entities.OrderRepository) *ReadAllOrdersUseCase {
	return &ReadAllOrdersUseCase{
		repository: repository,
	}
}

// UseCase lista todos os pedidos com paginacao e filtros opcionais de status e usuario
func (useCase *ReadAllOrdersUseCase) UseCase(filter entities.OrderFilter) (*entities.PaginatedOrders, error) {

	if filter.Status != "" && !entities.IsValidOrderStatus(filter.Status) {
		return nil, apperrors.ErrInvalidInput
	}

	paginated, err := useCase.repository.ReadAllOrders(filter)

	if err != nil {
		return nil, err
	}

	return paginated, nil
}
