package mapper

import (
	"e-commerce/adapters/controller/payment/dto"
	entitiesPayment "e-commerce/core/entities/payment"
)

func PaymentToDto(p entitiesPayment.Payment) dto.PaymentResponseDTO {
	return dto.PaymentResponseDTO{
		Id:                p.Id,
		OrderId:           p.OrderId,
		Provider:          p.Provider,
		Method:            p.Method,
		ExternalPaymentId: p.ExternalPaymentId,
		IdempotencyKey:    p.IdempotencyKey,
		Amount:            p.Amount,
		Installments:      p.Installments,
		Status:            p.Status,
		PixQrCode:         p.PixQrCode,
		PixCopyPaste:      p.PixCopyPaste,
		CardLast4:         p.CardLast4,
		CardBrand:         p.CardBrand,
		ExpiresAt:         p.ExpiresAt,
		PaidAt:            p.PaidAt,
		RefundedAt:        p.RefundedAt,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	}
}
