package use_cases

import (
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/order_item"
	useCasesOrder "e-commerce/core/use_cases/order"
)

type RemoveOrderItemUseCase struct {
	repository    entities.OrderItemRepository
	readOrderById *useCasesOrder.ReadOrderByIdUseCase
}

func NewRemoveOrderItemUseCase(repository entities.OrderItemRepository, readOrderById *useCasesOrder.ReadOrderByIdUseCase) *RemoveOrderItemUseCase {
	return &RemoveOrderItemUseCase{
		repository:    repository,
		readOrderById: readOrderById,
	}
}

// UseCase remove um item do pedido (somente em PREPARING).
// O ultimo item nao pode ser removido: nesse caso o pedido deve ser cancelado.
func (useCase *RemoveOrderItemUseCase) UseCase(orderId string, orderItemId string, userId string, role string) error {

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

	if len(order.Items) <= 1 {
		return apperrors.ErrUnprocessableEntity
	}

	if err = useCase.repository.RemoveOrderItem(orderId, orderItemId); err != nil {
		return err
	}

	return nil
}
