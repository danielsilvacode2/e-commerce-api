package use_cases

import (
	"context"
	"e-commerce/core/apperrors"
	entitiesAddress "e-commerce/core/entities/address"
	entitiesOrder "e-commerce/core/entities/order"
	entitiesOrderItem "e-commerce/core/entities/order_item"
	entitiesPayment "e-commerce/core/entities/payment"
	useCasesAddress "e-commerce/core/use_cases/address"
	useCasesCartProduct "e-commerce/core/use_cases/cart_product"
	useCasesUser "e-commerce/core/use_cases/user"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// OrderAddressSelection permite criar/atualizar pedido selecionando um endereco salvo (AddressId)
// ou cadastrando uma nova instancia de endereco (Address).
type OrderAddressSelection struct {
	AddressId string
	Address   *entitiesAddress.Address
}

type CreateOrderResult struct {
	Order   *entitiesOrder.Order
	Warning string
}

type CreateOrderUseCase struct {
	repository          entitiesOrder.OrderRepository
	addressRepository   entitiesAddress.AddressRepository
	paymentRepository   entitiesPayment.PaymentRepository
	readUserById        *useCasesUser.ReadUserByIdUseCase
	readAllCartProducts *useCasesCartProduct.ReadAllCartProductsUseCase
	validateAddress     *useCasesAddress.ValidateAddressUseCase
}

func NewCreateOrderUseCase(
	repository entitiesOrder.OrderRepository,
	addressRepository entitiesAddress.AddressRepository,
	paymentRepository entitiesPayment.PaymentRepository,
	readUserById *useCasesUser.ReadUserByIdUseCase,
	readAllCartProducts *useCasesCartProduct.ReadAllCartProductsUseCase,
	validateAddress *useCasesAddress.ValidateAddressUseCase,
) *CreateOrderUseCase {
	return &CreateOrderUseCase{
		repository:          repository,
		addressRepository:   addressRepository,
		paymentRepository:   paymentRepository,
		readUserById:        readUserById,
		readAllCartProducts: readAllCartProducts,
		validateAddress:     validateAddress,
	}
}

// UseCase valida ou reutiliza o endereco selecionado e faz o checkout dos itens selecionados
// (isSelect = true) do carrinho ativo do usuario.
func (useCase *CreateOrderUseCase) UseCase(
	ctx context.Context,
	selection OrderAddressSelection,
	userId string,
) (*CreateOrderResult, error) {
	selectedAddressID := strings.TrimSpace(selection.AddressId)
	if selectedAddressID != "" && selection.Address != nil {
		return nil, apperrors.NewValidationError(
			apperrors.ErrInvalidInput,
			"Informe apenas um endereço salvo (address_id) ou um novo endereço (address), não ambos.",
			apperrors.FieldError{
				Field:   "address_id",
				Message: "Selecione um endereço salvo ou cadastre um novo endereço.",
			},
		)
	}
	if selectedAddressID == "" && selection.Address == nil {
		return nil, apperrors.NewValidationError(
			apperrors.ErrInvalidInput,
			"Informe um endereço salvo ou preencha um novo endereço para entrega.",
			apperrors.FieldError{
				Field:   "address_id",
				Message: "Selecione um endereço salvo ou informe um novo endereço.",
			},
		)
	}

	var (
		resolvedAddress    *entitiesAddress.Address
		useExistingAddress bool
		warning            string
	)

	if selectedAddressID != "" {
		if _, err := uuid.Parse(selectedAddressID); err != nil {
			return nil, apperrors.NewValidationError(
				apperrors.ErrInvalidInput,
				"Identificador de endereço salvo inválido.",
				apperrors.FieldError{
					Field:   "address_id",
					Message: "Endereço selecionado inválido.",
				},
			)
		}
		if useCase.addressRepository == nil {
			return nil, apperrors.ErrDatabaseUnavailable
		}

		saved, err := useCase.addressRepository.ReadAddressById(selectedAddressID)
		if err != nil {
			return nil, err
		}
		if !saved.BelongsTo(userId) {
			return nil, apperrors.ErrForbidden
		}
		if !saved.IsComplete() {
			return nil, apperrors.NewValidationError(
				apperrors.ErrUnprocessableEntity,
				"O endereço selecionado está incompleto. Cadastre um novo endereço completo.",
				apperrors.FieldError{
					Field:   "address_id",
					Message: "Este endereço não possui todos os dados obrigatórios. Cadastre um novo endereço.",
				},
			)
		}

		resolvedAddress = saved
		useExistingAddress = true
	} else {
		if useCase.validateAddress == nil {
			return nil, apperrors.ErrAddressLookupUnavailable
		}

		validated, err := useCase.validateAddress.UseCase(ctx, *selection.Address)
		if err != nil {
			return nil, err
		}
		warning = validated.Warning

		// Se o usuario ja possui uma instancia identica cadastrada, reutiliza-a para evitar duplicatas exatas.
		if useCase.addressRepository != nil {
			existing, matchErr := useCase.addressRepository.FindMatchingUserAddress(userId, validated.Address)
			if matchErr == nil && existing != nil {
				resolvedAddress = existing
				useExistingAddress = true
			} else if matchErr != nil && !errors.Is(matchErr, apperrors.ErrNotFound) {
				return nil, matchErr
			}
		}

		if resolvedAddress == nil {
			addr := validated.Address
			resolvedAddress = &addr
			useExistingAddress = false
		}
	}

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
	if !useExistingAddress {
		resolvedAddress.Id = uuid.New()
		resolvedAddress.UserId = user.ID
		resolvedAddress.CreateAt = now
		resolvedAddress.UpdateAt = now
	}

	order := entitiesOrder.Order{
		Id:       uuid.New(),
		User:     user,
		Address:  resolvedAddress,
		Status:   entitiesOrder.OrderStatusPendingPayment,
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

	if err = useCase.repository.CreateOrder(order, cartProductIds, useExistingAddress); err != nil {
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

	return &CreateOrderResult{
		Order:   &order,
		Warning: warning,
	}, nil
}
