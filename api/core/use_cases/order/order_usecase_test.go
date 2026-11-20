package use_cases_test

import (
	"errors"
	"testing"

	"e-commerce/core/apperrors"
	entitiesOrder "e-commerce/core/entities/order"
	entitiesOrderItem "e-commerce/core/entities/order_item"
	entitiesUser "e-commerce/core/entities/user"
	useCasesOrder "e-commerce/core/use_cases/order"

	"github.com/google/uuid"
)

type fakeOrderRepository struct {
	orders map[string]*entitiesOrder.Order
}

func newFakeOrderRepository() *fakeOrderRepository {
	return &fakeOrderRepository{
		orders: make(map[string]*entitiesOrder.Order),
	}
}

func (f *fakeOrderRepository) CreateOrder(order entitiesOrder.Order, cartProductIds []string) error {
	f.orders[order.Id.String()] = &order
	return nil
}

func (f *fakeOrderRepository) ReadOrderById(ID string) (*entitiesOrder.Order, error) {
	order, ok := f.orders[ID]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	return order, nil
}

func (f *fakeOrderRepository) ReadAllOrders(filter entitiesOrder.OrderFilter) (*entitiesOrder.PaginatedOrders, error) {
	var list []entitiesOrder.Order
	for _, o := range f.orders {
		if filter.UserId != "" && o.User.ID.String() != filter.UserId {
			continue
		}
		if filter.Status != "" && o.Status != filter.Status {
			continue
		}
		list = append(list, *o)
	}
	return &entitiesOrder.PaginatedOrders{
		Data:       list,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalRows:  int64(len(list)),
		TotalPages: 1,
	}, nil
}

func (f *fakeOrderRepository) UpdateOrderZipCode(zipCode string, ID string) error {
	order, ok := f.orders[ID]
	if !ok {
		return apperrors.ErrNotFound
	}
	order.ZipCode = zipCode
	return nil
}

func (f *fakeOrderRepository) UpdateOrderStatus(ID string, currentStatus string, newStatus string, restoreStock bool) error {
	order, ok := f.orders[ID]
	if !ok {
		return apperrors.ErrNotFound
	}
	if order.Status != currentStatus {
		return apperrors.ErrConflict
	}
	order.Status = newStatus
	return nil
}

func (f *fakeOrderRepository) DeleteOrder(ID string, restoreStock bool) error {
	if _, ok := f.orders[ID]; !ok {
		return apperrors.ErrNotFound
	}
	delete(f.orders, ID)
	return nil
}

func TestReadOrderByIdUseCase(t *testing.T) {
	repo := newFakeOrderRepository()
	usecase := useCasesOrder.NewReadOrderByIdUseCase(repo)

	userId := uuid.New()
	orderId := uuid.New()
	repo.orders[orderId.String()] = &entitiesOrder.Order{
		Id:     orderId,
		User:   &entitiesUser.User{ID: userId},
		Status: entitiesOrder.OrderStatusPreparing,
	}

	t.Run("User can read own order", func(t *testing.T) {
		order, err := usecase.UseCase(orderId.String(), userId.String(), "user")
		if err != nil {
			t.Fatalf("expected nil err, got %v", err)
		}
		if order.Id != orderId {
			t.Errorf("expected orderId %s, got %s", orderId, order.Id)
		}
	})

	t.Run("User forbidden to read other user order", func(t *testing.T) {
		otherUserId := uuid.New()
		_, err := usecase.UseCase(orderId.String(), otherUserId.String(), "user")
		if !errors.Is(err, apperrors.ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("Admin can read any order", func(t *testing.T) {
		adminId := uuid.New()
		order, err := usecase.UseCase(orderId.String(), adminId.String(), "admin")
		if err != nil {
			t.Fatalf("expected nil err, got %v", err)
		}
		if order.Id != orderId {
			t.Errorf("expected orderId %s, got %s", orderId, order.Id)
		}
	})
}

func TestUpdateOrderStatusUseCase(t *testing.T) {
	repo := newFakeOrderRepository()
	readUseCase := useCasesOrder.NewReadOrderByIdUseCase(repo)
	usecase := useCasesOrder.NewUpdateOrderStatusUseCase(repo, readUseCase)

	userId := uuid.New()
	orderId := uuid.New()
	repo.orders[orderId.String()] = &entitiesOrder.Order{
		Id:     orderId,
		User:   &entitiesUser.User{ID: userId},
		Status: entitiesOrder.OrderStatusPreparing,
	}

	t.Run("Valid transition PREPARING to SHIPPED", func(t *testing.T) {
		err := usecase.UseCase(entitiesOrder.OrderStatusShipped, orderId.String(), userId.String(), "admin")
		if err != nil {
			t.Fatalf("expected nil err, got %v", err)
		}
		if repo.orders[orderId.String()].Status != entitiesOrder.OrderStatusShipped {
			t.Fatalf("expected status SHIPPED, got %s", repo.orders[orderId.String()].Status)
		}
	})

	t.Run("Invalid transition SHIPPED to PREPARING", func(t *testing.T) {
		err := usecase.UseCase(entitiesOrder.OrderStatusPreparing, orderId.String(), userId.String(), "admin")
		if !errors.Is(err, apperrors.ErrUnprocessableEntity) {
			t.Fatalf("expected ErrUnprocessableEntity, got %v", err)
		}
	})

	t.Run("Invalid status", func(t *testing.T) {
		err := usecase.UseCase("INVALID_STATUS", orderId.String(), userId.String(), "admin")
		if !errors.Is(err, apperrors.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})
}

func TestCancelOrderUseCase(t *testing.T) {
	repo := newFakeOrderRepository()
	readUseCase := useCasesOrder.NewReadOrderByIdUseCase(repo)
	cancelUseCase := useCasesOrder.NewCancelOrderUseCase(repo, readUseCase)

	userId := uuid.New()
	orderId := uuid.New()
	repo.orders[orderId.String()] = &entitiesOrder.Order{
		Id:     orderId,
		User:   &entitiesUser.User{ID: userId},
		Status: entitiesOrder.OrderStatusPreparing,
	}

	t.Run("User can cancel order in PREPARING", func(t *testing.T) {
		err := cancelUseCase.UseCase(orderId.String(), userId.String(), "user")
		if err != nil {
			t.Fatalf("expected nil err, got %v", err)
		}
		if repo.orders[orderId.String()].Status != entitiesOrder.OrderStatusCanceled {
			t.Fatalf("expected status CANCELED, got %s", repo.orders[orderId.String()].Status)
		}
	})

	t.Run("Cannot cancel already canceled order", func(t *testing.T) {
		err := cancelUseCase.UseCase(orderId.String(), userId.String(), "user")
		if !errors.Is(err, apperrors.ErrUnprocessableEntity) {
			t.Fatalf("expected ErrUnprocessableEntity, got %v", err)
		}
	})
}

func TestOrderTotalCalculation(t *testing.T) {
	order := entitiesOrder.Order{
		Items: []entitiesOrderItem.OrderItem{
			{
				Quantity:  2,
				UnitPrice: 25.50,
			},
			{
				Quantity:  1,
				UnitPrice: 10.25,
			},
		},
	}

	expected := 61.25
	total := order.Total()
	if total != expected {
		t.Errorf("expected total %f, got %f", expected, total)
	}
}
