package use_cases_test

import (
	"errors"
	"testing"
	"time"

	"e-commerce/core/apperrors"
	entitiesOrder "e-commerce/core/entities/order"
	entitiesOrderItem "e-commerce/core/entities/order_item"
	entitiesPayment "e-commerce/core/entities/payment"
	entitiesUser "e-commerce/core/entities/user"
	"e-commerce/core/ports"
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
	return f.UpdateOrderStatusWithDetails(ID, currentStatus, newStatus, restoreStock, nil, nil, nil)
}

func (f *fakeOrderRepository) UpdateOrderStatusWithDetails(
	ID string,
	currentStatus string,
	newStatus string,
	restoreStock bool,
	trackingCode *string,
	refundReason *string,
	refundRejectionReason *string,
) error {
	order, ok := f.orders[ID]
	if !ok {
		return apperrors.ErrNotFound
	}
	if order.Status != currentStatus {
		return apperrors.ErrConflict
	}
	order.Status = newStatus
	if trackingCode != nil {
		order.TrackingCode = *trackingCode
	}
	if refundReason != nil {
		order.RefundReason = *refundReason
	}
	if refundRejectionReason != nil {
		order.RefundRejectionReason = *refundRejectionReason
	}
	return nil
}

func (f *fakeOrderRepository) DeleteOrder(ID string, restoreStock bool) error {
	if _, ok := f.orders[ID]; !ok {
		return apperrors.ErrNotFound
	}
	delete(f.orders, ID)
	return nil
}

type fakePaymentRepoForOrder struct {
	orderRepo *fakeOrderRepository
	payments  map[string]*entitiesPayment.Payment
}

func (f *fakePaymentRepoForOrder) CreatePayment(p entitiesPayment.Payment) error {
	f.payments[p.Id.String()] = &p
	return nil
}
func (f *fakePaymentRepoForOrder) ReadPaymentById(id string) (*entitiesPayment.Payment, error) {
	if p, ok := f.payments[id]; ok {
		return p, nil
	}
	return nil, apperrors.ErrNotFound
}
func (f *fakePaymentRepoForOrder) ReadPaymentByOrderId(orderId string) (*entitiesPayment.Payment, error) {
	for _, p := range f.payments {
		if p.OrderId.String() == orderId {
			return p, nil
		}
	}
	return nil, apperrors.ErrNotFound
}
func (f *fakePaymentRepoForOrder) ReadPaymentByIdempotencyKey(key string) (*entitiesPayment.Payment, error) {
	for _, p := range f.payments {
		if p.IdempotencyKey == key {
			return p, nil
		}
	}
	return nil, apperrors.ErrNotFound
}
func (f *fakePaymentRepoForOrder) ReadPaymentByExternalId(externalId string) (*entitiesPayment.Payment, error) {
	for _, p := range f.payments {
		if p.ExternalPaymentId == externalId {
			return p, nil
		}
	}
	return nil, apperrors.ErrNotFound
}
func (f *fakePaymentRepoForOrder) ApprovePaymentAndOrder(p entitiesPayment.Payment, orderId string) error {
	return nil
}
func (f *fakePaymentRepoForOrder) FailOrExpirePaymentAndCancelOrder(p entitiesPayment.Payment, orderId string) error {
	return nil
}
func (f *fakePaymentRepoForOrder) RefundPaymentAndOrder(orderId string, expectedOrderStatus string, restoreStock bool, refundReason *string) (*entitiesPayment.Payment, error) {
	order, ok := f.orderRepo.orders[orderId]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	if order.Status != expectedOrderStatus {
		return nil, apperrors.ErrConflict
	}
	order.Status = entitiesOrder.OrderStatusRefunded
	if refundReason != nil {
		order.RefundReason = *refundReason
	}
	for _, p := range f.payments {
		if p.OrderId.String() == orderId && p.Status == entitiesPayment.PaymentStatusApproved {
			now := time.Now()
			p.Status = entitiesPayment.PaymentStatusRefunded
			p.RefundedAt = &now
			return p, nil
		}
	}
	return nil, nil
}

type fakeOrderEventQueue struct {
	events []ports.OrderNotificationEvent
}

