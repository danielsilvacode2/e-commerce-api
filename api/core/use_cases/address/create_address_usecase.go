package use_cases

import (
	"context"
	"e-commerce/core/apperrors"
	entitiesAddress "e-commerce/core/entities/address"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CreateAddressResult struct {
	Address *entitiesAddress.Address
	Warning string
}

type CreateAddressUseCase struct {
	repository      entitiesAddress.AddressRepository
	validateAddress *ValidateAddressUseCase
}

func NewCreateAddressUseCase(
	repository entitiesAddress.AddressRepository,
	validateAddress *ValidateAddressUseCase,
) *CreateAddressUseCase {
	return &CreateAddressUseCase{
		repository:      repository,
		validateAddress: validateAddress,
	}
}

// UseCase valida o endereco via BrasilAPI/IBGE (com fail-open) e persiste uma nova instancia
// vinculada ao usuario autenticado, evitando duplicatas exatas.
func (uc *CreateAddressUseCase) UseCase(
	ctx context.Context,
	userId string,
	input entitiesAddress.Address,
) (*CreateAddressResult, error) {
	userUUID, err := uuid.Parse(strings.TrimSpace(userId))
	if err != nil {
		return nil, apperrors.ErrUnauthorized
	}

	if uc.validateAddress == nil {
		return nil, apperrors.ErrAddressLookupUnavailable
	}

	validated, err := uc.validateAddress.UseCase(ctx, input)
	if err != nil {
		return nil, err
	}

	// Evita duplicata exata no catalogo de enderecos do usuario
	existing, matchErr := uc.repository.FindMatchingUserAddress(userUUID.String(), validated.Address)
	if matchErr == nil && existing != nil {
		return &CreateAddressResult{
			Address: existing,
			Warning: validated.Warning,
		}, nil
	}
	if matchErr != nil && !errors.Is(matchErr, apperrors.ErrNotFound) {
		return nil, matchErr
	}

	now := time.Now()
	addr := validated.Address
	addr.Id = uuid.New()
	addr.UserId = userUUID
	addr.CreateAt = now
	addr.UpdateAt = now

	if err = uc.repository.CreateAddress(addr); err != nil {
		return nil, err
	}

	return &CreateAddressResult{
		Address: &addr,
		Warning: validated.Warning,
	}, nil
}
