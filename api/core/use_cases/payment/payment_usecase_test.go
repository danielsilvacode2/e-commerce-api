package use_cases_test

import (
	adapterPayment "e-commerce/adapters/payment"
	adapterRabbitMQ "e-commerce/adapters/rabbitmq"
	"e-commerce/core/apperrors"
	entitiesOrder "e-commerce/core/entities/order"
	entitiesOrderItem "e-commerce/core/entities/order_item"
	entitiesPayment "e-commerce/core/entities/payment"
	entitiesUser "e-commerce/core/entities/user"
	"e-commerce/core/ports"
	useCasesOrder "e-commerce/core/use_cases/order"
	useCasesPayment "e-commerce/core/use_cases/payment"
	"errors"
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
	orderRepo       *stubOrderRepo
	byId            map[string]*entitiesPayment.Payment
	byKey           map[string]*entitiesPayment.Payment
	approveCalls    int
	failCalls       int
	createErr       error
	updateGwErr     error
	approveErr      error
	failErr         error
	deleteCalls     int
}

func newStubPaymentRepo(orderRepo *stubOrderRepo) *stubPaymentRepo {
	return &stubPaymentRepo{
		orderRepo: orderRepo,
		byId:      make(map[string]*entitiesPayment.Payment),
		byKey:     make(map[string]*entitiesPayment.Payment),
	}
}

func (r *stubPaymentRepo) CreatePayment(p entitiesPayment.Payment) error {
	if r.createErr != nil {
		return r.createErr
	}
	if _, exists := r.byKey[p.IdempotencyKey]; exists {
		return apperrors.ErrConflict
	}
	cp := p
	r.byId[p.Id.String()] = &cp
	r.byKey[p.IdempotencyKey] = &cp
	return nil
}
func (r *stubPaymentRepo) UpdatePaymentGatewayData(p entitiesPayment.Payment) error {
	if r.updateGwErr != nil {
		return r.updateGwErr
	}
	stored, ok := r.byId[p.Id.String()]
	if !ok {
		return apperrors.ErrNotFound
	}
	if p.ExternalPaymentId != "" {
		stored.ExternalPaymentId = p.ExternalPaymentId
	}
	if p.PixQrCode != "" {
		stored.PixQrCode = p.PixQrCode
	}
	if p.PixCopyPaste != "" {
		stored.PixCopyPaste = p.PixCopyPaste
	}
	if p.CardLast4 != "" {
		stored.CardLast4 = p.CardLast4
	}
	if p.CardBrand != "" {
		stored.CardBrand = p.CardBrand
	}
	if p.ExpiresAt != nil {
		stored.ExpiresAt = p.ExpiresAt
	}
	return nil
}
func (r *stubPaymentRepo) DeletePayment(id string) error {
	r.deleteCalls++
	if stored, ok := r.byId[id]; ok {
		delete(r.byKey, stored.IdempotencyKey)
		delete(r.byId, id)
	}
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
	if r.approveErr != nil {
		return r.approveErr
	}
	r.approveCalls++
	stored := r.byId[p.Id.String()]
	stored.Status = entitiesPayment.PaymentStatusApproved
	if p.ExternalPaymentId != "" {
		stored.ExternalPaymentId = p.ExternalPaymentId
	}
	if p.CardLast4 != "" {
		stored.CardLast4 = p.CardLast4
	}
	if p.CardBrand != "" {
		stored.CardBrand = p.CardBrand
	}
	now := time.Now()
	stored.PaidAt = &now
	r.orderRepo.orders[orderId].Status = entitiesOrder.OrderStatusPreparing
	return nil
}
func (r *stubPaymentRepo) FailOrExpirePaymentAndCancelOrder(p entitiesPayment.Payment, orderId string) error {
	if r.failErr != nil {
		return r.failErr
	}
	r.failCalls++
	stored := r.byId[p.Id.String()]
	stored.Status = p.Status
	if p.ExternalPaymentId != "" {
		stored.ExternalPaymentId = p.ExternalPaymentId
	}
	if p.CardLast4 != "" {
		stored.CardLast4 = p.CardLast4
	}
	if p.CardBrand != "" {
		stored.CardBrand = p.CardBrand
	}
	r.orderRepo.orders[orderId].Status = entitiesOrder.OrderStatusCanceled
	return nil
}
func (r *stubPaymentRepo) RefundPaymentAndOrder(orderId, expectedStatus string, restore bool, reason *string) (*entitiesPayment.Payment, error) {
	return nil, nil
}

type stubPaymentSyncQueue struct {
	events []ports.PaymentSyncEvent
}

