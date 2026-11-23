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

	refundedPayment, err := useCase.paymentRepository.RefundPaymentAndOrder(
		order.Id.String(),
		entitiesOrder.OrderStatusRefundRequested,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	if refundedPayment != nil && refundedPayment.ExternalPaymentId != "" && useCase.paymentGateway != nil {
		if gwErr := useCase.paymentGateway.RefundPayment(refundedPayment.ExternalPaymentId, refundedPayment.Amount); gwErr != nil {
			log.Printf("[WARN] falha ao notificar gateway sobre aprovacao de estorno order=%s: %v", order.Id, gwErr)
		}
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
