package controller

import (
	"bytes"
	"context"
	"e-commerce/adapters/security/auth"
	"e-commerce/core/apperrors"
	entitiesAddress "e-commerce/core/entities/address"
	entitiesCart "e-commerce/core/entities/cart"
	entitiesCartProduct "e-commerce/core/entities/cart_product"
	entitiesOrder "e-commerce/core/entities/order"
	entitiesProduct "e-commerce/core/entities/product"
	entitiesUser "e-commerce/core/entities/user"
	"e-commerce/core/ports"
	useCasesAddress "e-commerce/core/use_cases/address"
	useCasesCartProduct "e-commerce/core/use_cases/cart_product"
	useCasesOrder "e-commerce/core/use_cases/order"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type stubOrderRepoCtrl struct {
	orders           map[string]*entitiesOrder.Order
	lastUsedExisting bool
}

func (s *stubOrderRepoCtrl) CreateOrder(order entitiesOrder.Order, cartProductIds []string, useExistingAddress bool) error {
	s.lastUsedExisting = useExistingAddress
	copyOrder := order
	s.orders[order.Id.String()] = &copyOrder
	return nil
}
func (s *stubOrderRepoCtrl) ReadOrderById(ID string) (*entitiesOrder.Order, error) {
	if o, ok := s.orders[ID]; ok {
		return o, nil
	}
	return nil, apperrors.ErrNotFound
}
func (s *stubOrderRepoCtrl) ReadAllOrders(filter entitiesOrder.OrderFilter) (*entitiesOrder.PaginatedOrders, error) {
	return &entitiesOrder.PaginatedOrders{}, nil
}
func (s *stubOrderRepoCtrl) UpdateOrderAddress(address entitiesAddress.Address, ID string, useExistingAddress bool) error {
	o, ok := s.orders[ID]
	if !ok {
		return apperrors.ErrNotFound
	}
	s.lastUsedExisting = useExistingAddress
	copyAddr := address
	o.Address = &copyAddr
	return nil
}
func (s *stubOrderRepoCtrl) UpdateOrderStatus(string, string, string, bool) error { return nil }
func (s *stubOrderRepoCtrl) UpdateOrderStatusWithDetails(string, string, string, bool, *string, *string, *string) error {
	return nil
}
func (s *stubOrderRepoCtrl) DeleteOrder(string, bool) error { return nil }

type stubAddressRepoForOrderCtrl struct {
	addresses map[string]*entitiesAddress.Address
}

func (s *stubAddressRepoForOrderCtrl) CreateAddress(address entitiesAddress.Address) error {
	copyAddr := address
	s.addresses[address.Id.String()] = &copyAddr
	return nil
}
func (s *stubAddressRepoForOrderCtrl) ReadAddressById(ID string) (*entitiesAddress.Address, error) {
	if a, ok := s.addresses[ID]; ok {
		copyAddr := *a
		return &copyAddr, nil
	}
	return nil, apperrors.ErrNotFound
}
func (s *stubAddressRepoForOrderCtrl) ReadAddressesByUserId(userId string) ([]entitiesAddress.Address, error) {
	return nil, nil
}
func (s *stubAddressRepoForOrderCtrl) FindMatchingUserAddress(userId string, address entitiesAddress.Address) (*entitiesAddress.Address, error) {
	return nil, apperrors.ErrNotFound
}

type stubCartRepoCtrl struct {
	items []entitiesCartProduct.CartProduct
}

func (s *stubCartRepoCtrl) AddProductCart(entitiesCartProduct.CartProduct) error { return nil }
func (s *stubCartRepoCtrl) ReadAllProductsCart(string) ([]entitiesCartProduct.CartProduct, error) {
	return s.items, nil
}
func (s *stubCartRepoCtrl) ReadProductsCart(string) (*entitiesCartProduct.CartProduct, error) {
	return nil, apperrors.ErrNotFound
}
func (s *stubCartRepoCtrl) RemoveProductCart(string) error         { return nil }
func (s *stubCartRepoCtrl) UpdateIsSelect(bool, string) error      { return nil }
func (s *stubCartRepoCtrl) UpdateAllIsSelect(bool, string) error   { return nil }
func (s *stubCartRepoCtrl) DecreaseProductCart(string, int64) error { return nil }

type stubAddressGwCtrl struct {
	cepRes *ports.CepLookupResult
	cepErr error
	cities []string
	ibgeEr error
}

func (s *stubAddressGwCtrl) LookupByCep(ctx context.Context, cep string) (*ports.CepLookupResult, error) {
	if s.cepErr != nil {
		return nil, s.cepErr
	}
	return s.cepRes, nil
}
func (s *stubAddressGwCtrl) ListCitiesByState(ctx context.Context, uf string) ([]string, error) {
	if s.ibgeEr != nil {
		return nil, s.ibgeEr
	}
	return s.cities, nil
}

func TestOrderControllerCreateWithAddress(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	withClaims := func(c *gin.Context) {
		c.Set("claims", &auth.Claims{Id: userID, Role: "user"})
		c.Next()
	}

	cartItems := []entitiesCartProduct.CartProduct{
		{
			Id:       uuid.New(),
			Quantity: 1,
			IsSelect: true,
			Cart:     &entitiesCart.Cart{Id: uuid.New(), User: &entitiesUser.User{ID: userID, Name: "Ana", Email: "ana@example.com"}},
			Product:  &entitiesProduct.Product{Id: uuid.New(), Name: "Jaqueta", Price: 250},
		},
	}

	t.Run("retorna 201 com objeto address completo ao cadastrar novo endereco", func(t *testing.T) {
		orderRepo := &stubOrderRepoCtrl{orders: make(map[string]*entitiesOrder.Order)}
		addrRepo := &stubAddressRepoForOrderCtrl{addresses: make(map[string]*entitiesAddress.Address)}
		validateUC := useCasesAddress.NewValidateAddressUseCase(&stubAddressGwCtrl{
			cepRes: &ports.CepLookupResult{Cep: "01001000", State: "SP", City: "São Paulo", Street: "Praça da Sé", Neighborhood: "Sé"},
			cities: []string{"SÃO PAULO"},
		})
		createUC := useCasesOrder.NewCreateOrderUseCase(orderRepo, addrRepo, nil, nil, useCasesCartProduct.NewReadAllCartProductsUseCase(&stubCartRepoCtrl{items: cartItems}), validateUC)
		ctrl := NewOrderController(createUC, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

		r := gin.New()
		r.POST("/api/v1/orders", withClaims, ctrl.CreateOrderMe)

		body := `{"address":{"cep":"01001-000","street":"Praça da Sé","number":"100","complement":"Sala 2","neighborhood":"Sé","city":"São Paulo","state":"SP"}}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, esperado 201 (body=%s)", w.Code, w.Body.String())
		}

		var payload struct {
			Body struct {
				Address struct {
					Cep    string `json:"cep"`
					Street string `json:"street"`
					Number string `json:"number"`
					City   string `json:"city"`
					State  string `json:"state"`
				} `json:"address"`
			} `json:"Body:"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
			t.Fatalf("erro json: %v", err)
		}
		if payload.Body.Address.Cep != "01001000" || payload.Body.Address.Number != "100" {
			t.Fatalf("address inesperado: %+v", payload.Body.Address)
		}
	})

	t.Run("retorna 201 ao confirmar pedido reutilizando address_id salvo", func(t *testing.T) {
		savedID := uuid.New()
		orderRepo := &stubOrderRepoCtrl{orders: make(map[string]*entitiesOrder.Order)}
		addrRepo := &stubAddressRepoForOrderCtrl{
			addresses: map[string]*entitiesAddress.Address{
				savedID.String(): {
					Id:     savedID,
					UserId: userID,
					Cep:    "01001000",
					Street: "Praça da Sé",
					Number: "100",
					City:   "São Paulo",
					State:  "SP",
				},
			},
		}
		createUC := useCasesOrder.NewCreateOrderUseCase(orderRepo, addrRepo, nil, nil, useCasesCartProduct.NewReadAllCartProductsUseCase(&stubCartRepoCtrl{items: cartItems}), nil)
		ctrl := NewOrderController(createUC, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

		r := gin.New()
		r.POST("/api/v1/orders", withClaims, ctrl.CreateOrderMe)

		body := `{"address_id":"` + savedID.String() + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, esperado 201 (body=%s)", w.Code, w.Body.String())
		}
		if !orderRepo.lastUsedExisting {
			t.Fatal("esperava useExistingAddress=true")
		}
	})

	t.Run("retorna 422 com erro por campo quando cidade diverge do CEP", func(t *testing.T) {
		orderRepo := &stubOrderRepoCtrl{orders: make(map[string]*entitiesOrder.Order)}
		addrRepo := &stubAddressRepoForOrderCtrl{addresses: make(map[string]*entitiesAddress.Address)}
		validateUC := useCasesAddress.NewValidateAddressUseCase(&stubAddressGwCtrl{
			cepRes: &ports.CepLookupResult{Cep: "01001000", State: "SP", City: "São Paulo"},
			cities: []string{"SÃO PAULO", "CAMPINAS"},
		})
		createUC := useCasesOrder.NewCreateOrderUseCase(orderRepo, addrRepo, nil, nil, useCasesCartProduct.NewReadAllCartProductsUseCase(&stubCartRepoCtrl{items: cartItems}), validateUC)
		ctrl := NewOrderController(createUC, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

		r := gin.New()
		r.POST("/api/v1/orders", withClaims, ctrl.CreateOrderMe)

		body := `{"address":{"cep":"01001-000","street":"Av Brasil","number":"50","city":"Campinas","state":"SP"}}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, esperado 422 (body=%s)", w.Code, w.Body.String())
		}

		var payload struct {
			Body []struct {
				Field   string `json:"campo"`
				Message string `json:"mensagem"`
			} `json:"Body:"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
			t.Fatalf("erro json: %v", err)
		}
		if len(payload.Body) == 0 || payload.Body[0].Field != "city" {
			t.Fatalf("esperava erro no campo city, obteve %+v", payload.Body)
		}
	})
}