func (s *stubPaymentSyncQueue) DeclareQueue(name string) error { return nil }
func (s *stubPaymentSyncQueue) PublishPaymentSync(ev ports.PaymentSyncEvent) error {
	s.events = append(s.events, ev)
	return nil
}

type spyPaymentGateway struct {
	inner      ports.PaymentGateway
	pixCalls   int
	cardCalls  int
	cardErr    error
	pixErr     error
}

func (g *spyPaymentGateway) CreatePixPayment(orderId string, amount float64, payerEmail string, idempotencyKey string) (*ports.PixGatewayResult, error) {
	g.pixCalls++
	if g.pixErr != nil {
		return nil, g.pixErr
	}
	return g.inner.CreatePixPayment(orderId, amount, payerEmail, idempotencyKey)
}
func (g *spyPaymentGateway) ProcessCreditCardPayment(input ports.CreditCardGatewayInput) (*ports.CardGatewayResult, error) {
	g.cardCalls++
	if g.cardErr != nil {
		return nil, g.cardErr
	}
	return g.inner.ProcessCreditCardPayment(input)
}
func (g *spyPaymentGateway) RefundPayment(externalPaymentId string, amount float64) error {
	return g.inner.RefundPayment(externalPaymentId, amount)
}
func (g *spyPaymentGateway) GetPaymentStatus(externalPaymentId string) (string, error) {
	return g.inner.GetPaymentStatus(externalPaymentId)
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

	t.Run("Webhook rejects approval if order total diverges from payment amount", func(t *testing.T) {
		orderId := uuid.New()
		orderRepo.orders[orderId.String()] = &entitiesOrder.Order{
			Id:     orderId,
			User:   &entitiesUser.User{ID: userId, Name: "Lucas", Email: "lucas@example.com"},
			Status: entitiesOrder.OrderStatusPendingPayment,
			Items:  []entitiesOrderItem.OrderItem{{Quantity: 1, UnitPrice: 10.00}},
		}

		pix, err := processUC.UseCase(useCasesPayment.ProcessPaymentInput{
			OrderId:        orderId.String(),
			UserId:         userId.String(),
			Role:           "user",
			IdempotencyKey: "idem-pix-tamper",
			Method:         entitiesPayment.PaymentMethodPix,
		})
		if err != nil {
			t.Fatalf("expected nil err, got %v", err)
		}

		// Simula tentativa de adicionar item caro apos gerar o Pix
		orderRepo.orders[orderId.String()].Items = append(orderRepo.orders[orderId.String()].Items, entitiesOrderItem.OrderItem{
			Quantity:  1,
			UnitPrice: 9990.00,
		})

		_, err = webhookUC.UseCase(pix.Id.String(), "", useCasesPayment.PaymentActionApprove)
		if !errors.Is(err, apperrors.ErrConflict) {
			t.Fatalf("expected ErrConflict on amount mismatch, got %v", err)
		}
		if orderRepo.orders[orderId.String()].Status != entitiesOrder.OrderStatusPendingPayment {
			t.Fatalf("order must remain PENDING_PAYMENT, got %s", orderRepo.orders[orderId.String()].Status)
		}
	})
}

