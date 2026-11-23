package use_cases_test

import (
	adapterPayment "e-commerce/adapters/payment"
	"e-commerce/core/apperrors"
	entitiesOrder "e-commerce/core/entities/order"
	entitiesOrderItem "e-commerce/core/entities/order_item"
	entitiesPayment "e-commerce/core/entities/payment"
	entitiesUser "e-commerce/core/entities/user"
	"e-commerce/core/ports"
	useCasesOrder "e-commerce/core/use_cases/order"
	useCasesPayment "e-commerce/core/use_cases/payment"
	"testing"
	"time"

	"github.com/google/uuid"
)

type stubOrderRepo struct {
	orders map[string]*entitiesOrder.Order
}

func (s *stubOrderRepo) CreateOrder(order entitiesOrder.Order, cartProductIds []string) error {
	s.orders[order.Id.String()] = &order
	return nil
}
func (s *stubOrderRepo) ReadOrderById(ID string) (*entitiesOrder.Order, error) {
	o, ok := s.orders[ID]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	return o, nil
}
func (s *stubOrderRepo) ReadAllOrders(filter entitiesOrder.OrderFilter) (*entitiesOrder.PaginatedOrders, error) {
	return nil, nil
}
func (s *stubOrderRepo) UpdateOrderZipCode(zipCode string, ID string) error { return nil }
func (s *stubOrderRepo) UpdateOrderStatus(ID, cur, next string, restore bool) error {
	return nil
}
func (s *stubOrderRepo) UpdateOrderStatusWithDetails(ID, cur, next string, restore bool, t, r, rr *string) error {
	return nil
}
func (s *stubOrderRepo) DeleteOrder(ID string, restore bool) error { return nil }

type stubPaymentRepo struct {
	orderRepo    *stubOrderRepo
	byId         map[string]*entitiesPayment.Payment
	byKey        map[string]*entitiesPayment.Payment
	approveCalls int
	failCalls    int
}

func newStubPaymentRepo(orderRepo *stubOrderRepo) *stubPaymentRepo {
	return &stubPaymentRepo{
		orderRepo: orderRepo,
		byId:      make(map[string]*entitiesPayment.Payment),
		byKey:     make(map[string]*entitiesPayment.Payment),
	}
}

func (r *stubPaymentRepo) CreatePayment(p entitiesPayment.Payment) error {
	if _, exists := r.byKey[p.IdempotencyKey]; exists {
		return apperrors.ErrConflict
	}
	cp := p
	r.byId[p.Id.String()] = &cp
	r.byKey[p.IdempotencyKey] = &cp
	return nil
}
func (r *stubPaymentRepo) ReadPaymentById(id string) (*entitiesPayment.Payment, error) {
	if p, ok := r.byId[id]; ok {
		return p, nil
	}
	return nil, apperrors.ErrNotFound
}
func (r *stubPaymentRepo) ReadPaymentByOrderId(orderId string) (*entitiesPayment.Payment, error) {
	var latest *entitiesPayment.Payment
	for _, p := range r.byId {
		if p.OrderId.String() == orderId {
			latest = p
		}
	}
	if latest == nil {
		return nil, apperrors.ErrNotFound
	}
	return latest, nil
}
func (r *stubPaymentRepo) ReadPaymentByIdempotencyKey(key string) (*entitiesPayment.Payment, error) {
	if p, ok := r.byKey[key]; ok {
		return p, nil
	}
	return nil, apperrors.ErrNotFound
}
func (r *stubPaymentRepo) ReadPaymentByExternalId(externalId string) (*entitiesPayment.Payment, error) {
	for _, p := range r.byId {
		if p.ExternalPaymentId == externalId {
			return p, nil
		}
	}
	return nil, apperrors.ErrNotFound
}
func (r *stubPaymentRepo) ApprovePaymentAndOrder(p entitiesPayment.Payment, orderId string) error {
	r.approveCalls++
	stored := r.byId[p.Id.String()]
	stored.Status = entitiesPayment.PaymentStatusApproved
	now := time.Now()
	stored.PaidAt = &now
	r.orderRepo.orders[orderId].Status = entitiesOrder.OrderStatusPreparing
	return nil
}
func (r *stubPaymentRepo) FailOrExpirePaymentAndCancelOrder(p entitiesPayment.Payment, orderId string) error {
	r.failCalls++
	stored := r.byId[p.Id.String()]
	stored.Status = p.Status
	r.orderRepo.orders[orderId].Status = entitiesOrder.OrderStatusCanceled
	return nil
}
func (r *stubPaymentRepo) RefundPaymentAndOrder(orderId, expectedStatus string, restore bool, reason *string) (*entitiesPayment.Payment, error) {
	return nil, nil
}

type stubEventQueue struct {
	events []ports.OrderNotificationEvent
}

func (s *stubEventQueue) DeclareQueue(name string) error { return nil }
func (s *stubEventQueue) PublishOrderNotification(ev ports.OrderNotificationEvent) error {
	s.events = append(s.events, ev)
	return nil
}

