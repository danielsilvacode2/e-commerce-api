package use_cases

import (
	"e-commerce/core/apperrors"
	entitiesAddress "e-commerce/core/entities/address"
	"strings"
)

type ReadMyAddressesUseCase struct {
	repository entitiesAddress.AddressRepository
}

func NewReadMyAddressesUseCase(repository entitiesAddress.AddressRepository) *ReadMyAddressesUseCase {
	return &ReadMyAddressesUseCase{
		repository: repository,
	}
}

// UseCase retorna todos os enderecos completos salvos pelo usuario autenticado.
func (uc *ReadMyAddressesUseCase) UseCase(userId string) ([]entitiesAddress.Address, error) {
	if strings.TrimSpace(userId) == "" {
		return nil, apperrors.ErrUnauthorized
	}
	return uc.repository.ReadAddressesByUserId(userId)
}
