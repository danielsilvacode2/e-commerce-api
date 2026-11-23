package use_cases

import (
	"e-commerce/core/apperrors"
	entitiesOrder "e-commerce/core/entities/order"
	"e-commerce/core/ports"
	"log"
	"strings"
)

type RejectRefundUseCase struct {
	orderRepository entitiesOrder.OrderRepository
	readOrderById   *ReadOrderByIdUseCase
	eventQueue      ports.EventOrderQueue
}

func NewRejectRefundUseCase(
	orderRepository entitiesOrder.OrderRepository,
	readOrderById *ReadOrderByIdUseCase,
	eventQueue ports.EventOrderQueue,
) *RejectRefundUseCase {
	return &RejectRefundUseCase{
		orderRepository: orderRepository,
		readOrderById:   readOrderById,
		eventQueue:      eventQueue,
	}
}

// UseCase recusa uma solicitacao de estorno (REFUND_REQUESTED -> DELIVERED) com justificativa e notifica o cliente.
func (useCase *RejectRefundUseCase) UseCase(orderId string, rejectionReason string, adminUserId string, role string) (*entitiesOrder.Order, error) {
	if role != roleAdmin {
		return nil, apperrors.ErrForbidden
	}

	cleanReason := strings.TrimSpace(rejectionReason)
	if cleanReason == "" {
		return nil, apperrors.ErrInvalidInput
	}

	order, err := useCase.readOrderById.UseCase(orderId, adminUserId, role)
	if err != nil {
		return nil, err
	}

	if order.Status != entitiesOrder.OrderStatusRefundRequested {
		return nil, apperrors.ErrUnprocessableEntity
	}

	if err := useCase.orderRepository.UpdateOrderStatusWithDetails(
		order.Id.String(),
		entitiesOrder.OrderStatusRefundRequested,
		entitiesOrder.OrderStatusDelivered,
		false,
		nil,
		nil,
		&cleanReason,
	); err != nil {
		return nil, err
	}

	if useCase.eventQueue != nil && order.User != nil {
		if pubErr := useCase.eventQueue.PublishOrderNotification(ports.OrderNotificationEvent{
			EventType:       ports.EventOrderRefundRejected,
			OrderId:         order.Id.String(),
			UserEmail:       order.User.Email,
			UserName:        order.User.Name,
			OrderTotal:      order.Total(),
			RefundReason:    order.RefundReason,
			RejectionReason: cleanReason,
		}); pubErr != nil {
			log.Printf("[ERROR] falha ao publicar evento de estorno recusado order=%s: %v", order.Id, pubErr)
		}
	}

	return useCase.orderRepository.ReadOrderById(order.Id.String())
}
