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
	paymentRepo    entitiesPayment.PaymentRepository
	readOrderById  *useCasesOrder.ReadOrderByIdUseCase
	paymentGateway ports.PaymentGateway
	eventQueue     ports.EventOrderQueue
	inFlightKeys   sync.Map
}

func NewProcessPaymentUseCase(
	paymentRepo entitiesPayment.PaymentRepository,
	readOrderById *useCasesOrder.ReadOrderByIdUseCase,
	paymentGateway ports.PaymentGateway,
	eventQueue ports.EventOrderQueue,
) *ProcessPaymentUseCase {
	return &ProcessPaymentUseCase{
		paymentRepo:    paymentRepo,
		readOrderById:  readOrderById,
		paymentGateway: paymentGateway,
		eventQueue:     eventQueue,
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

	// Protecao contra requisicoes simultaneas com a mesma Idempotency-Key (ex: clique duplo)
	if _, loaded := uc.inFlightKeys.LoadOrStore(key, struct{}{}); loaded {
		// Se ja existe no banco, retorna o resultado original; caso contrario retorna conflito de processamento em andamento
		if existing, err := uc.paymentRepo.ReadPaymentByIdempotencyKey(key); err == nil {
			return existing, nil
		}
		return nil, apperrors.ErrConflict
	}
	defer uc.inFlightKeys.Delete(key)

	// 1. Verifica idempotencia no banco de dados
	if existing, err := uc.paymentRepo.ReadPaymentByIdempotencyKey(key); err == nil {
		if existing.OrderId.String() != input.OrderId {
			return nil, apperrors.ErrConflict
		}
		return existing, nil
	} else if !errors.Is(err, apperrors.ErrNotFound) {
		return nil, err
	}

	// 2. Valida acesso e estado do pedido
	order, err := uc.readOrderById.UseCase(input.OrderId, input.UserId, input.Role)
	if err != nil {
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

	// 3. Processa de acordo com o metodo escolhido
	if method == entitiesPayment.PaymentMethodPix {
		res, err := uc.paymentGateway.CreatePixPayment(order.Id.String(), order.Total(), payerEmail, key)
		if err != nil {
			return nil, err
		}

		payment := entitiesPayment.Payment{
			Id:                uuid.New(),
			OrderId:           order.Id,
			Provider:          entitiesPayment.PaymentProviderDefault,
			Method:            entitiesPayment.PaymentMethodPix,
			ExternalPaymentId: res.ExternalPaymentId,
			IdempotencyKey:    key,
			Amount:            order.Total(),
			Installments:      1,
			Status:            entitiesPayment.PaymentStatusPending,
			PixQrCode:         res.PixQrCode,
			PixCopyPaste:      res.PixCopyPaste,
			ExpiresAt:         &res.ExpiresAt,
		}

		if err := uc.paymentRepo.CreatePayment(payment); err != nil {
			if errors.Is(err, apperrors.ErrConflict) {
				return uc.paymentRepo.ReadPaymentByIdempotencyKey(key)
			}
			return nil, err
		}

		return uc.paymentRepo.ReadPaymentById(payment.Id.String())
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
		return nil, err
	}

	payment := entitiesPayment.Payment{
		Id:                uuid.New(),
		OrderId:           order.Id,
		Provider:          entitiesPayment.PaymentProviderDefault,
		Method:            entitiesPayment.PaymentMethodCreditCard,
		ExternalPaymentId: cardRes.ExternalPaymentId,
		IdempotencyKey:    key,
		Amount:            order.Total(),
		Installments:      installments,
		Status:            entitiesPayment.PaymentStatusPending,
		CardLast4:         cardRes.CardLast4,
		CardBrand:         cardRes.CardBrand,
	}

	if err := uc.paymentRepo.CreatePayment(payment); err != nil {
		if errors.Is(err, apperrors.ErrConflict) {
			return uc.paymentRepo.ReadPaymentByIdempotencyKey(key)
		}
		return nil, err
	}

	if cardRes.Approved {
		now := time.Now()
		payment.Status = entitiesPayment.PaymentStatusApproved
		payment.PaidAt = &now
		if err := uc.paymentRepo.ApprovePaymentAndOrder(payment, order.Id.String()); err != nil {
			return nil, err
		}
		uc.publishEvent(order, ports.EventOrderPaymentApproved, entitiesPayment.PaymentMethodCreditCard)
	} else {
		payment.Status = entitiesPayment.PaymentStatusFailed
		if err := uc.paymentRepo.FailOrExpirePaymentAndCancelOrder(payment, order.Id.String()); err != nil {
			return nil, err
		}
		uc.publishEvent(order, ports.EventOrderPaymentFailed, entitiesPayment.PaymentMethodCreditCard)
	}

	return uc.paymentRepo.ReadPaymentById(payment.Id.String())
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
