package use_cases

import (
	entities "e-commerce/core/entities/order_status_history"
	useCasesOrder "e-commerce/core/use_cases/order"
)

type ReadOrderStatusHistoryUseCase struct {
	repository    entities.OrderStatusHistoryRepository
	readOrderById *useCasesOrder.ReadOrderByIdUseCase
}

func NewReadOrderStatusHistoryUseCase(repository entities.OrderStatusHistoryRepository, readOrderById *useCasesOrder.ReadOrderByIdUseCase) *ReadOrderStatusHistoryUseCase {
	return &ReadOrderStatusHistoryUseCase{
		repository:    repository,
		readOrderById: readOrderById,
	}
}

func (useCase *ReadOrderStatusHistoryUseCase) UseCase(orderId string, userId string, role string) ([]entities.OrderStatusHistory, error) {

	if _, err := useCase.readOrderById.UseCase(orderId, userId, role); err != nil {
		return nil, err
	}

	listHistory, err := useCase.repository.ReadHistoryByOrderId(orderId)

	if err != nil {
		return nil, err
	}

	return listHistory, nil
}
