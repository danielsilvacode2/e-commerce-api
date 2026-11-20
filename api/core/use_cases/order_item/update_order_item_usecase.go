package use_cases

import (
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/order_item"
	useCasesOrder "e-commerce/core/use_cases/order"
)

type UpdateOrderItemUseCase struct {
	repository    entities.OrderItemRepository
	readOrderById *useCasesOrder.ReadOrderByIdUseCase
}

func NewUpdateOrderItemUseCase(repository entities.OrderItemRepository, readOrderById *useCasesOrder.ReadOrderByIdUseCase) *UpdateOrderItemUseCase {
	return &UpdateOrderItemUseCase{
		repository:    repository,
		readOrderById: readOrderById,
	}
}

// UseCase define a nova quantidade de um item do pedido (somente em PREPARING)
func (useCase *UpdateOrderItemUseCase) UseCase(quantity int64, orderId string, orderItemId string, userId string, role string) error {

	if quantity < 1 {
		return apperrors.ErrInvalidInput
	}

	order, err := useCase.readOrderById.UseCase(orderId, userId, role)

	if err != nil {
		return err
	}

	if !order.IsEditable() {
		return apperrors.ErrUnprocessableEntity
	}

	item, err := useCase.repository.ReadOrderItemById(orderItemId)

	if err != nil {
		return err
	}

	if item.OrderId != order.Id {
		return apperrors.ErrNotFound
	}

	if err = useCase.repository.UpdateOrderItemQuantity(orderId, orderItemId, quantity); err != nil {
		return err
	}

	return nil
}
