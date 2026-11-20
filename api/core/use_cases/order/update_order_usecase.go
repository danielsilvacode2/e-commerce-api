package use_cases

import (
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/order"
)

type UpdateOrderUseCase struct {
	repository    entities.OrderRepository
	readOrderById *ReadOrderByIdUseCase
}

func NewUpdateOrderUseCase(repository entities.OrderRepository, readOrderById *ReadOrderByIdUseCase) *UpdateOrderUseCase {
	return &UpdateOrderUseCase{
		repository:    repository,
		readOrderById: readOrderById,
	}
}

// UseCase altera o CEP de entrega; permitido apenas enquanto o pedido esta em PREPARING
func (useCase *UpdateOrderUseCase) UseCase(zipCode string, orderId string, userId string, role string) error {

	order, err := useCase.readOrderById.UseCase(orderId, userId, role)

	if err != nil {
		return err
	}

	if !order.IsEditable() {
		return apperrors.ErrUnprocessableEntity
	}

	if err = useCase.repository.UpdateOrderZipCode(zipCode, orderId); err != nil {
		return err
	}

	return nil
}
