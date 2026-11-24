package ports

import (
	entities "e-commerce/core/entities/user"
	"time"
)

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
	QueuePaymentSync        = "payment_sync"

	EventOrderPaymentApproved = "ORDER_PAYMENT_APPROVED"
	EventOrderPaymentFailed   = "ORDER_PAYMENT_FAILED"
	EventOrderShipped         = "ORDER_SHIPPED"
	EventOrderDelivered       = "ORDER_DELIVERED"
	EventOrderRefundApproved  = "ORDER_REFUND_APPROVED"
	EventOrderRefundRejected  = "ORDER_REFUND_REJECTED"

	PaymentSyncActionUpdateGatewayData = "UPDATE_GATEWAY_DATA"
	PaymentSyncActionApprove           = "APPROVE"
	PaymentSyncActionFail              = "FAIL"
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

type PaymentSyncEvent struct {
	Action            string     `json:"action"`
	PaymentId         string     `json:"payment_id"`
	OrderId           string     `json:"order_id"`
	Provider          string     `json:"provider"`
	Method            string     `json:"method"`
	ExternalPaymentId string     `json:"external_payment_id"`
	IdempotencyKey    string     `json:"idempotency_key"`
	Amount            float64    `json:"amount"`
	Installments      uint8      `json:"installments"`
	Status            string     `json:"status"`
	PixQrCode         string     `json:"pix_qr_code,omitempty"`
	PixCopyPaste      string     `json:"pix_copy_paste,omitempty"`
	CardLast4         string     `json:"card_last4,omitempty"`
	CardBrand         string     `json:"card_brand,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	PaidAt            *time.Time `json:"paid_at,omitempty"`
	UserEmail         string     `json:"user_email,omitempty"`
	UserName          string     `json:"user_name,omitempty"`
}

type EventPaymentSyncQueue interface {
	DeclareQueue(eventName string) error
	PublishPaymentSync(event PaymentSyncEvent) error
}

type EventPaymentSyncConsumer interface {
	DeclareQueue(eventName string) error
	ConsumePaymentSync() error
}
