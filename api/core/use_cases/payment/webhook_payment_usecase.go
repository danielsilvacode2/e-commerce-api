package use_cases

import (
	"e-commerce/core/apperrors"
	entitiesOrder "e-commerce/core/entities/order"
	entitiesPayment "e-commerce/core/entities/payment"
	"e-commerce/core/ports"
	"log"
	"math"
	"strings"
	"time"
)

const (
	PaymentActionApprove = "APPROVE"
	PaymentActionFail    = "FAIL"
	PaymentActionExpire  = "EXPIRE"
)

type WebhookPaymentUseCase struct {
	paymentRepo    entitiesPayment.PaymentRepository
	orderRepo      entitiesOrder.OrderRepository
	paymentGateway ports.PaymentGateway
	eventQueue     ports.EventOrderQueue
}

func NewWebhookPaymentUseCase(
	paymentRepo entitiesPayment.PaymentRepository,
	orderRepo entitiesOrder.OrderRepository,
	paymentGateway ports.PaymentGateway,
	eventQueue ports.EventOrderQueue,
) *WebhookPaymentUseCase {
	return &WebhookPaymentUseCase{
		paymentRepo:    paymentRepo,
		orderRepo:      orderRepo,
		paymentGateway: paymentGateway,
		eventQueue:     eventQueue,
	}
}

// SyncByExternalId consulta o status real no gateway (usado pelo webhook nativo do Mercado Pago) e aplica de forma idempotente.
func (uc *WebhookPaymentUseCase) SyncByExternalId(externalPaymentId string) (*entitiesPayment.Payment, error) {
	if strings.TrimSpace(externalPaymentId) == "" || uc.paymentGateway == nil {
		return nil, apperrors.ErrInvalidInput
	}

	remoteStatus, err := uc.paymentGateway.GetPaymentStatus(externalPaymentId)
	if err != nil {
		return nil, err
	}

	switch remoteStatus {
	case entitiesPayment.PaymentStatusApproved:
		return uc.UseCase("", externalPaymentId, PaymentActionApprove)
	case entitiesPayment.PaymentStatusFailed:
		return uc.UseCase("", externalPaymentId, PaymentActionFail)
	default:
		return uc.paymentRepo.ReadPaymentByExternalId(externalPaymentId)
	}
}

// UseCase processa a confirmacao/recusa/expiracao de um pagamento de forma idempotente (webhook ou simulacao Pix).
func (uc *WebhookPaymentUseCase) UseCase(paymentId string, externalPaymentId string, action string) (*entitiesPayment.Payment, error) {
	act := strings.ToUpper(strings.TrimSpace(action))
	if act != PaymentActionApprove && act != PaymentActionFail && act != PaymentActionExpire {
		return nil, apperrors.ErrInvalidInput
	}

	var payment *entitiesPayment.Payment
	var err error

	if strings.TrimSpace(paymentId) != "" {
		payment, err = uc.paymentRepo.ReadPaymentById(paymentId)
	} else if strings.TrimSpace(externalPaymentId) != "" {
		payment, err = uc.paymentRepo.ReadPaymentByExternalId(externalPaymentId)
	} else {
		return nil, apperrors.ErrInvalidInput
	}

	if err != nil {
		return nil, err
	}

	// Idempotencia: se o pagamento ja saiu de PENDING, retorna o estado atual sem reprocessar
	if payment.Status != entitiesPayment.PaymentStatusPending {
		return payment, nil
	}

	order, err := uc.orderRepo.ReadOrderById(payment.OrderId.String())
	if err != nil {
		return nil, err
	}

	if order.Status != entitiesOrder.OrderStatusPendingPayment {
		return payment, nil
	}

	switch act {
	case PaymentActionApprove:
		if math.Abs(order.Total()-payment.Amount) > 0.01 {
			log.Printf("[WARN] divergencia de valor ao aprovar pagamento order=%s payment=%s orderTotal=%.2f paymentAmount=%.2f", order.Id, payment.Id, order.Total(), payment.Amount)
			return nil, apperrors.ErrConflict
		}
		now := time.Now()
		payment.Status = entitiesPayment.PaymentStatusApproved
		payment.PaidAt = &now
		if err := uc.paymentRepo.ApprovePaymentAndOrder(*payment, order.Id.String()); err != nil {
			return nil, err
		}
		uc.publishEvent(order, ports.EventOrderPaymentApproved, payment.Method)

	case PaymentActionFail:
		payment.Status = entitiesPayment.PaymentStatusFailed
		if err := uc.paymentRepo.FailOrExpirePaymentAndCancelOrder(*payment, order.Id.String()); err != nil {
			return nil, err
		}
		uc.publishEvent(order, ports.EventOrderPaymentFailed, payment.Method)

	case PaymentActionExpire:
		payment.Status = entitiesPayment.PaymentStatusExpired
		if err := uc.paymentRepo.FailOrExpirePaymentAndCancelOrder(*payment, order.Id.String()); err != nil {
			return nil, err
		}
		uc.publishEvent(order, ports.EventOrderPaymentFailed, payment.Method)
	}

	return uc.paymentRepo.ReadPaymentById(payment.Id.String())
}

func (uc *WebhookPaymentUseCase) publishEvent(order *entitiesOrder.Order, eventType string, method string) {
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
