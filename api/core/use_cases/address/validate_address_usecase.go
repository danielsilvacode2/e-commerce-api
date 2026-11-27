package use_cases

import (
	"context"
	"e-commerce/core/apperrors"
	entitiesAddress "e-commerce/core/entities/address"
	"e-commerce/core/ports"
	"errors"
	"fmt"
	"log"
)

const DegradedAddressValidationWarning = "Validação externa de CEP indisponível no momento; endereço aceito com validação local."

// ValidateAddressResult encapsula o endereco normalizado e eventual aviso de validacao degradada (fail-open).
type ValidateAddressResult struct {
	Address entitiesAddress.Address
	Warning string
}

type ValidateAddressUseCase struct {
	gateway ports.AddressLookupGateway
}

func NewValidateAddressUseCase(gateway ports.AddressLookupGateway) *ValidateAddressUseCase {
	return &ValidateAddressUseCase{
		gateway: gateway,
	}
}

// UseCase valida integralmente o endereco:
// 1. Regras locais (formato CEP 8 digitos, UF valida, rua, cidade e numero obrigatorios).
// 2. Consulta do CEP no provedor (rejeita 404 ou divergencia de cidade/UF com 422).
// 3. Validacao de pertencimento da cidade ao estado via IBGE (ignorando acentos e caixa).
// 4. Politica fail-open: se o provedor externo estiver indisponivel (5xx/timeout), aceita com validacao local + aviso.
func (uc *ValidateAddressUseCase) UseCase(ctx context.Context, input entitiesAddress.Address) (*ValidateAddressResult, error) {
	normalized, localErrors := input.ValidateLocal()
	if len(localErrors) > 0 {
		baseErr := apperrors.ErrInvalidInput
		// Se o unico erro local for UF nao reconhecida (2+ letras preenchidas, mas inexistente), trata como 422.
		if len(localErrors) == 1 && localErrors[0].Field == "state" && normalized.State != "" {
			baseErr = apperrors.ErrUnprocessableEntity
		}
		return nil, apperrors.NewValidationError(
			baseErr,
			"Verifique os campos de endereço informados.",
			localErrors...,
		)
	}

	if uc.gateway == nil {
		log.Printf("[WARN] AddressLookupGateway nao configurado; aceitando endereco CEP=%s em modo fail-open.", normalized.Cep)
		return &ValidateAddressResult{
			Address: normalized,
			Warning: DegradedAddressValidationWarning,
		}, nil
	}

	var semanticErrors []apperrors.FieldError
	fieldErrorAdded := make(map[string]bool)
	addFieldError := func(field, msg string) {
		if fieldErrorAdded[field] {
			return
		}
		fieldErrorAdded[field] = true
		semanticErrors = append(semanticErrors, apperrors.FieldError{
			Field:   field,
			Message: msg,
		})
	}

	degraded := false

	// 1. Consulta do CEP na BrasilAPI
	cepInfo, cepErr := uc.gateway.LookupByCep(ctx, normalized.Cep)
	if cepErr != nil {
		switch {
		case errors.Is(cepErr, apperrors.ErrCepNotFound):
			addFieldError("cep", "CEP não encontrado.")
		case errors.Is(cepErr, apperrors.ErrInvalidInput):
			addFieldError("cep", "Informe um CEP válido com 8 dígitos (ex.: 00000-000).")
		default:
			// Fail-open em indisponibilidade/timeout da BrasilAPI
			degraded = true
			log.Printf("[WARN] BrasilAPI CEP indisponivel para %s (%v); seguindo em modo fail-open.", normalized.Cep, cepErr)
		}
	}

	// 2. Validacao IBGE: cidade pertence ao estado informado (sem diferenca de acento e caixa)
	cities, ibgeErr := uc.gateway.ListCitiesByState(ctx, normalized.State)
	if ibgeErr != nil {
		switch {
		case errors.Is(ibgeErr, apperrors.ErrInvalidInput):
			addFieldError("state", "Informe uma sigla de estado (UF) brasileira válida.")
		default:
			degraded = true
			log.Printf("[WARN] BrasilAPI IBGE indisponivel para UF %s (%v); seguindo em modo fail-open.", normalized.State, ibgeErr)
		}
	} else {
		cityBelongsToState := false
		for _, city := range cities {
			if entitiesAddress.MatchesCity(city, normalized.City) {
				cityBelongsToState = true
				break
			}
		}
		if !cityBelongsToState {
			addFieldError("city", fmt.Sprintf("A cidade '%s' não pertence ao estado %s.", normalized.City, normalized.State))
		}
	}

	// 3. Validacao de coerencia entre UF/cidade informadas e o CEP retornado pela BrasilAPI
	if cepInfo != nil {
		cepState := entitiesAddress.NormalizeState(cepInfo.State)
		if cepState != "" && cepState != normalized.State {
			addFieldError("state", fmt.Sprintf("O estado informado (%s) diverge do CEP (%s).", normalized.State, cepState))
		}
		if cepInfo.City != "" && !entitiesAddress.MatchesCity(normalized.City, cepInfo.City) {
			addFieldError("city", fmt.Sprintf("A cidade informada (%s) diverge do CEP (%s).", normalized.City, cepInfo.City))
		}
	}

	if len(semanticErrors) > 0 {
		return nil, apperrors.NewValidationError(
			apperrors.ErrUnprocessableEntity,
			"Os dados de endereço informados não são válidos.",
			semanticErrors...,
		)
	}

	result := &ValidateAddressResult{
		Address: normalized,
	}
	if degraded {
		result.Warning = DegradedAddressValidationWarning
	}
	return result, nil
}
