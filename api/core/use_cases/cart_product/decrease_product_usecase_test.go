package use_cases

import (
	"e-commerce/core/apperrors"
	entitiesCart "e-commerce/core/entities/cart"
	entitiesCartProduct "e-commerce/core/entities/cart_product"
	entitiesUser "e-commerce/core/entities/user"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type fakeCartProductRepo struct {
	item        *entitiesCartProduct.CartProduct
	readErr     error
	decreasedBy int64
	decreaseErr error
}

func (f *fakeCartProductRepo) AddProductCart(entitiesCartProduct.CartProduct) error { return nil }
func (f *fakeCartProductRepo) ReadAllProductsCart(string) ([]entitiesCartProduct.CartProduct, error) {
	return nil, nil
}
func (f *fakeCartProductRepo) ReadProductsCart(string) (*entitiesCartProduct.CartProduct, error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	return f.item, nil
}
func (f *fakeCartProductRepo) RemoveProductCart(string) error                { return nil }
func (f *fakeCartProductRepo) UpdateIsSelect(bool, string) error             { return nil }
func (f *fakeCartProductRepo) UpdateAllIsSelect(bool, string) error          { return nil }
func (f *fakeCartProductRepo) DecreaseProductCart(_ string, qty int64) error {
	f.decreasedBy = qty
	return f.decreaseErr
}

func TestDecreaseProductCartUseCase(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()

	t.Run("permite diminuir quantidade de 2 para 1", func(t *testing.T) {
		repo := &fakeCartProductRepo{
			item: &entitiesCartProduct.CartProduct{
				Quantity: 2,
				Cart: &entitiesCart.Cart{
					User: &entitiesUser.User{ID: ownerID},
				},
			},
		}
		readUC := NewReadCartProductUseCase(repo)
		uc := NewDecreaseProductCartUseCase(repo, *readUC)

		if err := uc.UseCase("item-1", 1, ownerID.String()); err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if repo.decreasedBy != 1 {
			t.Fatalf("decreasedBy = %d, esperado 1", repo.decreasedBy)
		}
	})

	t.Run("bloqueia diminuir quantidade para menos que 1", func(t *testing.T) {
		repo := &fakeCartProductRepo{
			item: &entitiesCartProduct.CartProduct{
				Quantity: 1,
				Cart: &entitiesCart.Cart{
					User: &entitiesUser.User{ID: ownerID},
				},
			},
		}
		readUC := NewReadCartProductUseCase(repo)
		uc := NewDecreaseProductCartUseCase(repo, *readUC)

		if err := uc.UseCase("item-1", 1, ownerID.String()); !errors.Is(err, apperrors.ErrUnprocessableEntity) {
			t.Fatalf("erro = %v, esperado ErrUnprocessableEntity", err)
		}
	})

	t.Run("bloqueia usuario que nao e dono do carrinho", func(t *testing.T) {
		repo := &fakeCartProductRepo{
			item: &entitiesCartProduct.CartProduct{
				Quantity: 3,
				Cart: &entitiesCart.Cart{
					User: &entitiesUser.User{ID: ownerID},
				},
			},
		}
		readUC := NewReadCartProductUseCase(repo)
		uc := NewDecreaseProductCartUseCase(repo, *readUC)

		if err := uc.UseCase("item-1", 1, otherID.String()); !errors.Is(err, apperrors.ErrForbidden) {
			t.Fatalf("erro = %v, esperado ErrForbidden", err)
		}
	})

	t.Run("propaga ErrNotFound quando item nao existe", func(t *testing.T) {
		repo := &fakeCartProductRepo{readErr: apperrors.ErrNotFound}
		readUC := NewReadCartProductUseCase(repo)
		uc := NewDecreaseProductCartUseCase(repo, *readUC)

		if err := uc.UseCase("item-inexistente", 1, ownerID.String()); !errors.Is(err, apperrors.ErrNotFound) {
			t.Fatalf("erro = %v, esperado ErrNotFound", err)
		}
	})
}
