package use_cases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"e-commerce/core/apperrors"
	entitiesAddress "e-commerce/core/entities/address"
	entitiesCart "e-commerce/core/entities/cart"
	entitiesCartProduct "e-commerce/core/entities/cart_product"
	entitiesOrder "e-commerce/core/entities/order"
	entitiesOrderItem "e-commerce/core/entities/order_item"
	entitiesPayment "e-commerce/core/entities/payment"
	entitiesProduct "e-commerce/core/entities/product"
	entitiesUser "e-commerce/core/entities/user"
	"e-commerce/core/ports"
	useCasesAddress "e-commerce/core/use_cases/address"
	useCasesCartProduct "e-commerce/core/use_cases/cart_product"
	useCasesOrder "e-commerce/core/use_cases/order"

	"github.com/google/uuid"
)

type fakeOrderRepository struct {
	orders           map[string]*entitiesOrder.Order
	lastUsedExisting bool
}

func newFakeOrderRepository() *fakeOrderRepository {
	return &fakeOrderRepository{
		orders: make(map[string]*entitiesOrder.Order),
	}
}

func (f *fakeOrderRepository) CreateOrder(order entitiesOrder.Order, cartProductIds []string, useExistingAddress bool) error {
	f.lastUsedExisting = useExistingAddress
	copyOrder := order
	f.orders[order.Id.String()] = &copyOrder
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

func (f *fakeOrderRepository) UpdateOrderAddress(address entitiesAddress.Address, ID string, useExistingAddress bool) error {
	order, ok := f.orders[ID]
	if !ok {
		return apperrors.ErrNotFound
	}
	f.lastUsedExisting = useExistingAddress
	copyAddr := address
	order.Address = &copyAddr
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
func (f *fakePaymentRepoForOrder) UpdatePaymentGatewayData(p entitiesPayment.Payment) error {
	return nil
}
func (f *fakePaymentRepoForOrder) DeletePayment(id string) error {
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
	pendingOrderId := uuid.New()
	repo.orders[pendingOrderId.String()] = &entitiesOrder.Order{
		Id:     pendingOrderId,
		User:   &entitiesUser.User{ID: userId},
		Status: entitiesOrder.OrderStatusPendingPayment,
	}

	preparingOrderId := uuid.New()
	repo.orders[preparingOrderId.String()] = &entitiesOrder.Order{
		Id:     preparingOrderId,
		User:   &entitiesUser.User{ID: userId},
		Status: entitiesOrder.OrderStatusPreparing,
	}

	t.Run("User cannot cancel paid order in PREPARING without refund flow", func(t *testing.T) {
		err := cancelUseCase.UseCase(preparingOrderId.String(), userId.String(), "user")
		if !errors.Is(err, apperrors.ErrUnprocessableEntity) {
			t.Fatalf("expected ErrUnprocessableEntity, got %v", err)
		}
	})

	t.Run("User can cancel unpaid order in PENDING_PAYMENT", func(t *testing.T) {
		err := cancelUseCase.UseCase(pendingOrderId.String(), userId.String(), "user")
		if err != nil {
			t.Fatalf("expected nil err, got %v", err)
		}
		if repo.orders[pendingOrderId.String()].Status != entitiesOrder.OrderStatusCanceled {
			t.Fatalf("expected status CANCELED, got %s", repo.orders[pendingOrderId.String()].Status)
		}
	})

	t.Run("Cannot cancel already canceled order", func(t *testing.T) {
		err := cancelUseCase.UseCase(pendingOrderId.String(), userId.String(), "user")
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

type fakeCartProductRepoForOrder struct {
	items []entitiesCartProduct.CartProduct
}

func (f *fakeCartProductRepoForOrder) AddProductCart(entitiesCartProduct.CartProduct) error { return nil }
func (f *fakeCartProductRepoForOrder) ReadAllProductsCart(string) ([]entitiesCartProduct.CartProduct, error) {
	return f.items, nil
}
func (f *fakeCartProductRepoForOrder) ReadProductsCart(string) (*entitiesCartProduct.CartProduct, error) {
	return nil, apperrors.ErrNotFound
}
func (f *fakeCartProductRepoForOrder) RemoveProductCart(string) error         { return nil }
func (f *fakeCartProductRepoForOrder) UpdateIsSelect(bool, string) error      { return nil }
func (f *fakeCartProductRepoForOrder) UpdateAllIsSelect(bool, string) error   { return nil }
func (f *fakeCartProductRepoForOrder) DecreaseProductCart(string, int64) error { return nil }

type fakeAddressGatewayForOrder struct {
	cepResult *ports.CepLookupResult
	cepErr    error
	cities    []string
	citiesErr error
}

func (f *fakeAddressGatewayForOrder) LookupByCep(ctx context.Context, cep string) (*ports.CepLookupResult, error) {
	if f.cepErr != nil {
		return nil, f.cepErr
	}
	return f.cepResult, nil
}

func (f *fakeAddressGatewayForOrder) ListCitiesByState(ctx context.Context, uf string) ([]string, error) {
	if f.citiesErr != nil {
		return nil, f.citiesErr
	}
	return f.cities, nil
}

type fakeAddressRepoForOrder struct {
	addresses map[string]*entitiesAddress.Address
}

func newFakeAddressRepoForOrder() *fakeAddressRepoForOrder {
	return &fakeAddressRepoForOrder{
		addresses: make(map[string]*entitiesAddress.Address),
	}
}

func (f *fakeAddressRepoForOrder) CreateAddress(address entitiesAddress.Address) error {
	copyAddr := address
	f.addresses[address.Id.String()] = &copyAddr
	return nil
}

func (f *fakeAddressRepoForOrder) ReadAddressById(ID string) (*entitiesAddress.Address, error) {
	if addr, ok := f.addresses[ID]; ok {
		copyAddr := *addr
		return &copyAddr, nil
	}
	return nil, apperrors.ErrNotFound
}

func (f *fakeAddressRepoForOrder) ReadAddressesByUserId(userId string) ([]entitiesAddress.Address, error) {
	var list []entitiesAddress.Address
	for _, a := range f.addresses {
		if a.UserId.String() == userId && a.IsComplete() {
			list = append(list, *a)
		}
	}
	return list, nil
}

func (f *fakeAddressRepoForOrder) FindMatchingUserAddress(userId string, address entitiesAddress.Address) (*entitiesAddress.Address, error) {
	for _, a := range f.addresses {
		if a.UserId.String() == userId &&
			a.Cep == address.Cep &&
			a.Street == address.Street &&
			a.Number == address.Number &&
			a.Complement == address.Complement &&
			a.Neighborhood == address.Neighborhood &&
			a.City == address.City &&
			a.State == address.State {
			copyAddr := *a
			return &copyAddr, nil
		}
	}
	return nil, apperrors.ErrNotFound
}

func TestCreateAndUpdateOrderAddressUseCases(t *testing.T) {
	userID := uuid.New()
	productID := uuid.New()
	cartItems := []entitiesCartProduct.CartProduct{
		{
			Id:       uuid.New(),
			Quantity: 2,
			IsSelect: true,
			Cart: &entitiesCart.Cart{
				Id:   uuid.New(),
				User: &entitiesUser.User{ID: userID, Name: "Maria", Email: "maria@example.com"},
			},
			Product: &entitiesProduct.Product{
				Id:    productID,
				Name:  "Camiseta Oversized",
				Price: 99.90,
			},
		},
	}

	t.Run("CreateOrder persiste nova instancia de Address validada junto com Order", func(t *testing.T) {
		orderRepo := newFakeOrderRepository()
		addrRepo := newFakeAddressRepoForOrder()
		paymentRepo := &fakePaymentRepoForOrder{orderRepo: orderRepo, payments: make(map[string]*entitiesPayment.Payment)}
		cartUC := useCasesCartProduct.NewReadAllCartProductsUseCase(&fakeCartProductRepoForOrder{items: cartItems})
		validateUC := useCasesAddress.NewValidateAddressUseCase(&fakeAddressGatewayForOrder{
			cepResult: &ports.CepLookupResult{
				Cep:          "01001000",
				State:        "SP",
				City:         "São Paulo",
				Neighborhood: "Sé",
				Street:       "Praça da Sé",
			},
			cities: []string{"SÃO PAULO", "CAMPINAS"},
		})

		createUC := useCasesOrder.NewCreateOrderUseCase(orderRepo, addrRepo, paymentRepo, nil, cartUC, validateUC)
		res, err := createUC.UseCase(context.Background(), useCasesOrder.OrderAddressSelection{
			Address: &entitiesAddress.Address{
				Cep:          "01001-000",
				Street:       "Praça da Sé",
				Number:       "100A",
				Complement:   "Apto 12",
				Neighborhood: "Sé",
				City:         "Sao Paulo",
				State:        "sp",
			},
		}, userID.String())
		if err != nil {
			t.Fatalf("expected nil err, got %v", err)
		}
		if res.Warning != "" {
			t.Fatalf("expected empty warning, got %q", res.Warning)
		}
		if orderRepo.lastUsedExisting {
			t.Fatalf("expected new address instance, got useExistingAddress=true")
		}
		if res.Order.Address == nil || res.Order.Address.Id == uuid.Nil || res.Order.Address.UserId != userID {
			t.Fatalf("expected persisted Address with UUID and UserId, got %+v", res.Order.Address)
		}
		if res.Order.Address.Cep != "01001000" || res.Order.Address.State != "SP" || res.Order.Address.Number != "100A" {
			t.Fatalf("unexpected address stored: %+v", res.Order.Address)
		}
	})

	t.Run("CreateOrder reutiliza endereco salvo proprio via address_id sem chamar BrasilAPI", func(t *testing.T) {
		orderRepo := newFakeOrderRepository()
		addrRepo := newFakeAddressRepoForOrder()
		savedID := uuid.New()
		addrRepo.addresses[savedID.String()] = &entitiesAddress.Address{
			Id:           savedID,
			UserId:       userID,
			Cep:          "01001000",
			Street:       "Praça da Sé",
			Number:       "100",
			Neighborhood: "Sé",
			City:         "São Paulo",
			State:        "SP",
		}
		cartUC := useCasesCartProduct.NewReadAllCartProductsUseCase(&fakeCartProductRepoForOrder{items: cartItems})

		// validateAddress nil garante que nao precisa consultar BrasilAPI ao reutilizar endereco salvo
		createUC := useCasesOrder.NewCreateOrderUseCase(orderRepo, addrRepo, nil, nil, cartUC, nil)
		res, err := createUC.UseCase(context.Background(), useCasesOrder.OrderAddressSelection{
			AddressId: savedID.String(),
		}, userID.String())
		if err != nil {
			t.Fatalf("expected nil err reusing saved address, got %v", err)
		}
		if !orderRepo.lastUsedExisting {
			t.Fatal("expected useExistingAddress=true")
		}
		if res.Order.Address.Id != savedID {
			t.Fatalf("expected address ID %s, got %s", savedID, res.Order.Address.Id)
		}
	})

	t.Run("CreateOrder bloqueia reutilizacao de endereco salvo de outro usuario com 403", func(t *testing.T) {
		orderRepo := newFakeOrderRepository()
		addrRepo := newFakeAddressRepoForOrder()
		otherUserID := uuid.New()
		savedID := uuid.New()
		addrRepo.addresses[savedID.String()] = &entitiesAddress.Address{
			Id:     savedID,
			UserId: otherUserID,
			Cep:    "01001000",
			Street: "Praça da Sé",
			Number: "100",
			City:   "São Paulo",
			State:  "SP",
		}
		cartUC := useCasesCartProduct.NewReadAllCartProductsUseCase(&fakeCartProductRepoForOrder{items: cartItems})
		createUC := useCasesOrder.NewCreateOrderUseCase(orderRepo, addrRepo, nil, nil, cartUC, nil)

		_, err := createUC.UseCase(context.Background(), useCasesOrder.OrderAddressSelection{
			AddressId: savedID.String(),
		}, userID.String())
		if !errors.Is(err, apperrors.ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("CreateOrder rejeita reutilizacao de endereco legado incompleto com 422", func(t *testing.T) {
		orderRepo := newFakeOrderRepository()
		addrRepo := newFakeAddressRepoForOrder()
		legacyID := uuid.New()
		addrRepo.addresses[legacyID.String()] = &entitiesAddress.Address{
			Id:     legacyID,
			UserId: userID,
			Cep:    "01001000",
		}
		cartUC := useCasesCartProduct.NewReadAllCartProductsUseCase(&fakeCartProductRepoForOrder{items: cartItems})
		createUC := useCasesOrder.NewCreateOrderUseCase(orderRepo, addrRepo, nil, nil, cartUC, nil)

		_, err := createUC.UseCase(context.Background(), useCasesOrder.OrderAddressSelection{
			AddressId: legacyID.String(),
		}, userID.String())
		if !errors.Is(err, apperrors.ErrUnprocessableEntity) {
			t.Fatalf("expected ErrUnprocessableEntity for incomplete legacy address, got %v", err)
		}
	})

	t.Run("CreateOrder rejeita endereco com cidade divergente do CEP sem criar pedido", func(t *testing.T) {
		orderRepo := newFakeOrderRepository()
		addrRepo := newFakeAddressRepoForOrder()
		cartUC := useCasesCartProduct.NewReadAllCartProductsUseCase(&fakeCartProductRepoForOrder{items: cartItems})
		validateUC := useCasesAddress.NewValidateAddressUseCase(&fakeAddressGatewayForOrder{
			cepResult: &ports.CepLookupResult{
				Cep:   "01001000",
				State: "SP",
				City:  "São Paulo",
			},
			cities: []string{"SÃO PAULO", "CAMPINAS"},
		})

		createUC := useCasesOrder.NewCreateOrderUseCase(orderRepo, addrRepo, nil, nil, cartUC, validateUC)
		_, err := createUC.UseCase(context.Background(), useCasesOrder.OrderAddressSelection{
			Address: &entitiesAddress.Address{
				Cep:    "01001-000",
				Street: "Rua Principal",
				Number: "50",
				City:   "Campinas",
				State:  "SP",
			},
		}, userID.String())
		if !errors.Is(err, apperrors.ErrUnprocessableEntity) {
			t.Fatalf("expected ErrUnprocessableEntity, got %v", err)
		}
		if len(orderRepo.orders) != 0 {
			t.Fatalf("order should not be persisted on invalid address")
		}
	})

	t.Run("CreateOrder aceita com warning em fail-open quando BrasilAPI esta indisponivel", func(t *testing.T) {
		orderRepo := newFakeOrderRepository()
		addrRepo := newFakeAddressRepoForOrder()
		cartUC := useCasesCartProduct.NewReadAllCartProductsUseCase(&fakeCartProductRepoForOrder{items: cartItems})
		validateUC := useCasesAddress.NewValidateAddressUseCase(&fakeAddressGatewayForOrder{
			cepErr:    apperrors.ErrAddressLookupUnavailable,
			citiesErr: apperrors.ErrAddressLookupUnavailable,
		})

		createUC := useCasesOrder.NewCreateOrderUseCase(orderRepo, addrRepo, nil, nil, cartUC, validateUC)
		res, err := createUC.UseCase(context.Background(), useCasesOrder.OrderAddressSelection{
			Address: &entitiesAddress.Address{
				Cep:    "01001-000",
				Street: "Praça da Sé",
				Number: "S/N",
				City:   "São Paulo",
				State:  "SP",
			},
		}, userID.String())
		if err != nil {
			t.Fatalf("expected nil err on fail-open, got %v", err)
		}
		if res.Warning == "" {
			t.Fatal("expected degraded validation warning")
		}
		if len(orderRepo.orders) != 1 {
			t.Fatalf("expected 1 persisted order, got %d", len(orderRepo.orders))
		}
	})

	t.Run("UpdateOrder atualiza endereco de pedido editavel criando nova instancia e bloqueia em SHIPPED", func(t *testing.T) {
		orderRepo := newFakeOrderRepository()
		addrRepo := newFakeAddressRepoForOrder()
		readUC := useCasesOrder.NewReadOrderByIdUseCase(orderRepo)
		validateUC := useCasesAddress.NewValidateAddressUseCase(&fakeAddressGatewayForOrder{
			cepResult: &ports.CepLookupResult{
				Cep:   "20040020",
				State: "RJ",
				City:  "Rio de Janeiro",
			},
			cities: []string{"RIO DE JANEIRO"},
		})
		updateUC := useCasesOrder.NewUpdateOrderUseCase(orderRepo, addrRepo, readUC, validateUC)

		orderID := uuid.New()
		oldAddrID := uuid.New()
		orderRepo.orders[orderID.String()] = &entitiesOrder.Order{
			Id:      orderID,
			User:    &entitiesUser.User{ID: userID},
			Status:  entitiesOrder.OrderStatusPreparing,
			Address: &entitiesAddress.Address{Id: oldAddrID, UserId: userID, Cep: "01001000"},
		}

		_, err := updateUC.UseCase(context.Background(), useCasesOrder.OrderAddressSelection{
			Address: &entitiesAddress.Address{
				Cep:    "20040-020",
				Street: "Rua da Assembleia",
				Number: "10",
				City:   "Rio de Janeiro",
				State:  "RJ",
			},
		}, orderID.String(), userID.String(), "user")
		if err != nil {
			t.Fatalf("expected nil err updating editable order, got %v", err)
		}
		if orderRepo.orders[orderID.String()].Address.Cep != "20040020" || orderRepo.orders[orderID.String()].Address.Id == oldAddrID {
			t.Fatalf("expected new address instance with cep 20040020, got %+v", orderRepo.orders[orderID.String()].Address)
		}

		orderRepo.orders[orderID.String()].Status = entitiesOrder.OrderStatusShipped
		_, err = updateUC.UseCase(context.Background(), useCasesOrder.OrderAddressSelection{
			Address: &entitiesAddress.Address{
				Cep:    "20040-020",
				Street: "Rua da Assembleia",
				Number: "10",
				City:   "Rio de Janeiro",
				State:  "RJ",
			},
		}, orderID.String(), userID.String(), "user")
		if !errors.Is(err, apperrors.ErrUnprocessableEntity) {
			t.Fatalf("expected ErrUnprocessableEntity for SHIPPED order, got %v", err)
		}
	})
}
