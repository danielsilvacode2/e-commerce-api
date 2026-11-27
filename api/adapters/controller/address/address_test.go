package controller

import (
	"bytes"
	"context"
	"e-commerce/adapters/security/auth"
	"e-commerce/core/apperrors"
	entitiesAddress "e-commerce/core/entities/address"
	"e-commerce/core/ports"
	useCasesAddress "e-commerce/core/use_cases/address"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type stubAddressLookupGateway struct {
	result *ports.CepLookupResult
	err    error
	cities []string
}

func (s *stubAddressLookupGateway) LookupByCep(ctx context.Context, cep string) (*ports.CepLookupResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.result, nil
}

func (s *stubAddressLookupGateway) ListCitiesByState(ctx context.Context, uf string) ([]string, error) {
	if len(s.cities) > 0 {
		return s.cities, nil
	}
	return nil, apperrors.ErrAddressLookupUnavailable
}

type stubAddressRepoCtrl struct {
	list    []entitiesAddress.Address
	created []entitiesAddress.Address
	err     error
}

func (s *stubAddressRepoCtrl) CreateAddress(addr entitiesAddress.Address) error {
	if s.err != nil {
		return s.err
	}
	s.created = append(s.created, addr)
	return nil
}
func (s *stubAddressRepoCtrl) ReadAddressById(string) (*entitiesAddress.Address, error) {
	return nil, apperrors.ErrNotFound
}
func (s *stubAddressRepoCtrl) ReadAddressesByUserId(string) ([]entitiesAddress.Address, error) {
	return s.list, s.err
}
func (s *stubAddressRepoCtrl) FindMatchingUserAddress(string, entitiesAddress.Address) (*entitiesAddress.Address, error) {
	return nil, apperrors.ErrNotFound
}

func TestAddressControllerLookupByCep(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("retorna 200 com dados do endereco para CEP valido com mascara", func(t *testing.T) {
		gw := &stubAddressLookupGateway{
			result: &ports.CepLookupResult{
				Cep:          "01001000",
				Street:       "Praça da Sé",
				Neighborhood: "Sé",
				City:         "São Paulo",
				State:        "SP",
			},
		}
		ctrl := NewAddressController(useCasesAddress.NewLookupAddressByCepUseCase(gw), nil, nil)

		r := gin.New()
		r.GET("/api/v1/addresses/lookup/:cep", ctrl.LookupByCep)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/addresses/lookup/01001-000", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, esperado 200 (body=%s)", w.Code, w.Body.String())
		}

		var payload struct {
			Body struct {
				Cep          string `json:"cep"`
				Street       string `json:"street"`
				Neighborhood string `json:"neighborhood"`
				City         string `json:"city"`
				State        string `json:"state"`
			} `json:"Body:"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
			t.Fatalf("erro ao decodificar JSON: %v", err)
		}
		if payload.Body.Cep != "01001000" || payload.Body.Street != "Praça da Sé" || payload.Body.City != "São Paulo" || payload.Body.State != "SP" {
			t.Fatalf("payload inesperado: %+v", payload.Body)
		}
	})

	t.Run("retorna 400 com erro de campo para CEP invalido", func(t *testing.T) {
		ctrl := NewAddressController(useCasesAddress.NewLookupAddressByCepUseCase(&stubAddressLookupGateway{}), nil, nil)
		r := gin.New()
		r.GET("/api/v1/addresses/lookup/:cep", ctrl.LookupByCep)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/addresses/lookup/1234", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, esperado 400 (body=%s)", w.Code, w.Body.String())
		}
	})

	t.Run("retorna 404 quando CEP nao e encontrado", func(t *testing.T) {
		ctrl := NewAddressController(useCasesAddress.NewLookupAddressByCepUseCase(&stubAddressLookupGateway{
			err: apperrors.ErrCepNotFound,
		}), nil, nil)
		r := gin.New()
		r.GET("/api/v1/addresses/lookup/:cep", ctrl.LookupByCep)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/addresses/lookup/99999999", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, esperado 404 (body=%s)", w.Code, w.Body.String())
		}
	})

	t.Run("retorna 503 quando BrasilAPI esta indisponivel", func(t *testing.T) {
		ctrl := NewAddressController(useCasesAddress.NewLookupAddressByCepUseCase(&stubAddressLookupGateway{
			err: apperrors.ErrAddressLookupUnavailable,
		}), nil, nil)
		r := gin.New()
		r.GET("/api/v1/addresses/lookup/:cep", ctrl.LookupByCep)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/addresses/lookup/01001000", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, esperado 503 (body=%s)", w.Code, w.Body.String())
		}
	})
}

func TestAddressControllerReadMyAddresses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	addr1ID := uuid.New()
	addr2ID := uuid.New()

	withClaims := func(c *gin.Context) {
		c.Set("claims", &auth.Claims{Id: userID, Role: "user"})
		c.Next()
	}

	repo := &stubAddressRepoCtrl{
		list: []entitiesAddress.Address{
			{Id: addr1ID, UserId: userID, Cep: "01001000", Street: "Praça da Sé", Number: "100", City: "São Paulo", State: "SP"},
			{Id: addr2ID, UserId: userID, Cep: "20040020", Street: "Rua da Assembleia", Number: "10", City: "Rio de Janeiro", State: "RJ"},
		},
	}

	ctrl := NewAddressController(nil, useCasesAddress.NewReadMyAddressesUseCase(repo), nil)
	r := gin.New()
	r.GET("/api/v1/addresses/me", withClaims, ctrl.ReadMyAddresses)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/addresses/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, esperado 200 (body=%s)", w.Code, w.Body.String())
	}

	var payload struct {
		Body []struct {
			Id     string `json:"id"`
			Street string `json:"street"`
		} `json:"Body:"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("erro ao decodificar JSON: %v", err)
	}
	if len(payload.Body) != 2 || payload.Body[0].Id != addr1ID.String() {
		t.Fatalf("lista inesperada: %+v", payload.Body)
	}
}

func TestAddressControllerCreateAddress(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	withClaims := func(c *gin.Context) {
		c.Set("claims", &auth.Claims{Id: userID, Role: "user"})
		c.Next()
	}

	gw := &stubAddressLookupGateway{
		result: &ports.CepLookupResult{
			Cep:          "01001000",
			Street:       "Praça da Sé",
			Neighborhood: "Sé",
			City:         "São Paulo",
			State:        "SP",
		},
		cities: []string{"SÃO PAULO"},
	}
	repo := &stubAddressRepoCtrl{}
	validateUC := useCasesAddress.NewValidateAddressUseCase(gw)
	createUC := useCasesAddress.NewCreateAddressUseCase(repo, validateUC)
	ctrl := NewAddressController(nil, nil, createUC)

	r := gin.New()
	r.POST("/api/v1/addresses", withClaims, ctrl.CreateAddress)

	body := `{"cep":"01001-000","street":"Praça da Sé","number":"200","complement":"Sala 1","neighborhood":"Sé","city":"São Paulo","state":"SP"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/addresses", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, esperado 201 (body=%s)", w.Code, w.Body.String())
	}
	if len(repo.created) != 1 || repo.created[0].UserId != userID || repo.created[0].Cep != "01001000" {
		t.Fatalf("endereco criado inesperado: %+v", repo.created)
	}
}
