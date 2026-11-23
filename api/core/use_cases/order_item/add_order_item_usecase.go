package use_cases

import (
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/order_item"
	useCasesOrder "e-commerce/core/use_cases/order"
	useCasesProduct "e-commerce/core/use_cases/product"

	"github.com/google/uuid"
)

type AddOrderItemUseCase struct {
	repository      entities.OrderItemRepository
	readOrderById   *useCasesOrder.ReadOrderByIdUseCase
	readProductById *useCasesProduct.ReadProductByIdUseCase
}

func NewAddOrderItemUseCase(
	repository entities.OrderItemRepository,
	readOrderById *useCasesOrder.ReadOrderByIdUseCase,
	readProductById *useCasesProduct.ReadProductByIdUseCase,
) *AddOrderItemUseCase {
	return &AddOrderItemUseCase{
		repository:      repository,
		readOrderById:   readOrderById,
		readProductById: readProductById,
	}
}

// UseCase adiciona um produto ao pedido (somente em PENDING_PAYMENT).
// Se o produto ja estiver no pedido, a quantidade e somada ao item existente.
func (useCase *AddOrderItemUseCase) UseCase(item entities.OrderItem, orderId string, userId string, role string) error {

	if item.Quantity < 1 {
		return apperrors.ErrInvalidInput
	}

	order, err := useCase.readOrderById.UseCase(orderId, userId, role)

	if err != nil {
		return err
	}

	if !order.CanModifyItems() {
		return apperrors.ErrUnprocessableEntity
	}

	product, err := useCase.readProductById.UseCase(item.Product.Id.String())

	if err != nil {
		return err
	}

	if product.Quantity < item.Quantity {
		return apperrors.ErrUnprocessableEntity
	}

	for _, existing := range order.Items {

		if existing.Product.Id == product.Id {
			return useCase.repository.UpdateOrderItemQuantity(orderId, existing.Id.String(), existing.Quantity+item.Quantity)
		}
	}

	item.Id = uuid.New()
	item.OrderId = order.Id
	item.Product = product
	item.UnitPrice = product.Price

	if err = useCase.repository.AddOrderItem(item); err != nil {
		return err
	}

	return nil
}
