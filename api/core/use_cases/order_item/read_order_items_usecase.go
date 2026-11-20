package use_cases

import (
	entities "e-commerce/core/entities/order_item"
	useCasesOrder "e-commerce/core/use_cases/order"
)

type ReadOrderItemsUseCase struct {
	repository    entities.OrderItemRepository
	readOrderById *useCasesOrder.ReadOrderByIdUseCase
}

func NewReadOrderItemsUseCase(repository entities.OrderItemRepository, readOrderById *useCasesOrder.ReadOrderByIdUseCase) *ReadOrderItemsUseCase {
	return &ReadOrderItemsUseCase{
		repository:    repository,
		readOrderById: readOrderById,
	}
}

func (useCase *ReadOrderItemsUseCase) UseCase(orderId string, userId string, role string) ([]entities.OrderItem, error) {

	if _, err := useCase.readOrderById.UseCase(orderId, userId, role); err != nil {
		return nil, err
	}

	listItems, err := useCase.repository.ReadOrderItemsByOrderId(orderId)

	if err != nil {
		return nil, err
	}

	return listItems, nil
}