func TestProcessPaymentIdempotencyAndTransitions(t *testing.T) {
	orderRepo := &stubOrderRepo{orders: make(map[string]*entitiesOrder.Order)}
	paymentRepo := newStubPaymentRepo(orderRepo)
	eventQueue := &stubEventQueue{}
	gateway := adapterPayment.NewMockPaymentGateway()
	readOrder := useCasesOrder.NewReadOrderByIdUseCase(orderRepo)

	processUC := useCasesPayment.NewProcessPaymentUseCase(paymentRepo, readOrder, gateway, eventQueue)
	webhookUC := useCasesPayment.NewWebhookPaymentUseCase(paymentRepo, orderRepo, gateway, eventQueue)

	userId := uuid.New()

	t.Run("Credit card approved transitions order to PREPARING and is idempotent on retry", func(t *testing.T) {
		orderId := uuid.New()
		orderRepo.orders[orderId.String()] = &entitiesOrder.Order{
			Id:     orderId,
			User:   &entitiesUser.User{ID: userId, Name: "Joao", Email: "joao@example.com"},
			Status: entitiesOrder.OrderStatusPendingPayment,
			Items:  []entitiesOrderItem.OrderItem{{Quantity: 1, UnitPrice: 199.90}},
		}

		input := useCasesPayment.ProcessPaymentInput{
			OrderId:        orderId.String(),
			UserId:         userId.String(),
			Role:           "user",
			IdempotencyKey: "idem-card-1",
			Method:         entitiesPayment.PaymentMethodCreditCard,
			Installments:   2,
			CardNumber:     "4111111111111111",
			CardHolderName: "JOAO SILVA",
			CardExpMonth:   12,
			CardExpYear:    2030,
			CardCVV:        "123",
		}

		first, err := processUC.UseCase(input)
		if err != nil {
			t.Fatalf("expected nil err, got %v", err)
		}
		if first.Status != entitiesPayment.PaymentStatusApproved {
			t.Fatalf("expected APPROVED, got %s", first.Status)
		}
		if orderRepo.orders[orderId.String()].Status != entitiesOrder.OrderStatusPreparing {
			t.Fatalf("expected order PREPARING, got %s", orderRepo.orders[orderId.String()].Status)
		}

		// Retry com a mesma Idempotency-Key nao deve reprocessar nem chamar ApprovePaymentAndOrder novamente
		second, err := processUC.UseCase(input)
		if err != nil {
			t.Fatalf("expected nil err on idempotent retry, got %v", err)
		}
		if second.Id != first.Id {
			t.Fatalf("expected same payment id %s, got %s", first.Id, second.Id)
		}
		if paymentRepo.approveCalls != 1 {
			t.Fatalf("expected exactly 1 approve call, got %d", paymentRepo.approveCalls)
		}
	})

	t.Run("Credit card ending 0000 fails payment and cancels order", func(t *testing.T) {
		orderId := uuid.New()
		orderRepo.orders[orderId.String()] = &entitiesOrder.Order{
			Id:     orderId,
			User:   &entitiesUser.User{ID: userId, Name: "Maria", Email: "maria@example.com"},
			Status: entitiesOrder.OrderStatusPendingPayment,
			Items:  []entitiesOrderItem.OrderItem{{Quantity: 1, UnitPrice: 150.00}},
		}

		res, err := processUC.UseCase(useCasesPayment.ProcessPaymentInput{
			OrderId:        orderId.String(),
			UserId:         userId.String(),
			Role:           "user",
			IdempotencyKey: "idem-card-fail-1",
			Method:         entitiesPayment.PaymentMethodCreditCard,
			Installments:   1,
			CardNumber:     "4111111111110000",
			CardHolderName: "MARIA",
			CardExpMonth:   12,
			CardExpYear:    2030,
			CardCVV:        "123",
		})
		if err != nil {
			t.Fatalf("expected nil err, got %v", err)
		}
		if res.Status != entitiesPayment.PaymentStatusFailed {
			t.Fatalf("expected FAILED, got %s", res.Status)
		}
		if orderRepo.orders[orderId.String()].Status != entitiesOrder.OrderStatusCanceled {
			t.Fatalf("expected order CANCELED, got %s", orderRepo.orders[orderId.String()].Status)
		}
	})

	t.Run("Pix creates PENDING payment and webhook approval is idempotent", func(t *testing.T) {
		orderId := uuid.New()
		orderRepo.orders[orderId.String()] = &entitiesOrder.Order{
			Id:     orderId,
			User:   &entitiesUser.User{ID: userId, Name: "Pedro", Email: "pedro@example.com"},
			Status: entitiesOrder.OrderStatusPendingPayment,
			Items:  []entitiesOrderItem.OrderItem{{Quantity: 2, UnitPrice: 50.00}},
		}

		pix, err := processUC.UseCase(useCasesPayment.ProcessPaymentInput{
			OrderId:        orderId.String(),
			UserId:         userId.String(),
			Role:           "user",
			IdempotencyKey: "idem-pix-1",
			Method:         entitiesPayment.PaymentMethodPix,
		})
		if err != nil {
			t.Fatalf("expected nil err, got %v", err)
		}
		if pix.Status != entitiesPayment.PaymentStatusPending || pix.PixCopyPaste == "" || pix.PixQrCode == "" {
			t.Fatalf("expected pending pix with qr code, got %+v", pix)
		}

		// Primeiro webhook aprova
		approved, err := webhookUC.UseCase(pix.Id.String(), "", useCasesPayment.PaymentActionApprove)
		if err != nil || approved.Status != entitiesPayment.PaymentStatusApproved {
			t.Fatalf("expected APPROVED, got %+v err=%v", approved, err)
		}
		if orderRepo.orders[orderId.String()].Status != entitiesOrder.OrderStatusPreparing {
			t.Fatalf("expected PREPARING, got %s", orderRepo.orders[orderId.String()].Status)
		}

		// Reenvio de webhook nao reprocessa
		callsBefore := paymentRepo.approveCalls
		again, err := webhookUC.UseCase(pix.Id.String(), "", useCasesPayment.PaymentActionApprove)
		if err != nil || again.Status != entitiesPayment.PaymentStatusApproved {
			t.Fatalf("expected idempotent APPROVED, got %+v err=%v", again, err)
		}
		if paymentRepo.approveCalls != callsBefore {
			t.Fatalf("webhook should be idempotent, approveCalls changed from %d to %d", callsBefore, paymentRepo.approveCalls)
		}
	})
}
