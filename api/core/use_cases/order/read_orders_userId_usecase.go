package use_cases

import (
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/order"
)

type ReadOrdersByUserIdUseCase struct {
	repository entities.OrderRepository
}

func NewReadOrdersByUserIdUseCase(repository entities.OrderRepository) *ReadOrdersByUserIdUseCase {
	return &ReadOrdersByUserIdUseCase{
		repository: repository,
	}
}

// UseCase lista os pedidos de um usuario com paginacao e filtro opcional de status
func (useCase *ReadOrdersByUserIdUseCase) UseCase(userId string, page int, limit int, status string) (*entities.PaginatedOrders, error) {

	if status != "" && !entities.IsValidOrderStatus(status) {
		return nil, apperrors.ErrInvalidInput
	}

	paginated, err := useCase.repository.ReadAllOrders(entities.OrderFilter{
		Page:   page,
		Limit:  limit,
		Status: status,
		UserId: userId,
	})

	if err != nil {
		return nil, err
	}

	return paginated, nil
}
