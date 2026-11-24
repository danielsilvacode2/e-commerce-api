package entities

import (
	"time"

	"github.com/google/uuid"
)

const (
	PaymentStatusPending  = "PENDING"
	PaymentStatusApproved = "APPROVED"
	PaymentStatusFailed   = "FAILED"
	PaymentStatusExpired  = "EXPIRED"
	PaymentStatusRefunded = "REFUNDED"

	PaymentMethodPix        = "PIX"
	PaymentMethodCreditCard = "CREDIT_CARD"

	PaymentProviderDefault = "mercado_pago"
)

type Payment struct {
	Id                uuid.UUID
	OrderId           uuid.UUID
	Provider          string
	Method            string
	ExternalPaymentId string
	IdempotencyKey    string
	Amount            float64
	Installments      uint8
	Status            string
	PixQrCode         string
	PixCopyPaste      string
	CardLast4         string
	CardBrand         string
	ExpiresAt         *time.Time
	PaidAt            *time.Time
	RefundedAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func IsValidPaymentMethod(method string) bool {
	return method == PaymentMethodPix || method == PaymentMethodCreditCard
}

func IsValidPaymentStatus(status string) bool {
	switch status {
	case PaymentStatusPending,
		PaymentStatusApproved,
		PaymentStatusFailed,
		PaymentStatusExpired,
		PaymentStatusRefunded:
		return true
	}
	return false
}

func (p *Payment) IsExpired(now time.Time) bool {
	return p.Status == PaymentStatusPending && p.ExpiresAt != nil && now.After(*p.ExpiresAt)
}

type PaymentRepository interface {
	CreatePayment(payment Payment) error
	UpdatePaymentGatewayData(payment Payment) error
	DeletePayment(id string) error
	ReadPaymentById(id string) (*Payment, error)
	ReadPaymentByOrderId(orderId string) (*Payment, error)
	ReadPaymentByIdempotencyKey(key string) (*Payment, error)
	ReadPaymentByExternalId(externalId string) (*Payment, error)
	ApprovePaymentAndOrder(payment Payment, orderId string) error
	FailOrExpirePaymentAndCancelOrder(payment Payment, orderId string) error
	RefundPaymentAndOrder(orderId string, expectedOrderStatus string, restoreStock bool, refundReason *string) (*Payment, error)
}