func TestPrePersistAndRabbitMQFallback(t *testing.T) {
	userId := uuid.New()

	t.Run("Does not call gateway when pre-persist fails with database unavailable", func(t *testing.T) {
		orderRepo := &stubOrderRepo{orders: make(map[string]*entitiesOrder.Order)}
		paymentRepo := newStubPaymentRepo(orderRepo)
		paymentRepo.createErr = apperrors.ErrDatabaseUnavailable
		eventQueue := &stubEventQueue{}
		syncQueue := &stubPaymentSyncQueue{}
		spyGw := &spyPaymentGateway{inner: adapterPayment.NewMockPaymentGateway()}
		readOrder := useCasesOrder.NewReadOrderByIdUseCase(orderRepo)

		processUC := useCasesPayment.NewProcessPaymentUseCase(paymentRepo, readOrder, spyGw, eventQueue, syncQueue)

		orderId := uuid.New()
		orderRepo.orders[orderId.String()] = &entitiesOrder.Order{
			Id:     orderId,
			User:   &entitiesUser.User{ID: userId, Name: "Ana", Email: "ana@example.com"},
			Status: entitiesOrder.OrderStatusPendingPayment,
			Items:  []entitiesOrderItem.OrderItem{{Quantity: 1, UnitPrice: 120.00}},
		}

		_, err := processUC.UseCase(useCasesPayment.ProcessPaymentInput{
			OrderId:        orderId.String(),
			UserId:         userId.String(),
			Role:           "user",
			IdempotencyKey: "idem-db-down-before-gw",
			Method:         entitiesPayment.PaymentMethodCreditCard,
			Installments:   1,
			CardNumber:     "4111111111111111",
			CardHolderName: "ANA",
			CardExpMonth:   12,
			CardExpYear:    2030,
			CardCVV:        "123",
		})
		if !errors.Is(err, apperrors.ErrDatabaseUnavailable) {
			t.Fatalf("expected ErrDatabaseUnavailable, got %v", err)
		}
		if spyGw.cardCalls != 0 {
			t.Fatalf("gateway must NOT be called when DB is unavailable before charge, got %d calls", spyGw.cardCalls)
		}
	})

	t.Run("Deletes orphan PENDING payment when gateway returns error", func(t *testing.T) {
		orderRepo := &stubOrderRepo{orders: make(map[string]*entitiesOrder.Order)}
		paymentRepo := newStubPaymentRepo(orderRepo)
		eventQueue := &stubEventQueue{}
		syncQueue := &stubPaymentSyncQueue{}
		spyGw := &spyPaymentGateway{
			inner:   adapterPayment.NewMockPaymentGateway(),
			cardErr: errors.New("gateway timeout"),
		}
		readOrder := useCasesOrder.NewReadOrderByIdUseCase(orderRepo)

		processUC := useCasesPayment.NewProcessPaymentUseCase(paymentRepo, readOrder, spyGw, eventQueue, syncQueue)

		orderId := uuid.New()
		orderRepo.orders[orderId.String()] = &entitiesOrder.Order{
			Id:     orderId,
			User:   &entitiesUser.User{ID: userId, Name: "Bruno", Email: "bruno@example.com"},
			Status: entitiesOrder.OrderStatusPendingPayment,
			Items:  []entitiesOrderItem.OrderItem{{Quantity: 1, UnitPrice: 80.00}},
		}

		_, err := processUC.UseCase(useCasesPayment.ProcessPaymentInput{
			OrderId:        orderId.String(),
			UserId:         userId.String(),
			Role:           "user",
			IdempotencyKey: "idem-gw-err",
			Method:         entitiesPayment.PaymentMethodCreditCard,
			Installments:   1,
			CardNumber:     "4111111111111111",
			CardHolderName: "BRUNO",
			CardExpMonth:   12,
			CardExpYear:    2030,
			CardCVV:        "123",
		})
		if err == nil {
			t.Fatal("expected gateway error, got nil")
		}
		if paymentRepo.deleteCalls != 1 {
			t.Fatalf("expected orphan payment to be deleted once, got %d", paymentRepo.deleteCalls)
		}
		if _, lookupErr := paymentRepo.ReadPaymentByIdempotencyKey("idem-gw-err"); !errors.Is(lookupErr, apperrors.ErrNotFound) {
			t.Fatalf("expected orphan payment removed from repo, got err=%v", lookupErr)
		}
	})

	t.Run("Enqueues approved credit card in RabbitMQ fallback when DB fails after gateway and consumer reconciles", func(t *testing.T) {
		orderRepo := &stubOrderRepo{orders: make(map[string]*entitiesOrder.Order)}
		paymentRepo := newStubPaymentRepo(orderRepo)
		paymentRepo.approveErr = apperrors.ErrDatabaseUnavailable
		eventQueue := &stubEventQueue{}
		syncQueue := &stubPaymentSyncQueue{}
		gateway := adapterPayment.NewMockPaymentGateway()
		readOrder := useCasesOrder.NewReadOrderByIdUseCase(orderRepo)

		processUC := useCasesPayment.NewProcessPaymentUseCase(paymentRepo, readOrder, gateway, eventQueue, syncQueue)

		orderId := uuid.New()
		orderRepo.orders[orderId.String()] = &entitiesOrder.Order{
			Id:     orderId,
			User:   &entitiesUser.User{ID: userId, Name: "Carla", Email: "carla@example.com"},
			Status: entitiesOrder.OrderStatusPendingPayment,
			Items:  []entitiesOrderItem.OrderItem{{Quantity: 1, UnitPrice: 250.00}},
		}

		res, err := processUC.UseCase(useCasesPayment.ProcessPaymentInput{
			OrderId:        orderId.String(),
			UserId:         userId.String(),
			Role:           "user",
			IdempotencyKey: "idem-card-fallback-1",
			Method:         entitiesPayment.PaymentMethodCreditCard,
			Installments:   3,
			CardNumber:     "4111111111111111",
			CardHolderName: "CARLA",
			CardExpMonth:   12,
			CardExpYear:    2030,
			CardCVV:        "123",
		})
		if err != nil {
			t.Fatalf("expected nil err with RabbitMQ fallback, got %v", err)
		}
		if res.Status != entitiesPayment.PaymentStatusApproved || res.ExternalPaymentId == "" {
			t.Fatalf("expected APPROVED payment with external id, got %+v", res)
		}
		if len(syncQueue.events) != 1 || syncQueue.events[0].Action != ports.PaymentSyncActionApprove {
			t.Fatalf("expected 1 APPROVE payment_sync event, got %+v", syncQueue.events)
		}

		// Banco volta ao ar e consumer processa o evento enfileirado
		paymentRepo.approveErr = nil
		consumer := adapterRabbitMQ.NewPaymentSyncConsumer(nil, paymentRepo, eventQueue).(*adapterRabbitMQ.PaymentSyncConsumer)
		if err := consumer.HandleEvent(syncQueue.events[0]); err != nil {
			t.Fatalf("expected consumer reconciliation to succeed, got %v", err)
		}

		stored, err := paymentRepo.ReadPaymentById(res.Id.String())
		if err != nil || stored.Status != entitiesPayment.PaymentStatusApproved || stored.ExternalPaymentId == "" {
			t.Fatalf("expected stored payment APPROVED with external id, got %+v err=%v", stored, err)
		}
		if orderRepo.orders[orderId.String()].Status != entitiesOrder.OrderStatusPreparing {
			t.Fatalf("expected order PREPARING after consumer sync, got %s", orderRepo.orders[orderId.String()].Status)
		}
		if len(eventQueue.events) != 1 || eventQueue.events[0].EventType != ports.EventOrderPaymentApproved {
			t.Fatalf("expected ORDER_PAYMENT_APPROVED notification, got %+v", eventQueue.events)
		}
	})

	t.Run("Enqueues Pix gateway data in RabbitMQ fallback when DB fails after Pix creation and consumer reconciles", func(t *testing.T) {
		orderRepo := &stubOrderRepo{orders: make(map[string]*entitiesOrder.Order)}
		paymentRepo := newStubPaymentRepo(orderRepo)
		paymentRepo.updateGwErr = apperrors.ErrDatabaseUnavailable
		eventQueue := &stubEventQueue{}
		syncQueue := &stubPaymentSyncQueue{}
		gateway := adapterPayment.NewMockPaymentGateway()
		readOrder := useCasesOrder.NewReadOrderByIdUseCase(orderRepo)

		processUC := useCasesPayment.NewProcessPaymentUseCase(paymentRepo, readOrder, gateway, eventQueue, syncQueue)

		orderId := uuid.New()
		orderRepo.orders[orderId.String()] = &entitiesOrder.Order{
			Id:     orderId,
			User:   &entitiesUser.User{ID: userId, Name: "Diego", Email: "diego@example.com"},
			Status: entitiesOrder.OrderStatusPendingPayment,
			Items:  []entitiesOrderItem.OrderItem{{Quantity: 1, UnitPrice: 99.00}},
		}

		res, err := processUC.UseCase(useCasesPayment.ProcessPaymentInput{
			OrderId:        orderId.String(),
			UserId:         userId.String(),
			Role:           "user",
			IdempotencyKey: "idem-pix-fallback-1",
			Method:         entitiesPayment.PaymentMethodPix,
		})
		if err != nil {
			t.Fatalf("expected nil err with RabbitMQ fallback, got %v", err)
		}
		if res.ExternalPaymentId == "" || res.PixCopyPaste == "" {
			t.Fatalf("expected Pix response with external id and copy/paste, got %+v", res)
		}
		if len(syncQueue.events) != 1 || syncQueue.events[0].Action != ports.PaymentSyncActionUpdateGatewayData {
			t.Fatalf("expected 1 UPDATE_GATEWAY_DATA event, got %+v", syncQueue.events)
		}

		paymentRepo.updateGwErr = nil
		consumer := adapterRabbitMQ.NewPaymentSyncConsumer(nil, paymentRepo, eventQueue).(*adapterRabbitMQ.PaymentSyncConsumer)
		if err := consumer.HandleEvent(syncQueue.events[0]); err != nil {
			t.Fatalf("expected consumer reconciliation to succeed, got %v", err)
		}

		stored, err := paymentRepo.ReadPaymentById(res.Id.String())
		if err != nil || stored.ExternalPaymentId != res.ExternalPaymentId || stored.PixCopyPaste != res.PixCopyPaste {
			t.Fatalf("expected stored Pix data synced, got %+v err=%v", stored, err)
		}
	})
}
