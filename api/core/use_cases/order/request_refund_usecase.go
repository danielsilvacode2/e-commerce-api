package use_cases

import (
	"e-commerce/core/apperrors"
	entitiesOrder "e-commerce/core/entities/order"
	entitiesPayment "e-commerce/core/entities/payment"
	"e-commerce/core/ports"
	"log"
	"strings"
)

type RequestRefundUseCase struct {
	orderRepository   entitiesOrder.OrderRepository
	paymentRepository entitiesPayment.PaymentRepository
	readOrderById     *ReadOrderByIdUseCase
	paymentGateway    ports.PaymentGateway
	eventQueue        ports.EventOrderQueue
}

func NewRequestRefundUseCase(
	orderRepository entitiesOrder.OrderRepository,
	paymentRepository entitiesPayment.PaymentRepository,
	readOrderById *ReadOrderByIdUseCase,
	paymentGateway ports.PaymentGateway,
	eventQueue ports.EventOrderQueue,
) *RequestRefundUseCase {
	return &RequestRefundUseCase{
		orderRepository:   orderRepository,
		paymentRepository: paymentRepository,
		readOrderById:     readOrderById,
		paymentGateway:    paymentGateway,
		eventQueue:        eventQueue,
	}
}

// UseCase aplica as regras estritas de estorno do cliente:
// - PREPARING: estorno automatico imediato (sem aprovacao do admin), restaura estoque, atualiza pedido e pagamento para REFUNDED e notifica.
// - SHIPPED: BLOQUEADO (retorna ErrUnprocessableEntity).
// - DELIVERED: transiciona pedido para REFUND_REQUESTED com motivo obrigatorio, aguardando aprovacao do admin.
func (useCase *RequestRefundUseCase) UseCase(orderId string, reason string, userId string, role string) (*entitiesOrder.Order, error) {
	order, err := useCase.readOrderById.UseCase(orderId, userId, role)
	if err != nil {
		return nil, err
	}

	// Em transito (SHIPPED) ou qualquer outro status nao permitido -> bloqueado
	if !order.CanCustomerRequestRefund() {
		return nil, apperrors.ErrUnprocessableEntity
	}

	cleanReason := strings.TrimSpace(reason)

	// Regra 1: PREPARING -> estorno automatico imediato
	if order.IsAutoRefundable() {
		if cleanReason == "" {
			cleanReason = "Cancelamento e estorno solicitado pelo cliente durante a preparação"
		}

		// Executa estorno no gateway ANTES de confirmar no banco para evitar dessincronizacao financeira
		if useCase.paymentGateway != nil {
			if currentPayment, pErr := useCase.paymentRepository.ReadPaymentByOrderId(order.Id.String()); pErr == nil &&
				currentPayment != nil &&
				currentPayment.Status == entitiesPayment.PaymentStatusApproved &&
				currentPayment.ExternalPaymentId != "" {
				if gwErr := useCase.paymentGateway.RefundPayment(currentPayment.ExternalPaymentId, currentPayment.Amount); gwErr != nil {
					log.Printf("[ERROR] falha ao efetivar estorno automatico no gateway order=%s: %v", order.Id, gwErr)
					return nil, gwErr
				}
			}
		}

		if _, err := useCase.paymentRepository.RefundPaymentAndOrder(
			order.Id.String(),
			entitiesOrder.OrderStatusPreparing,
			true,
			&cleanReason,
		); err != nil {
			return nil, err
		}

		if useCase.eventQueue != nil && order.User != nil {
			if pubErr := useCase.eventQueue.PublishOrderNotification(ports.OrderNotificationEvent{
				EventType:    ports.EventOrderRefundApproved,
				OrderId:      order.Id.String(),
				UserEmail:    order.User.Email,
				UserName:     order.User.Name,
				OrderTotal:   order.Total(),
				RefundReason: cleanReason,
			}); pubErr != nil {
				log.Printf("[ERROR] falha ao publicar evento de estorno automatico order=%s: %v", order.Id, pubErr)
			}
		}

		return useCase.orderRepository.ReadOrderById(order.Id.String())
	}

	// Regra 3: DELIVERED -> solicita estorno para aprovacao manual do admin
	if cleanReason == "" {
		return nil, apperrors.ErrInvalidInput
	}

	if err := useCase.orderRepository.UpdateOrderStatusWithDetails(
		order.Id.String(),
		entitiesOrder.OrderStatusDelivered,
		entitiesOrder.OrderStatusRefundRequested,
		false,
		nil,
		&cleanReason,
		nil,
	); err != nil {
		return nil, err
	}

	return useCase.orderRepository.ReadOrderById(order.Id.String())
}
