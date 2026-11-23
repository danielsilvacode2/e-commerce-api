package use_cases

import (
	"e-commerce/core/apperrors"
	entitiesOrder "e-commerce/core/entities/order"
	entitiesPayment "e-commerce/core/entities/payment"
	"e-commerce/core/ports"
	"log"
)

type ApproveRefundUseCase struct {
	orderRepository   entitiesOrder.OrderRepository
	paymentRepository entitiesPayment.PaymentRepository
	readOrderById     *ReadOrderByIdUseCase
	paymentGateway    ports.PaymentGateway
	eventQueue        ports.EventOrderQueue
}

func NewApproveRefundUseCase(
	orderRepository entitiesOrder.OrderRepository,
	paymentRepository entitiesPayment.PaymentRepository,
	readOrderById *ReadOrderByIdUseCase,
	paymentGateway ports.PaymentGateway,
	eventQueue ports.EventOrderQueue,
) *ApproveRefundUseCase {
	return &ApproveRefundUseCase{
		orderRepository:   orderRepository,
		paymentRepository: paymentRepository,
		readOrderById:     readOrderById,
		paymentGateway:    paymentGateway,
		eventQueue:        eventQueue,
	}
}

// UseCase aprova uma solicitacao de estorno (REFUND_REQUESTED -> REFUNDED), atualiza pagamento e notifica o cliente.
func (useCase *ApproveRefundUseCase) UseCase(orderId string, adminUserId string, role string) (*entitiesOrder.Order, error) {
	if role != roleAdmin {
		return nil, apperrors.ErrForbidden
	}

	order, err := useCase.readOrderById.UseCase(orderId, adminUserId, role)
	if err != nil {
		return nil, err
	}

	if order.Status != entitiesOrder.OrderStatusRefundRequested {
		return nil, apperrors.ErrUnprocessableEntity
	}

	// Executa estorno no gateway ANTES de confirmar no banco para evitar dessincronizacao financeira
	if useCase.paymentGateway != nil {
		if currentPayment, pErr := useCase.paymentRepository.ReadPaymentByOrderId(order.Id.String()); pErr == nil &&
			currentPayment != nil &&
			currentPayment.Status == entitiesPayment.PaymentStatusApproved &&
			currentPayment.ExternalPaymentId != "" {
			if gwErr := useCase.paymentGateway.RefundPayment(currentPayment.ExternalPaymentId, currentPayment.Amount); gwErr != nil {
				log.Printf("[ERROR] falha ao efetivar aprovacao de estorno no gateway order=%s: %v", order.Id, gwErr)
				return nil, gwErr
			}
		}
	}

	if _, err := useCase.paymentRepository.RefundPaymentAndOrder(
		order.Id.String(),
		entitiesOrder.OrderStatusRefundRequested,
		false,
		nil,
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
			RefundReason: order.RefundReason,
		}); pubErr != nil {
			log.Printf("[ERROR] falha ao publicar evento de estorno aprovado order=%s: %v", order.Id, pubErr)
		}
	}

	return useCase.orderRepository.ReadOrderById(order.Id.String())
}
