package use_cases

import (
	"e-commerce/core/apperrors"
	entitiesCart "e-commerce/core/entities/cart"
	entitiesUser "e-commerce/core/entities/user"
	"testing"

	"github.com/google/uuid"
)

type fakeCartRepo struct {
	readResults []*entitiesCart.Cart
	readErrs    []error
	createErr   error
	readCalls   int
	createCalls int
}

func (f *fakeCartRepo) CreateCart(entitiesCart.Cart) error {
	f.createCalls++
	return f.createErr
}

func (f *fakeCartRepo) ReadCartByUserId(string, string) (*entitiesCart.Cart, error) {
	idx := f.readCalls
	f.readCalls++
	var cart *entitiesCart.Cart
	var err error
	if idx < len(f.readResults) {
		cart = f.readResults[idx]
	}
	if idx < len(f.readErrs) {
		err = f.readErrs[idx]
	}
	return cart, err
}

func TestReadOrCreateCartUseCase_RecoversFromConcurrentConflict(t *testing.T) {
	userID := uuid.New()
	existingCart := &entitiesCart.Cart{
		Id:     uuid.New(),
		Status: entitiesCart.CartStatusActive,
		User:   &entitiesUser.User{ID: userID},
	}

	repo := &fakeCartRepo{
		readResults: []*entitiesCart.Cart{nil, existingCart},
		readErrs:    []error{apperrors.ErrNotFound, nil},
		createErr:   apperrors.ErrConflict,
	}

	uc := NewReadOrCreateCartUseCase(repo)
	got, err := uc.UseCase(userID.String())
	if err != nil {
		t.Fatalf("esperava recuperacao apos ErrConflict, got err=%v", err)
	}
	if got == nil || got.Id != existingCart.Id {
		t.Fatalf("cart retornado = %#v, esperado id=%s", got, existingCart.Id)
	}
	if repo.createCalls != 1 || repo.readCalls != 2 {
		t.Fatalf("chamadas inesperadas: readCalls=%d createCalls=%d", repo.readCalls, repo.createCalls)
	}
}
