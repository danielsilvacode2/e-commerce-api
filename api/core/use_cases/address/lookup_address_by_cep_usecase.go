package use_cases

import (
	"context"
	"e-commerce/core/apperrors"
	entitiesAddress "e-commerce/core/entities/address"
	"e-commerce/core/ports"
)

type LookupAddressByCepUseCase struct {
	gateway ports.AddressLookupGateway
}

func NewLookupAddressByCepUseCase(gateway ports.AddressLookupGateway) *LookupAddressByCepUseCase {
	return &LookupAddressByCepUseCase{
		gateway: gateway,
	}
}

// UseCase valida o formato do CEP (8 digitos, aceitando mascara 00000-000) e consulta o provedor externo.
func (uc *LookupAddressByCepUseCase) UseCase(ctx context.Context, rawCep string) (*ports.CepLookupResult, error) {
	cep, err := entitiesAddress.NormalizeCep(rawCep)
	if err != nil {
		return nil, err
	}

	if uc.gateway == nil {
		return nil, apperrors.ErrAddressLookupUnavailable
	}

	return uc.gateway.LookupByCep(ctx, cep)
}
