package use_cases

import (
	entities "e-commerce/core/entities/order"
)

type DeleteOrderUseCase struct {
	repository entities.OrderRepository
}

func NewDeleteOrderUseCase(repository entities.OrderRepository) *DeleteOrderUseCase {
	return &DeleteOrderUseCase{
		repository: repository,
	}
}

// UseCase remove o pedido (hard delete; o schema nao possui soft delete).
// Se o estoque ainda estiver reservado (PREPARING/SHIPPED), ele e devolvido.
func (useCase *DeleteOrderUseCase) UseCase(orderId string) error {

	order, err := useCase.repository.ReadOrderById(orderId)

	if err != nil {
		return err
	}

	if err = useCase.repository.DeleteOrder(orderId, order.HasReservedStock()); err != nil {
		return err
	}

	return nil
}
