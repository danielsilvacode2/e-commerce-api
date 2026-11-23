package ports

import entities "e-commerce/core/entities/user"

type EventUserQueue interface {
	DeclareQueue(eventName string) error
	PublishEvent(eventName string, user entities.User) error
}

type EventeUserQueueConsumerDeclare interface {
	ConsumerUser() error
	DeclareQueue(eventName string) error
}

const (
	QueueCreateUser         = "createusers"
	QueueOrderNotifications = "order_notifications"

	EventOrderPaymentApproved = "ORDER_PAYMENT_APPROVED"
	EventOrderPaymentFailed   = "ORDER_PAYMENT_FAILED"
	EventOrderShipped         = "ORDER_SHIPPED"
	EventOrderDelivered       = "ORDER_DELIVERED"
	EventOrderRefundApproved  = "ORDER_REFUND_APPROVED"
	EventOrderRefundRejected  = "ORDER_REFUND_REJECTED"
)

type OrderNotificationEvent struct {
	EventType       string  `json:"event_type"`
	OrderId         string  `json:"order_id"`
	UserEmail       string  `json:"user_email"`
	UserName        string  `json:"user_name"`
	OrderTotal      float64 `json:"order_total"`
	PaymentMethod   string  `json:"payment_method,omitempty"`
	TrackingCode    string  `json:"tracking_code,omitempty"`
	RefundReason    string  `json:"refund_reason,omitempty"`
	RejectionReason string  `json:"rejection_reason,omitempty"`
}

type EventOrderQueue interface {
	DeclareQueue(eventName string) error
	PublishOrderNotification(event OrderNotificationEvent) error
}

type EventOrderQueueConsumer interface {
	DeclareQueue(eventName string) error
	ConsumeOrderNotifications() error
}
