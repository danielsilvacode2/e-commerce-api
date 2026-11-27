package use_cases_test

import (
	"context"
	"e-commerce/core/apperrors"
	entitiesAddress "e-commerce/core/entities/address"
	"e-commerce/core/ports"
	useCasesAddress "e-commerce/core/use_cases/address"
	"errors"
	"testing"
)

type fakeAddressGateway struct {
	cepResults map[string]*ports.CepLookupResult
	cepErr     error
	citiesByUF map[string][]string
	citiesErr  error
}

func (f *fakeAddressGateway) LookupByCep(ctx context.Context, cep string) (*ports.CepLookupResult, error) {
	if f.cepErr != nil {
		return nil, f.cepErr
	}
	if res, ok := f.cepResults[cep]; ok {
		return res, nil
	}
	return nil, apperrors.ErrCepNotFound
}

func (f *fakeAddressGateway) ListCitiesByState(ctx context.Context, uf string) ([]string, error) {
	if f.citiesErr != nil {
		return nil, f.citiesErr
	}
	if cities, ok := f.citiesByUF[uf]; ok {
		return cities, nil
	}
	return nil, apperrors.ErrInvalidInput
}

func newDefaultGateway() *fakeAddressGateway {
	return &fakeAddressGateway{
		cepResults: map[string]*ports.CepLookupResult{
			"01001000": {
				Cep:          "01001000",
				State:        "SP",
				City:         "São Paulo",
				Neighborhood: "Sé",
				Street:       "Praça da Sé",
			},
			"18195000": {
				Cep:          "18195000",
				State:        "SP",
				City:         "Capela do Alto",
				Neighborhood: "",
				Street:       "", // CEP unico de municipio sem rua
			},
		},
		citiesByUF: map[string][]string{
			"SP": {"SÃO PAULO", "CAMPINAS", "CAPELA DO ALTO"},
			"RJ": {"RIO DE JANEIRO", "NITERÓI"},
		},
	}
}

func TestLookupAddressByCepUseCase(t *testing.T) {
	uc := useCasesAddress.NewLookupAddressByCepUseCase(newDefaultGateway())

	t.Run("aceita CEP com mascara e retorna dados", func(t *testing.T) {
		res, err := uc.UseCase(context.Background(), "01001-000")
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if res.Street != "Praça da Sé" || res.City != "São Paulo" || res.State != "SP" {
			t.Fatalf("resultado inesperado: %+v", res)
		}
	})

	t.Run("rejeita CEP com formato invalido", func(t *testing.T) {
		_, err := uc.UseCase(context.Background(), "1234")
		if !errors.Is(err, apperrors.ErrInvalidInput) {
			t.Fatalf("esperava ErrInvalidInput, obteve %v", err)
		}
	})

	t.Run("retorna ErrCepNotFound quando CEP nao existe", func(t *testing.T) {
		_, err := uc.UseCase(context.Background(), "99999-999")
		if !errors.Is(err, apperrors.ErrCepNotFound) {
			t.Fatalf("esperava ErrCepNotFound, obteve %v", err)
		}
	})
}