func (f *fakeOrderEventQueue) DeclareQueue(eventName string) error { return nil }
func (f *fakeOrderEventQueue) PublishOrderNotification(event ports.OrderNotificationEvent) error {
	f.events = append(f.events, event)
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
	eventQueue := &fakeOrderEventQueue{}
	readUseCase := useCasesOrder.NewReadOrderByIdUseCase(repo)
	usecase := useCasesOrder.NewUpdateOrderStatusUseCase(repo, readUseCase, eventQueue)

	userId := uuid.New()
	orderId := uuid.New()
	repo.orders[orderId.String()] = &entitiesOrder.Order{
		Id:     orderId,
		User:   &entitiesUser.User{ID: userId, Name: "Ana", Email: "ana@example.com"},
		Status: entitiesOrder.OrderStatusPreparing,
	}

	t.Run("Valid transition PREPARING to SHIPPED publishes notification with tracking", func(t *testing.T) {
		err := usecase.UseCase(entitiesOrder.OrderStatusShipped, "BR123456789BR", orderId.String(), userId.String(), "admin")
		if err != nil {
			t.Fatalf("expected nil err, got %v", err)
		}
		if repo.orders[orderId.String()].Status != entitiesOrder.OrderStatusShipped {
			t.Fatalf("expected status SHIPPED, got %s", repo.orders[orderId.String()].Status)
		}
		if repo.orders[orderId.String()].TrackingCode != "BR123456789BR" {
			t.Fatalf("expected tracking code BR123456789BR, got %s", repo.orders[orderId.String()].TrackingCode)
		}
		if len(eventQueue.events) != 1 || eventQueue.events[0].EventType != ports.EventOrderShipped {
			t.Fatalf("expected ORDER_SHIPPED event, got %+v", eventQueue.events)
		}
	})

	t.Run("Invalid transition SHIPPED to PREPARING", func(t *testing.T) {
		err := usecase.UseCase(entitiesOrder.OrderStatusPreparing, "", orderId.String(), userId.String(), "admin")
		if !errors.Is(err, apperrors.ErrUnprocessableEntity) {
			t.Fatalf("expected ErrUnprocessableEntity, got %v", err)
		}
	})

	t.Run("Invalid status", func(t *testing.T) {
		err := usecase.UseCase("INVALID_STATUS", "", orderId.String(), userId.String(), "admin")
		if !errors.Is(err, apperrors.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})
}

func TestRefundRulesUseCases(t *testing.T) {
	repo := newFakeOrderRepository()
	paymentRepo := &fakePaymentRepoForOrder{orderRepo: repo, payments: make(map[string]*entitiesPayment.Payment)}
	eventQueue := &fakeOrderEventQueue{}
	readUseCase := useCasesOrder.NewReadOrderByIdUseCase(repo)

	requestRefund := useCasesOrder.NewRequestRefundUseCase(repo, paymentRepo, readUseCase, nil, eventQueue)
	approveRefund := useCasesOrder.NewApproveRefundUseCase(repo, paymentRepo, readUseCase, nil, eventQueue)
	rejectRefund := useCasesOrder.NewRejectRefundUseCase(repo, readUseCase, eventQueue)

	userId := uuid.New()
	adminId := uuid.New()

	t.Run("PREPARING triggers immediate automatic refund without admin approval", func(t *testing.T) {
		orderId := uuid.New()
		payId := uuid.New()
		repo.orders[orderId.String()] = &entitiesOrder.Order{
			Id:     orderId,
			User:   &entitiesUser.User{ID: userId, Name: "Cliente", Email: "c@example.com"},
			Status: entitiesOrder.OrderStatusPreparing,
		}
		paymentRepo.payments[payId.String()] = &entitiesPayment.Payment{
			Id:      payId,
			OrderId: orderId,
			Status:  entitiesPayment.PaymentStatusApproved,
		}

		updated, err := requestRefund.UseCase(orderId.String(), "Desisti da compra", userId.String(), "user")
		if err != nil {
			t.Fatalf("expected nil err, got %v", err)
		}
		if updated.Status != entitiesOrder.OrderStatusRefunded {
			t.Fatalf("expected REFUNDED, got %s", updated.Status)
		}
		if paymentRepo.payments[payId.String()].Status != entitiesPayment.PaymentStatusRefunded {
			t.Fatalf("expected payment REFUNDED, got %s", paymentRepo.payments[payId.String()].Status)
		}
	})

	t.Run("SHIPPED blocks customer refund", func(t *testing.T) {
		orderId := uuid.New()
		repo.orders[orderId.String()] = &entitiesOrder.Order{
			Id:     orderId,
			User:   &entitiesUser.User{ID: userId, Name: "Cliente", Email: "c@example.com"},
			Status: entitiesOrder.OrderStatusShipped,
		}

		_, err := requestRefund.UseCase(orderId.String(), "Quero estorno", userId.String(), "user")
		if !errors.Is(err, apperrors.ErrUnprocessableEntity) {
			t.Fatalf("expected ErrUnprocessableEntity for SHIPPED, got %v", err)
		}
	})

	t.Run("DELIVERED transitions to REFUND_REQUESTED and admin can reject or approve", func(t *testing.T) {
		orderId := uuid.New()
		payId := uuid.New()
		repo.orders[orderId.String()] = &entitiesOrder.Order{
			Id:     orderId,
			User:   &entitiesUser.User{ID: userId, Name: "Cliente", Email: "c@example.com"},
			Status: entitiesOrder.OrderStatusDelivered,
		}
		paymentRepo.payments[payId.String()] = &entitiesPayment.Payment{
			Id:      payId,
			OrderId: orderId,
			Status:  entitiesPayment.PaymentStatusApproved,
		}

		// Sem motivo -> ErrInvalidInput
		if _, err := requestRefund.UseCase(orderId.String(), "", userId.String(), "user"); !errors.Is(err, apperrors.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput when reason is empty, got %v", err)
		}

		// Com motivo -> REFUND_REQUESTED
		updated, err := requestRefund.UseCase(orderId.String(), "Produto com defeito", userId.String(), "user")
		if err != nil || updated.Status != entitiesOrder.OrderStatusRefundRequested {
			t.Fatalf("expected REFUND_REQUESTED, got status=%v err=%v", updated, err)
		}

		// Admin recusa -> volta para DELIVERED com justificativa
		rejected, err := rejectRefund.UseCase(orderId.String(), "Fora do prazo de 7 dias", adminId.String(), "admin")
		if err != nil || rejected.Status != entitiesOrder.OrderStatusDelivered || rejected.RefundRejectionReason != "Fora do prazo de 7 dias" {
			t.Fatalf("expected DELIVERED with rejection reason, got %+v err=%v", rejected, err)
		}

		// Nova solicitacao e aprovacao do admin -> REFUNDED
		if _, err := requestRefund.UseCase(orderId.String(), "Nova evidencia do defeito", userId.String(), "user"); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		approved, err := approveRefund.UseCase(orderId.String(), adminId.String(), "admin")
		if err != nil || approved.Status != entitiesOrder.OrderStatusRefunded {
			t.Fatalf("expected REFUNDED after admin approval, got %+v err=%v", approved, err)
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
