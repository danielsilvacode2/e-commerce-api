package dto

import (
	fielderrors "e-commerce/adapters/controller/fieldErrors"
	"e-commerce/config"
	entitiesPayment "e-commerce/core/entities/payment"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ProcessPaymentRequestDTO struct {
	Method          string `json:"method"`
	Installments    uint8  `json:"installments"`
	PayerDocType    string `json:"payer_doc_type"`
	PayerDocNumber  string `json:"payer_doc_number"`
	CardToken       string `json:"card_token"`
	PaymentMethodId string `json:"payment_method_id"`
	CardNumber      string `json:"card_number"`
	CardHolderName  string `json:"card_holder_name"`
	CardExpMonth    int    `json:"card_exp_month"`
	CardExpYear     int    `json:"card_exp_year"`
	CardCVV         string `json:"card_cvv"`
}

func (dto *ProcessPaymentRequestDTO) Validator() []fielderrors.FieldValidateErrors {
	var listFieldErrors []fielderrors.FieldValidateErrors

	method := strings.ToUpper(strings.TrimSpace(dto.Method))
	if !entitiesPayment.IsValidPaymentMethod(method) {
		listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
			Field:   "method",
			Message: "método de pagamento inválido (use PIX ou CREDIT_CARD)",
		})
		return listFieldErrors
	}

	if method == entitiesPayment.PaymentMethodCreditCard {
		// Em producao (Mercado Pago ativo), exige tokenizacao no front-end (PCI-DSS) e recusa dados brutos de cartao
		if config.AccessTokenPayment != "" && strings.TrimSpace(dto.CardToken) == "" {
			listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
				Field:   "card_token",
				Message: "card_token é obrigatório para pagamento com cartão",
			})
			return listFieldErrors
		}

		// Quando tokenizado pelo SDK do Mercado Pago no front, valida card_token e payment_method_id
		if strings.TrimSpace(dto.CardToken) != "" {
			if strings.TrimSpace(dto.PaymentMethodId) == "" {
				listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
					Field:   "payment_method_id",
					Message: "bandeira do cartão (payment_method_id) é obrigatória",
				})
			}
			if dto.Installments > 12 {
				listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
					Field:   "installments",
					Message: "número máximo de parcelas é 12",
				})
			}
			return listFieldErrors
		}

		digits := onlyDigits(dto.CardNumber)
		if len(digits) < 13 || len(digits) > 19 {
			listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
				Field:   "card_number",
				Message: "número do cartão inválido",
			})
		}

		if len(strings.TrimSpace(dto.CardHolderName)) < 3 {
			listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
				Field:   "card_holder_name",
				Message: "nome do titular do cartão é obrigatório",
			})
		}

		if dto.CardExpMonth < 1 || dto.CardExpMonth > 12 {
			listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
				Field:   "card_exp_month",
				Message: "mês de validade inválido (1 a 12)",
			})
		}

		now := time.Now()
		if dto.CardExpYear < now.Year() || (dto.CardExpYear == now.Year() && dto.CardExpMonth < int(now.Month())) {
			listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
				Field:   "card_exp_year",
				Message: "cartão expirado",
			})
		}

		cvvDigits := onlyDigits(dto.CardCVV)
		if len(cvvDigits) < 3 || len(cvvDigits) > 4 {
			listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
				Field:   "card_cvv",
				Message: "CVV inválido (3 ou 4 dígitos)",
			})
		}

		if dto.Installments > 12 {
			listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
				Field:   "installments",
				Message: "número máximo de parcelas é 12",
			})
		}
	}

	return listFieldErrors
}

type WebhookPaymentRequestDTO struct {
	PaymentId         string `json:"payment_id"`
	ExternalPaymentId string `json:"external_payment_id"`
	Action            string `json:"action"`
}

func (dto *WebhookPaymentRequestDTO) Validator() []fielderrors.FieldValidateErrors {
	var listFieldErrors []fielderrors.FieldValidateErrors

	act := strings.ToUpper(strings.TrimSpace(dto.Action))
	if act != "APPROVE" && act != "FAIL" && act != "EXPIRE" {
		listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
			Field:   "action",
			Message: "ação inválida (APPROVE, FAIL ou EXPIRE)",
		})
	}

	return listFieldErrors
}

type PaymentResponseDTO struct {
	Id                uuid.UUID  `json:"id"`
	OrderId           uuid.UUID  `json:"order_id"`
	Provider          string     `json:"provider"`
	Method            string     `json:"method"`
	ExternalPaymentId string     `json:"external_payment_id,omitempty"`
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
	RefundedAt        *time.Time `json:"refunded_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
