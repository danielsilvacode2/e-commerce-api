package ports

import "time"

type PixGatewayResult struct {
	ExternalPaymentId string
	PixQrCode         string
	PixCopyPaste      string
	ExpiresAt         time.Time
}

type CreditCardGatewayInput struct {
	OrderId         string
	Amount          float64
	Installments    uint8
	PayerEmail      string
	PayerDocType    string
	PayerDocNumber  string
	CardToken       string
	PaymentMethodId string
	CardNumber      string
	CardHolderName  string
	CardExpMonth    int
	CardExpYear     int
	CardCVV         string
	IdempotencyKey  string
}

type CardGatewayResult struct {
	ExternalPaymentId string
	Approved          bool
	CardLast4         string
	CardBrand         string
	FailureReason     string
}

type PaymentGateway interface {
	CreatePixPayment(orderId string, amount float64, payerEmail string, idempotencyKey string) (*PixGatewayResult, error)
	ProcessCreditCardPayment(input CreditCardGatewayInput) (*CardGatewayResult, error)
	RefundPayment(externalPaymentId string, amount float64) error
	GetPaymentStatus(externalPaymentId string) (string, error)
}
