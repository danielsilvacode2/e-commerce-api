package use_cases

import (
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/order"
)

type CancelOrderUseCase struct {
	repository    entities.OrderRepository
	readOrderById *ReadOrderByIdUseCase
}

func NewCancelOrderUseCase(repository entities.OrderRepository, readOrderById *ReadOrderByIdUseCase) *CancelOrderUseCase {
	return &CancelOrderUseCase{
		repository:    repository,
		readOrderById: readOrderById,
	}
}

// UseCase cancela o pedido e devolve o estoque.
// O dono so pode cancelar enquanto o pedido ainda nao foi pago (PENDING_PAYMENT);
// pedidos pagos em PREPARING devem usar o fluxo de estorno (RequestRefund).
func (useCase *CancelOrderUseCase) UseCase(orderId string, userId string, role string) error {

	order, err := useCase.readOrderById.UseCase(orderId, userId, role)

	if err != nil {
		return err
	}

	if role != roleAdmin && order.Status != entities.OrderStatusPendingPayment {
		return apperrors.ErrUnprocessableEntity
	}

	if !order.CanTransitionTo(entities.OrderStatusCanceled) {
		return apperrors.ErrUnprocessableEntity
	}

	if err = useCase.repository.UpdateOrderStatus(orderId, order.Status, entities.OrderStatusCanceled, true); err != nil {
		return err
	}

	return nil
}
