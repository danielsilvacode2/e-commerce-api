package use_cases

import (
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/order"
	"e-commerce/core/ports"
	"log"
	"strings"
)

type UpdateOrderStatusUseCase struct {
	repository    entities.OrderRepository
	readOrderById *ReadOrderByIdUseCase
	eventQueue    ports.EventOrderQueue
}

func NewUpdateOrderStatusUseCase(
	repository entities.OrderRepository,
	readOrderById *ReadOrderByIdUseCase,
	eventQueue ports.EventOrderQueue,
) *UpdateOrderStatusUseCase {
	return &UpdateOrderStatusUseCase{
		repository:    repository,
		readOrderById: readOrderById,
		eventQueue:    eventQueue,
	}
}

// UseCase aplica uma transicao de status valida pelo admin (ex: PREPARING -> SHIPPED -> DELIVERED).
// Ao cancelar, o estoque dos itens e devolvido. Publica notificacao de envio/entrega na fila.
func (useCase *UpdateOrderStatusUseCase) UseCase(status string, trackingCode string, orderId string, userId string, role string) error {

	if !entities.IsValidOrderStatus(status) {
		return apperrors.ErrInvalidInput
	}

	order, err := useCase.readOrderById.UseCase(orderId, userId, role)

	if err != nil {
		return err
	}

	if order.IsFinalized() || !order.CanTransitionTo(status) {
		return apperrors.ErrUnprocessableEntity
	}

	restoreStock := status == entities.OrderStatusCanceled
	var trackingPtr *string
	cleanTracking := strings.TrimSpace(trackingCode)
	if status == entities.OrderStatusShipped && cleanTracking != "" {
		trackingPtr = &cleanTracking
	}

	if err = useCase.repository.UpdateOrderStatusWithDetails(orderId, order.Status, status, restoreStock, trackingPtr, nil, nil); err != nil {
		return err
	}

	if useCase.eventQueue != nil && order.User != nil {
		var eventType string
		switch status {
		case entities.OrderStatusShipped:
			eventType = ports.EventOrderShipped
		case entities.OrderStatusDelivered:
			eventType = ports.EventOrderDelivered
		}

		if eventType != "" {
			if pubErr := useCase.eventQueue.PublishOrderNotification(ports.OrderNotificationEvent{
				EventType:    eventType,
				OrderId:      order.Id.String(),
				UserEmail:    order.User.Email,
				UserName:     order.User.Name,
				OrderTotal:   order.Total(),
				TrackingCode: cleanTracking,
			}); pubErr != nil {
				log.Printf("[ERROR] falha ao publicar evento %s para order=%s: %v", eventType, orderId, pubErr)
			}
		}
	}

	return nil
}
