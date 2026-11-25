package use_cases

import (
	"context"
	redisAdapter "e-commerce/adapters/redis"
	entitiesOrder "e-commerce/core/entities/order"
	entitiesPayment "e-commerce/core/entities/payment"
	"e-commerce/core/ports"
	useCasesOrder "e-commerce/core/use_cases/order"
	"log"
	"math"
	"sync"
	"time"
)

const gatewayStatusSyncCooldown = 15 * time.Second

type ReadPaymentByOrderUseCase struct {
	paymentRepo    entitiesPayment.PaymentRepository
	readOrderById  *useCasesOrder.ReadOrderByIdUseCase
	paymentGateway ports.PaymentGateway
	eventQueue     ports.EventOrderQueue
	lastGatewaySync sync.Map
}

func NewReadPaymentByOrderUseCase(
	paymentRepo entitiesPayment.PaymentRepository,
	readOrderById *useCasesOrder.ReadOrderByIdUseCase,
	paymentGateway ports.PaymentGateway,
	eventQueue ports.EventOrderQueue,
) *ReadPaymentByOrderUseCase {
	return &ReadPaymentByOrderUseCase{
		paymentRepo:    paymentRepo,
		readOrderById:  readOrderById,
		paymentGateway: paymentGateway,
		eventQueue:     eventQueue,
	}
}

// UseCase consulta o ultimo pagamento do pedido.
// Sincroniza status real no gateway (ex: Mercado Pago Pix pago) e verifica expiracao automatica.
func (uc *ReadPaymentByOrderUseCase) UseCase(orderId string, userId string, role string) (*entitiesPayment.Payment, error) {
	order, err := uc.readOrderById.UseCase(orderId, userId, role)
	if err != nil {
		return nil, err
	}

	payment, err := uc.paymentRepo.ReadPaymentByOrderId(order.Id.String())
	if err != nil {
		return nil, err
	}

	if order.Status == entitiesOrder.OrderStatusPendingPayment && payment.Status == entitiesPayment.PaymentStatusPending {
		// 1. Consulta status em tempo real no gateway com cooldown para evitar excesso de chamadas durante polling
		if uc.paymentGateway != nil && payment.ExternalPaymentId != "" && uc.shouldSyncGatewayStatus(payment.ExternalPaymentId) {
			if remoteStatus, gwErr := uc.paymentGateway.GetPaymentStatus(payment.ExternalPaymentId); gwErr == nil {
				switch remoteStatus {
				case entitiesPayment.PaymentStatusApproved:
					if math.Abs(order.Total()-payment.Amount) > 0.01 {
						log.Printf("[WARN] divergencia de valor ao sincronizar pagamento order=%s payment=%s orderTotal=%.2f paymentAmount=%.2f", order.Id, payment.Id, order.Total(), payment.Amount)
						break
					}
					now := time.Now()
					payment.Status = entitiesPayment.PaymentStatusApproved
					payment.PaidAt = &now
					if appErr := uc.paymentRepo.ApprovePaymentAndOrder(*payment, order.Id.String()); appErr == nil {
						uc.publishEvent(order, ports.EventOrderPaymentApproved, payment.Method)
						return uc.paymentRepo.ReadPaymentById(payment.Id.String())
					}
				case entitiesPayment.PaymentStatusFailed:
					payment.Status = entitiesPayment.PaymentStatusFailed
					if failErr := uc.paymentRepo.FailOrExpirePaymentAndCancelOrder(*payment, order.Id.String()); failErr == nil {
						uc.publishEvent(order, ports.EventOrderPaymentFailed, payment.Method)
						return uc.paymentRepo.ReadPaymentById(payment.Id.String())
					}
				}
			}
		}

		// 2. Verifica expiracao local
		if payment.IsExpired(time.Now()) {
			payment.Status = entitiesPayment.PaymentStatusExpired
			if expErr := uc.paymentRepo.FailOrExpirePaymentAndCancelOrder(*payment, order.Id.String()); expErr == nil {
				uc.publishEvent(order, ports.EventOrderPaymentFailed, payment.Method)
				return uc.paymentRepo.ReadPaymentById(payment.Id.String())
			}
		}
	}

	return payment, nil
}

func (uc *ReadPaymentByOrderUseCase) shouldSyncGatewayStatus(externalPaymentID string) bool {
	now := time.Now()
	if prev, ok := uc.lastGatewaySync.Load(externalPaymentID); ok {
		if lastTime, valid := prev.(time.Time); valid && now.Sub(lastTime) < gatewayStatusSyncCooldown {
			return false
		}
	}
	if !redisAdapter.AcquirePaymentGatewaySyncSlot(context.Background(), externalPaymentID, gatewayStatusSyncCooldown) {
		return false
	}
	uc.lastGatewaySync.Store(externalPaymentID, now)
	return true
}

func (uc *ReadPaymentByOrderUseCase) publishEvent(order *entitiesOrder.Order, eventType string, method string) {
	if uc.eventQueue == nil || order.User == nil {
		return
	}
	if pubErr := uc.eventQueue.PublishOrderNotification(ports.OrderNotificationEvent{
		EventType:     eventType,
		OrderId:       order.Id.String(),
		UserEmail:     order.User.Email,
		UserName:      order.User.Name,
		OrderTotal:    order.Total(),
		PaymentMethod: method,
	}); pubErr != nil {
		log.Printf("[ERROR] falha ao publicar evento %s order=%s: %v", eventType, order.Id, pubErr)
	}
}
