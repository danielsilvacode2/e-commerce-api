package use_cases

import (
	"e-commerce/core/apperrors"
	entitiesOrder "e-commerce/core/entities/order"
	entitiesOrderItem "e-commerce/core/entities/order_item"
	entitiesPayment "e-commerce/core/entities/payment"
	useCasesCartProduct "e-commerce/core/use_cases/cart_product"
	useCasesUser "e-commerce/core/use_cases/user"
	"time"

	"github.com/google/uuid"
)

type CreateOrderUseCase struct {
	repository          entitiesOrder.OrderRepository
	paymentRepository   entitiesPayment.PaymentRepository
	readUserById        *useCasesUser.ReadUserByIdUseCase
	readAllCartProducts *useCasesCartProduct.ReadAllCartProductsUseCase
}

func NewCreateOrderUseCase(
	repository entitiesOrder.OrderRepository,
	paymentRepository entitiesPayment.PaymentRepository,
	readUserById *useCasesUser.ReadUserByIdUseCase,
	readAllCartProducts *useCasesCartProduct.ReadAllCartProductsUseCase,
) *CreateOrderUseCase {
	return &CreateOrderUseCase{
		repository:          repository,
		paymentRepository:   paymentRepository,
		readUserById:        readUserById,
		readAllCartProducts: readAllCartProducts,
	}
}

// UseCase faz o checkout dos itens selecionados (isSelect = true) do carrinho ativo do usuario.
// O pedido inicia em PENDING_PAYMENT e um registro inicial de pagamento PENDING e criado.
func (useCase *CreateOrderUseCase) UseCase(zipCode string, userId string) (*entitiesOrder.Order, error) {

	listCartProducts, err := useCase.readAllCartProducts.UseCase(userId)

	if err != nil {
		return nil, err
	}

	if len(listCartProducts) == 0 {
		return nil, apperrors.ErrUnprocessableEntity
	}

	// Reaproveita os dados do usuario ja trazidos no JOIN do carrinho, evitando uma consulta extra ao banco.
	var user = listCartProducts[0].Cart.User
	if user == nil || user.Email == "" {
		user, err = useCase.readUserById.UseCase(userId)
		if err != nil {
			return nil, err
		}
	}

	now := time.Now()
	order := entitiesOrder.Order{
		Id:       uuid.New(),
		User:     user,
		Status:   entitiesOrder.OrderStatusPendingPayment,
		ZipCode:  zipCode,
		CreateAt: now,
		UpdateAt: now,
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

	if useCase.paymentRepository != nil {
		expiresAt := now.Add(15 * time.Minute)
		initialPayment := entitiesPayment.Payment{
			Id:             uuid.New(),
			OrderId:        order.Id,
			Provider:       entitiesPayment.PaymentProviderDefault,
			Method:         entitiesPayment.PaymentMethodPix,
			IdempotencyKey: "init:" + order.Id.String(),
			Amount:         order.Total(),
			Installments:   1,
			Status:         entitiesPayment.PaymentStatusPending,
			ExpiresAt:      &expiresAt,
		}
		_ = useCase.paymentRepository.CreatePayment(initialPayment)
	}

	return &order, nil
}
