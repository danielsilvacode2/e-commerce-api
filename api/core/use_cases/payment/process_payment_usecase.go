package use_cases

import (
	"e-commerce/core/apperrors"
	entitiesOrder "e-commerce/core/entities/order"
	entitiesPayment "e-commerce/core/entities/payment"
	"e-commerce/core/ports"
	useCasesOrder "e-commerce/core/use_cases/order"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ProcessPaymentInput struct {
	OrderId         string
	UserId          string
	Role            string
	IdempotencyKey  string
	Method          string
	Installments    uint8
	PayerDocType    string
	PayerDocNumber  string
	CardToken       string
	PaymentMethodId string
	CardNumber      string
	CardHolderName  string
	CardExpMonth    int
	CardExpYear     int
	CardCVV         string
}

type ProcessPaymentUseCase struct {
	paymentRepo      entitiesPayment.PaymentRepository
	readOrderById    *useCasesOrder.ReadOrderByIdUseCase
	paymentGateway   ports.PaymentGateway
	eventQueue       ports.EventOrderQueue
	paymentSyncQueue ports.EventPaymentSyncQueue
	inFlightKeys     sync.Map
}

func NewProcessPaymentUseCase(
	paymentRepo entitiesPayment.PaymentRepository,
	readOrderById *useCasesOrder.ReadOrderByIdUseCase,
	paymentGateway ports.PaymentGateway,
	eventQueue ports.EventOrderQueue,
	paymentSyncQueue ...ports.EventPaymentSyncQueue,
) *ProcessPaymentUseCase {
	var syncQ ports.EventPaymentSyncQueue
	if len(paymentSyncQueue) > 0 {
		syncQ = paymentSyncQueue[0]
	}
	return &ProcessPaymentUseCase{
		paymentRepo:      paymentRepo,
		readOrderById:    readOrderById,
		paymentGateway:   paymentGateway,
		eventQueue:       eventQueue,
		paymentSyncQueue: syncQ,
	}
}

// UseCase processa um pagamento (PIX ou CREDIT_CARD) de forma estritamente idempotente via Idempotency-Key.
func (uc *ProcessPaymentUseCase) UseCase(input ProcessPaymentInput) (*entitiesPayment.Payment, error) {
	key := strings.TrimSpace(input.IdempotencyKey)
	if key == "" {
		return nil, apperrors.ErrInvalidInput
	}

	method := strings.ToUpper(strings.TrimSpace(input.Method))
	if !entitiesPayment.IsValidPaymentMethod(method) {
		return nil, apperrors.ErrInvalidInput
	}

	// 1. Valida acesso e propriedade do pedido ANTES de consultar qualquer pagamento por Idempotency-Key (previne IDOR)
	order, err := uc.readOrderById.UseCase(input.OrderId, input.UserId, input.Role)
	if err != nil {
		return nil, err
	}

	// Protecao contra requisicoes simultaneas com a mesma Idempotency-Key (ex: clique duplo)
	if _, loaded := uc.inFlightKeys.LoadOrStore(key, struct{}{}); loaded {
		// Se ja existe no banco para este mesmo pedido, retorna o resultado original; caso contrario retorna conflito
		if existing, err := uc.paymentRepo.ReadPaymentByIdempotencyKey(key); err == nil {
			if existing.OrderId.String() != order.Id.String() {
				return nil, apperrors.ErrConflict
			}
			return existing, nil
		}
		return nil, apperrors.ErrConflict
	}
	defer uc.inFlightKeys.Delete(key)

	// 2. Verifica idempotencia no banco de dados
	if existing, err := uc.paymentRepo.ReadPaymentByIdempotencyKey(key); err == nil {
		if existing.OrderId.String() != order.Id.String() {
			return nil, apperrors.ErrConflict
		}
		return existing, nil
	} else if !errors.Is(err, apperrors.ErrNotFound) {
		return nil, err
	}

	if order.Status != entitiesOrder.OrderStatusPendingPayment {
		return nil, apperrors.ErrUnprocessableEntity
	}

	installments := input.Installments
	if installments < 1 {
		installments = 1
	}

	payerEmail := ""
	if order.User != nil {
		payerEmail = order.User.Email
	}

	// 3. Pre-persiste o pagamento como PENDING ANTES de chamar o gateway.
	// Se o banco ja estiver indisponivel, falha antes de cobrar o cliente.
	// Tambem garante idempotencia no banco entre multiplas instancias da API.
	paymentInstallments := installments
	if method == entitiesPayment.PaymentMethodPix {
		paymentInstallments = 1
	}

	payment := entitiesPayment.Payment{
		Id:             uuid.New(),
		OrderId:        order.Id,
		Provider:       entitiesPayment.PaymentProviderDefault,
		Method:         method,
		IdempotencyKey: key,
		Amount:         order.Total(),
		Installments:   paymentInstallments,
		Status:         entitiesPayment.PaymentStatusPending,
	}

	if err := uc.paymentRepo.CreatePayment(payment); err != nil {
		if errors.Is(err, apperrors.ErrConflict) {
			existing, readErr := uc.paymentRepo.ReadPaymentByIdempotencyKey(key)
			if readErr != nil {
				return nil, readErr
			}
			if existing.OrderId.String() != order.Id.String() {
				return nil, apperrors.ErrConflict
			}
			return existing, nil
		}
		return nil, err
	}

	// 4. Processa no gateway de acordo com o metodo escolhido
	if method == entitiesPayment.PaymentMethodPix {
		res, err := uc.paymentGateway.CreatePixPayment(order.Id.String(), order.Total(), payerEmail, key)
		if err != nil {
			if delErr := uc.paymentRepo.DeletePayment(payment.Id.String()); delErr != nil {
				log.Printf("[WARN] falha ao remover pagamento Pix pendente orfao id=%s: %v", payment.Id, delErr)
			}
			return nil, err
		}

		payment.ExternalPaymentId = res.ExternalPaymentId
		payment.PixQrCode = res.PixQrCode
		payment.PixCopyPaste = res.PixCopyPaste
		payment.ExpiresAt = &res.ExpiresAt

		if err := uc.paymentRepo.UpdatePaymentGatewayData(payment); err != nil {
			if uc.enqueuePaymentSync(ports.PaymentSyncActionUpdateGatewayData, payment, order) {
				return &payment, nil
			}
			return nil, err
		}

		persisted, err := uc.paymentRepo.ReadPaymentById(payment.Id.String())
		if err != nil {
			return &payment, nil
		}
		return persisted, nil
	}

	// CREDIT_CARD
	cardRes, err := uc.paymentGateway.ProcessCreditCardPayment(ports.CreditCardGatewayInput{
		OrderId:         order.Id.String(),
		Amount:          order.Total(),
		Installments:    installments,
		PayerEmail:      payerEmail,
		PayerDocType:    input.PayerDocType,
		PayerDocNumber:  input.PayerDocNumber,
		CardToken:       input.CardToken,
		PaymentMethodId: input.PaymentMethodId,
		CardNumber:      input.CardNumber,
		CardHolderName:  input.CardHolderName,
		CardExpMonth:    input.CardExpMonth,
		CardExpYear:     input.CardExpYear,
		CardCVV:         input.CardCVV,
		IdempotencyKey:  key,
	})
	if err != nil {
		if delErr := uc.paymentRepo.DeletePayment(payment.Id.String()); delErr != nil {
			log.Printf("[WARN] falha ao remover pagamento Cartao pendente orfao id=%s: %v", payment.Id, delErr)
		}
		return nil, err
	}

	payment.ExternalPaymentId = cardRes.ExternalPaymentId
	payment.CardLast4 = cardRes.CardLast4
	payment.CardBrand = cardRes.CardBrand

	if cardRes.Approved {
		now := time.Now()
		payment.Status = entitiesPayment.PaymentStatusApproved
		payment.PaidAt = &now
		if err := uc.paymentRepo.ApprovePaymentAndOrder(payment, order.Id.String()); err != nil {
			if uc.enqueuePaymentSync(ports.PaymentSyncActionApprove, payment, order) {
				return &payment, nil
			}
			return nil, err
		}
		uc.publishEvent(order, ports.EventOrderPaymentApproved, entitiesPayment.PaymentMethodCreditCard)
	} else {
		payment.Status = entitiesPayment.PaymentStatusFailed
		if err := uc.paymentRepo.FailOrExpirePaymentAndCancelOrder(payment, order.Id.String()); err != nil {
			if uc.enqueuePaymentSync(ports.PaymentSyncActionFail, payment, order) {
				return &payment, nil
			}
			return nil, err
		}
		uc.publishEvent(order, ports.EventOrderPaymentFailed, entitiesPayment.PaymentMethodCreditCard)
	}

	persisted, err := uc.paymentRepo.ReadPaymentById(payment.Id.String())
	if err != nil {
		return &payment, nil
	}
	return persisted, nil
}

func (uc *ProcessPaymentUseCase) enqueuePaymentSync(action string, payment entitiesPayment.Payment, order *entitiesOrder.Order) bool {
	if uc.paymentSyncQueue == nil {
		return false
	}

	event := ports.PaymentSyncEvent{
		Action:            action,
		PaymentId:         payment.Id.String(),
		OrderId:           order.Id.String(),
		Provider:          payment.Provider,
		Method:            payment.Method,
		ExternalPaymentId: payment.ExternalPaymentId,
		IdempotencyKey:    payment.IdempotencyKey,
		Amount:            payment.Amount,
		Installments:      payment.Installments,
		Status:            payment.Status,
		PixQrCode:         payment.PixQrCode,
		PixCopyPaste:      payment.PixCopyPaste,
		CardLast4:         payment.CardLast4,
		CardBrand:         payment.CardBrand,
		ExpiresAt:         payment.ExpiresAt,
		PaidAt:            payment.PaidAt,
	}
	if order.User != nil {
		event.UserEmail = order.User.Email
		event.UserName = order.User.Name
	}

	if err := uc.paymentSyncQueue.PublishPaymentSync(event); err != nil {
		log.Printf("[ERROR] falha ao enfileirar fallback payment_sync (%s) para payment=%s order=%s: %v", action, payment.Id, order.Id, err)
		return false
	}

	log.Printf("[WARN] banco indisponivel apos gateway; pagamento enfileirado em payment_sync (%s) payment=%s order=%s", action, payment.Id, order.Id)
	return true
}

func (uc *ProcessPaymentUseCase) publishEvent(order *entitiesOrder.Order, eventType string, method string) {
	if uc.eventQueue == nil || order.User == nil {
		return
	}
	if err := uc.eventQueue.PublishOrderNotification(ports.OrderNotificationEvent{
		EventType:     eventType,
		OrderId:       order.Id.String(),
		UserEmail:     order.User.Email,
		UserName:      order.User.Name,
		OrderTotal:    order.Total(),
		PaymentMethod: method,
	}); err != nil {
		log.Printf("[ERROR] falha ao publicar evento %s para order=%s: %v", eventType, order.Id, err)
	}
}
