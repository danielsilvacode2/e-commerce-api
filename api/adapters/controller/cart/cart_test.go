package controller

import (
	"bytes"
	"e-commerce/adapters/security/auth"
	"e-commerce/core/apperrors"
	entitiesCart "e-commerce/core/entities/cart"
	entitiesCartProduct "e-commerce/core/entities/cart_product"
	entitiesUser "e-commerce/core/entities/user"
	use_cases "e-commerce/core/use_cases/cart_product"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type stubCartProductRepo struct {
	allItems []entitiesCartProduct.CartProduct
	single   *entitiesCartProduct.CartProduct
	readErr  error
	removed  string
}

func (s *stubCartProductRepo) AddProductCart(entitiesCartProduct.CartProduct) error { return nil }
func (s *stubCartProductRepo) ReadAllProductsCart(string) ([]entitiesCartProduct.CartProduct, error) {
	return s.allItems, nil
}
func (s *stubCartProductRepo) ReadProductsCart(string) (*entitiesCartProduct.CartProduct, error) {
	if s.readErr != nil {
		return nil, s.readErr
	}
	return s.single, nil
}
func (s *stubCartProductRepo) RemoveProductCart(id string) error {
	s.removed = id
	return nil
}
func (s *stubCartProductRepo) UpdateIsSelect(bool, string) error      { return nil }
func (s *stubCartProductRepo) UpdateAllIsSelect(bool, string) error   { return nil }
func (s *stubCartProductRepo) DecreaseProductCart(string, int64) error { return nil }

func TestCartControllerEmptyAndRemove(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()

	withClaims := func(c *gin.Context) {
		c.Set("claims", &auth.Claims{Id: userID, Role: "user"})
		c.Next()
	}

	t.Run("retorna objeto CartResponseDTO com lista vazia quando carrinho esta vazio", func(t *testing.T) {
		repo := &stubCartProductRepo{allItems: nil}
		readAll := use_cases.NewReadAllCartProductsUseCase(repo)
		ctrl := NewCartController(
			use_cases.AddProductUseCase{},
			*readAll,
			use_cases.RemoveProductCartUseCase{},
			use_cases.UpdateIsSelectUseCase{},
			use_cases.UpdateAllIsSelectUseCase{},
			use_cases.DecreaseProductCartUseCase{},
		)

		r := gin.New()
		r.GET("/api/v1/cart", withClaims, ctrl.ReadAllProductCartMe)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, esperado 200", w.Code)
		}

		var payload struct {
			Body *struct {
				List []any `json:"list"`
			} `json:"Body:"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
			t.Fatalf("erro ao decodificar JSON: %v (body=%s)", err, w.Body.String())
		}
		if payload.Body == nil {
			t.Fatalf("Body: nao deveria ser null: %s", w.Body.String())
		}
		if payload.Body.List == nil || len(payload.Body.List) != 0 {
			t.Fatalf("esperava list vazia [], got %#v", payload.Body.List)
		}
	})

	t.Run("remover item retorna 204 sem corpo", func(t *testing.T) {
		itemID := uuid.New()
		repo := &stubCartProductRepo{
			single: &entitiesCartProduct.CartProduct{
				Id: itemID,
				Cart: &entitiesCart.Cart{
					User: &entitiesUser.User{ID: userID},
				},
			},
		}
		readOne := use_cases.NewReadCartProductUseCase(repo)
		removeUC := use_cases.NewRemoveProductCartUseCase(repo, *readOne)
		ctrl := NewCartController(
			use_cases.AddProductUseCase{},
			use_cases.ReadAllCartProductsUseCase{},
			*removeUC,
			use_cases.UpdateIsSelectUseCase{},
			use_cases.UpdateAllIsSelectUseCase{},
			use_cases.DecreaseProductCartUseCase{},
		)

		r := gin.New()
		r.DELETE("/api/v1/cart/:id", withClaims, ctrl.RemoveProductCartMe)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/cart/"+itemID.String(), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, esperado 204 (body=%s)", w.Code, w.Body.String())
		}
		if w.Body.Len() != 0 {
			t.Fatalf("corpo deveria estar vazio em 204, got %q", w.Body.String())
		}
		if repo.removed != itemID.String() {
			t.Fatalf("removed = %q, esperado %q", repo.removed, itemID.String())
		}
	})

	t.Run("remover item inexistente retorna 404", func(t *testing.T) {
		repo := &stubCartProductRepo{readErr: apperrors.ErrNotFound}
		readOne := use_cases.NewReadCartProductUseCase(repo)
		removeUC := use_cases.NewRemoveProductCartUseCase(repo, *readOne)
		ctrl := NewCartController(
			use_cases.AddProductUseCase{},
			use_cases.ReadAllCartProductsUseCase{},
			*removeUC,
			use_cases.UpdateIsSelectUseCase{},
			use_cases.UpdateAllIsSelectUseCase{},
			use_cases.DecreaseProductCartUseCase{},
		)

		r := gin.New()
		r.DELETE("/api/v1/cart/:id", withClaims, ctrl.RemoveProductCartMe)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/cart/inexistente", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, esperado 404 (body=%s)", w.Code, w.Body.String())
		}
	})

	t.Run("diminuir quantidade de 2 para 1 retorna 200", func(t *testing.T) {
		itemID := uuid.New()
		repo := &stubCartProductRepo{
			single: &entitiesCartProduct.CartProduct{
				Id:       itemID,
				Quantity: 2,
				Cart: &entitiesCart.Cart{
					User: &entitiesUser.User{ID: userID},
				},
			},
		}
		readOne := use_cases.NewReadCartProductUseCase(repo)
		decreaseUC := use_cases.NewDecreaseProductCartUseCase(repo, *readOne)
		ctrl := NewCartController(
			use_cases.AddProductUseCase{},
			use_cases.ReadAllCartProductsUseCase{},
			use_cases.RemoveProductCartUseCase{},
			use_cases.UpdateIsSelectUseCase{},
			use_cases.UpdateAllIsSelectUseCase{},
			*decreaseUC,
		)

		r := gin.New()
		r.PUT("/api/v1/cart/decrease/:id", withClaims, ctrl.DecreaseProductCartMe)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/cart/decrease/"+itemID.String(), bytes.NewBufferString(`{"quantity":1}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, esperado 200 (body=%s)", w.Code, w.Body.String())
		}
	})

	t.Run("diminuir quantidade abaixo de 1 retorna 422", func(t *testing.T) {
		itemID := uuid.New()
		repo := &stubCartProductRepo{
			single: &entitiesCartProduct.CartProduct{
				Id:       itemID,
				Quantity: 1,
				Cart: &entitiesCart.Cart{
					User: &entitiesUser.User{ID: userID},
				},
			},
		}
		readOne := use_cases.NewReadCartProductUseCase(repo)
		decreaseUC := use_cases.NewDecreaseProductCartUseCase(repo, *readOne)
		ctrl := NewCartController(
			use_cases.AddProductUseCase{},
			use_cases.ReadAllCartProductsUseCase{},
			use_cases.RemoveProductCartUseCase{},
			use_cases.UpdateIsSelectUseCase{},
			use_cases.UpdateAllIsSelectUseCase{},
			*decreaseUC,
		)

		r := gin.New()
		r.PUT("/api/v1/cart/decrease/:id", withClaims, ctrl.DecreaseProductCartMe)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/cart/decrease/"+itemID.String(), bytes.NewBufferString(`{"quantity":1}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, esperado 422 (body=%s)", w.Code, w.Body.String())
		}
	})
}
