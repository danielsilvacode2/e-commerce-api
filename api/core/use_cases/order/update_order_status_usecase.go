package use_cases

import (
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/order"
)

type UpdateOrderStatusUseCase struct {
	repository    entities.OrderRepository
	readOrderById *ReadOrderByIdUseCase
}

func NewUpdateOrderStatusUseCase(repository entities.OrderRepository, readOrderById *ReadOrderByIdUseCase) *UpdateOrderStatusUseCase {
	return &UpdateOrderStatusUseCase{
		repository:    repository,
		readOrderById: readOrderById,
	}
}

// UseCase aplica uma transicao de status valida (PREPARING -> SHIPPED -> DELIVERED, PREPARING/SHIPPED -> CANCELED).
// Ao cancelar, o estoque dos itens e devolvido.
func (useCase *UpdateOrderStatusUseCase) UseCase(status string, orderId string, userId string, role string) error {

	if !entities.IsValidOrderStatus(status) {
		return apperrors.ErrInvalidInput
	}

	order, err := useCase.readOrderById.UseCase(orderId, userId, role)

	if err != nil {
		return err
	}

	if order.IsFinalized() || !order.CanTransitionTo(status) {
		return apperrors.ErrUnprocessableEntity
	}

	restoreStock := status == entities.OrderStatusCanceled

	if err = useCase.repository.UpdateOrderStatus(orderId, order.Status, status, restoreStock); err != nil {
		return err
	}

	return nil
}