func TestValidateAddressUseCase(t *testing.T) {
	t.Run("valida endereco completo com sucesso ignorando acento e caixa da cidade", func(t *testing.T) {
		uc := useCasesAddress.NewValidateAddressUseCase(newDefaultGateway())
		res, err := uc.UseCase(context.Background(), entitiesAddress.Address{
			Cep:          "01001-000",
			Street:       "Praça da Sé",
			Number:       "123A",
			Complement:   "Apto 4",
			Neighborhood: "Sé",
			City:         "sao paulo",
			State:        "sp",
		})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if res.Warning != "" {
			t.Fatalf("nao esperava warning, obteve %q", res.Warning)
		}
		if res.Address.Cep != "01001000" || res.Address.State != "SP" || res.Address.Number != "123A" {
			t.Fatalf("endereco normalizado inesperado: %+v", res.Address)
		}
	})

	t.Run("permite preencher rua manualmente em CEP unico de municipio", func(t *testing.T) {
		uc := useCasesAddress.NewValidateAddressUseCase(newDefaultGateway())
		res, err := uc.UseCase(context.Background(), entitiesAddress.Address{
			Cep:    "18195-000",
			Street: "Estrada Municipal 10",
			Number: "S/N",
			City:   "Capela do Alto",
			State:  "SP",
		})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if res.Address.Street != "Estrada Municipal 10" || res.Address.Number != "S/N" {
			t.Fatalf("endereco inesperado: %+v", res.Address)
		}
	})

	t.Run("rejeita CEP inexistente com 422 no campo cep", func(t *testing.T) {
		uc := useCasesAddress.NewValidateAddressUseCase(newDefaultGateway())
		_, err := uc.UseCase(context.Background(), entitiesAddress.Address{
			Cep:    "99999-999",
			Street: "Rua X",
			Number: "10",
			City:   "São Paulo",
			State:  "SP",
		})
		if !errors.Is(err, apperrors.ErrUnprocessableEntity) {
			t.Fatalf("esperava ErrUnprocessableEntity, obteve %v", err)
		}
		var valErr *apperrors.ValidationError
		if !errors.As(err, &valErr) || len(valErr.Fields) == 0 || valErr.Fields[0].Field != "cep" {
			t.Fatalf("esperava erro no campo cep, obteve %+v", err)
		}
	})

	t.Run("rejeita UF invalida com 422 no campo state", func(t *testing.T) {
		uc := useCasesAddress.NewValidateAddressUseCase(newDefaultGateway())
		_, err := uc.UseCase(context.Background(), entitiesAddress.Address{
			Cep:    "01001000",
			Street: "Praça da Sé",
			Number: "10",
			City:   "São Paulo",
			State:  "XX",
		})
		if !errors.Is(err, apperrors.ErrUnprocessableEntity) {
			t.Fatalf("esperava ErrUnprocessableEntity, obteve %v", err)
		}
		var valErr *apperrors.ValidationError
		if !errors.As(err, &valErr) || valErr.Fields[0].Field != "state" {
			t.Fatalf("esperava erro no campo state, obteve %+v", err)
		}
	})

	t.Run("rejeita cidade que nao pertence a UF (IBGE) com 422", func(t *testing.T) {
		uc := useCasesAddress.NewValidateAddressUseCase(newDefaultGateway())
		_, err := uc.UseCase(context.Background(), entitiesAddress.Address{
			Cep:    "01001000",
			Street: "Praça da Sé",
			Number: "10",
			City:   "Rio de Janeiro",
			State:  "SP",
		})
		if !errors.Is(err, apperrors.ErrUnprocessableEntity) {
			t.Fatalf("esperava ErrUnprocessableEntity, obteve %v", err)
		}
		var valErr *apperrors.ValidationError
		if !errors.As(err, &valErr) || len(valErr.Fields) != 1 || valErr.Fields[0].Field != "city" {
			t.Fatalf("esperava erro no campo city, obteve %+v", valErr)
		}
	})

	t.Run("rejeita cidade valida na UF mas divergente do CEP com 422", func(t *testing.T) {
		uc := useCasesAddress.NewValidateAddressUseCase(newDefaultGateway())
		_, err := uc.UseCase(context.Background(), entitiesAddress.Address{
			Cep:    "01001000",
			Street: "Praça da Sé",
			Number: "10",
			City:   "Campinas",
			State:  "SP",
		})
		if !errors.Is(err, apperrors.ErrUnprocessableEntity) {
			t.Fatalf("esperava ErrUnprocessableEntity, obteve %v", err)
		}
		var valErr *apperrors.ValidationError
		if !errors.As(err, &valErr) || valErr.Fields[0].Field != "city" {
			t.Fatalf("esperava erro de divergencia no campo city, obteve %+v", valErr)
		}
	})

	t.Run("rejeita UF divergente do CEP com 422", func(t *testing.T) {
		uc := useCasesAddress.NewValidateAddressUseCase(newDefaultGateway())
		_, err := uc.UseCase(context.Background(), entitiesAddress.Address{
			Cep:    "01001000",
			Street: "Praça da Sé",
			Number: "10",
			City:   "Rio de Janeiro",
			State:  "RJ",
		})
		if !errors.Is(err, apperrors.ErrUnprocessableEntity) {
			t.Fatalf("esperava ErrUnprocessableEntity, obteve %v", err)
		}
		var valErr *apperrors.ValidationError
		if !errors.As(err, &valErr) || len(valErr.Fields) < 2 {
			t.Fatalf("esperava erros em state e city divergentes do CEP, obteve %+v", valErr)
		}
	})

	t.Run("rejeita rua ou numero vazios", func(t *testing.T) {
		uc := useCasesAddress.NewValidateAddressUseCase(newDefaultGateway())
		_, err := uc.UseCase(context.Background(), entitiesAddress.Address{
			Cep:    "01001000",
			Street: " ",
			Number: "",
			City:   "São Paulo",
			State:  "SP",
		})
		if !errors.Is(err, apperrors.ErrInvalidInput) {
			t.Fatalf("esperava ErrInvalidInput, obteve %v", err)
		}
	})

	t.Run("fail-open com aviso quando BrasilAPI esta indisponivel", func(t *testing.T) {
		gw := &fakeAddressGateway{
			cepErr:    apperrors.ErrAddressLookupUnavailable,
			citiesErr: apperrors.ErrAddressLookupUnavailable,
		}
		uc := useCasesAddress.NewValidateAddressUseCase(gw)
		res, err := uc.UseCase(context.Background(), entitiesAddress.Address{
			Cep:    "01001-000",
			Street: "Praça da Sé",
			Number: "100",
			City:   "São Paulo",
			State:  "SP",
		})
		if err != nil {
			t.Fatalf("em fail-open com dados locais validos nao deveria falhar: %v", err)
		}
		if res.Warning == "" {
			t.Fatal("esperava aviso de validacao degradada")
		}
		if res.Address.Cep != "01001000" {
			t.Fatalf("esperava CEP normalizado 01001000, obteve %q", res.Address.Cep)
		}
	})

	t.Run("mesmo com BrasilAPI indisponivel rejeita UF invalida localmente", func(t *testing.T) {
		gw := &fakeAddressGateway{
			cepErr:    apperrors.ErrAddressLookupUnavailable,
			citiesErr: apperrors.ErrAddressLookupUnavailable,
		}
		uc := useCasesAddress.NewValidateAddressUseCase(gw)
		_, err := uc.UseCase(context.Background(), entitiesAddress.Address{
			Cep:    "01001-000",
			Street: "Praça da Sé",
			Number: "100",
			City:   "São Paulo",
			State:  "ZZ",
		})
		if !errors.Is(err, apperrors.ErrUnprocessableEntity) {
			t.Fatalf("esperava ErrUnprocessableEntity para UF invalida, obteve %v", err)
		}
	})
}
