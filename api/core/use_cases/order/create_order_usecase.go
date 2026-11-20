package use_cases

import (
	"e-commerce/core/apperrors"
	entitiesOrder "e-commerce/core/entities/order"
	entitiesOrderItem "e-commerce/core/entities/order_item"
	useCasesCartProduct "e-commerce/core/use_cases/cart_product"
	useCasesUser "e-commerce/core/use_cases/user"

	"github.com/google/uuid"
)

type CreateOrderUseCase struct {
	repository          entitiesOrder.OrderRepository
	readUserById        *useCasesUser.ReadUserByIdUseCase
	readAllCartProducts *useCasesCartProduct.ReadAllCartProductsUseCase
}

func NewCreateOrderUseCase(
	repository entitiesOrder.OrderRepository,
	readUserById *useCasesUser.ReadUserByIdUseCase,
	readAllCartProducts *useCasesCartProduct.ReadAllCartProductsUseCase,
) *CreateOrderUseCase {
	return &CreateOrderUseCase{
		repository:          repository,
		readUserById:        readUserById,
		readAllCartProducts: readAllCartProducts,
	}
}

// UseCase faz o checkout dos itens selecionados (isSelect = true) do carrinho ativo do usuario
func (useCase *CreateOrderUseCase) UseCase(zipCode string, userId string) (*entitiesOrder.Order, error) {

	user, err := useCase.readUserById.UseCase(userId)

	if err != nil {
		return nil, err
	}

	listCartProducts, err := useCase.readAllCartProducts.UseCase(userId)

	if err != nil {
		return nil, err
	}

	order := entitiesOrder.Order{
		Id:      uuid.New(),
		User:    user,
		Status:  entitiesOrder.OrderStatusPreparing,
		ZipCode: zipCode,
	}

	var cartProductIds []string
	indexByProduct := make(map[uuid.UUID]int)

	for _, cartProduct := range listCartProducts {

		if !cartProduct.IsSelect {
			continue
		}

		cartProductIds = append(cartProductIds, cartProduct.Id.String())

		// o carrinho pode ter o mesmo produto em mais de uma linha: agrupa em um unico item
		if index, exists := indexByProduct[cartProduct.Product.Id]; exists {
			order.Items[index].Quantity += cartProduct.Quantity
			continue
		}

		indexByProduct[cartProduct.Product.Id] = len(order.Items)

		order.Items = append(order.Items, entitiesOrderItem.OrderItem{
			Id:        uuid.New(),
			OrderId:   order.Id,
			Product:   cartProduct.Product,
			Quantity:  cartProduct.Quantity,
			UnitPrice: cartProduct.Product.Price,
		})
	}

	if len(order.Items) == 0 {
		return nil, apperrors.ErrUnprocessableEntity
	}

	if err = useCase.repository.CreateOrder(order, cartProductIds); err != nil {
		return nil, err
	}

	createdOrder, err := useCase.repository.ReadOrderById(order.Id.String())

	if err != nil {
		return nil, err
	}

	return createdOrder, nil
}
