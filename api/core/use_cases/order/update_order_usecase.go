package use_cases

import (
	"context"
	"e-commerce/core/apperrors"
	entitiesAddress "e-commerce/core/entities/address"
	entities "e-commerce/core/entities/order"
	useCasesAddress "e-commerce/core/use_cases/address"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type UpdateOrderUseCase struct {
	repository        entities.OrderRepository
	addressRepository entitiesAddress.AddressRepository
	readOrderById     *ReadOrderByIdUseCase
	validateAddress   *useCasesAddress.ValidateAddressUseCase
}

func NewUpdateOrderUseCase(
	repository entities.OrderRepository,
	addressRepository entitiesAddress.AddressRepository,
	readOrderById *ReadOrderByIdUseCase,
	validateAddress *useCasesAddress.ValidateAddressUseCase,
) *UpdateOrderUseCase {
	return &UpdateOrderUseCase{
		repository:        repository,
		addressRepository: addressRepository,
		readOrderById:     readOrderById,
		validateAddress:   validateAddress,
	}
}

// UseCase altera o endereco de entrega (por endereco salvo ou nova instancia) sem mutar pedidos historicos.
func (useCase *UpdateOrderUseCase) UseCase(
	ctx context.Context,
	selection OrderAddressSelection,
	orderId string,
	userId string,
	role string,
) (string, error) {
	selectedAddressID := strings.TrimSpace(selection.AddressId)
	if (selectedAddressID != "" && selection.Address != nil) || (selectedAddressID == "" && selection.Address == nil) {
		return "", apperrors.NewValidationError(
			apperrors.ErrInvalidInput,
			"Informe um endereço salvo (address_id) ou um novo endereço (address).",
			apperrors.FieldError{
				Field:   "address_id",
				Message: "Selecione um endereço salvo ou informe um novo endereço.",
			},
		)
	}

	order, err := useCase.readOrderById.UseCase(orderId, userId, role)
	if err != nil {
		return "", err
	}

	if !order.IsEditable() {
		return "", apperrors.ErrUnprocessableEntity
	}

	ownerID := userId
	if order.User != nil && order.User.ID != uuid.Nil {
		ownerID = order.User.ID.String()
	}

	if selectedAddressID != "" {
		if _, err := uuid.Parse(selectedAddressID); err != nil {
			return "", apperrors.NewValidationError(
				apperrors.ErrInvalidInput,
				"Identificador de endereço salvo inválido.",
				apperrors.FieldError{
					Field:   "address_id",
					Message: "Endereço selecionado inválido.",
				},
			)
		}
		if useCase.addressRepository == nil {
			return "", apperrors.ErrDatabaseUnavailable
		}

		saved, err := useCase.addressRepository.ReadAddressById(selectedAddressID)
		if err != nil {
			return "", err
		}
		if !saved.BelongsTo(ownerID) {
			return "", apperrors.ErrForbidden
		}
		if !saved.IsComplete() {
			return "", apperrors.NewValidationError(
				apperrors.ErrUnprocessableEntity,
				"O endereço selecionado está incompleto.",
				apperrors.FieldError{
					Field:   "address_id",
					Message: "Este endereço está incompleto. Cadastre um novo endereço.",
				},
			)
		}

		if err = useCase.repository.UpdateOrderAddress(*saved, orderId, true); err != nil {
			return "", err
		}
		return "", nil
	}

	if useCase.validateAddress == nil {
		return "", apperrors.ErrAddressLookupUnavailable
	}

	validated, err := useCase.validateAddress.UseCase(ctx, *selection.Address)
	if err != nil {
		return "", err
	}

	if useCase.addressRepository != nil {
		existing, matchErr := useCase.addressRepository.FindMatchingUserAddress(ownerID, validated.Address)
		if matchErr == nil && existing != nil {
			if err = useCase.repository.UpdateOrderAddress(*existing, orderId, true); err != nil {
				return "", err
			}
			return validated.Warning, nil
		} else if matchErr != nil && !errors.Is(matchErr, apperrors.ErrNotFound) {
			return "", matchErr
		}
	}

	ownerUUID, err := uuid.Parse(ownerID)
	if err != nil {
		return "", apperrors.ErrInvalidInput
	}

	now := time.Now()
	newAddress := validated.Address
	newAddress.Id = uuid.New()
	newAddress.UserId = ownerUUID
	newAddress.CreateAt = now
	newAddress.UpdateAt = now

	if err = useCase.repository.UpdateOrderAddress(newAddress, orderId, false); err != nil {
		return "", err
	}

	return validated.Warning, nil
}
