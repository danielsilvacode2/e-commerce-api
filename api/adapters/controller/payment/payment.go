package controller

import (
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/controller/payment/dto"
	"e-commerce/adapters/controller/payment/mapper"
	"e-commerce/adapters/security/auth"
	"e-commerce/core/apperrors"
	useCasesPayment "e-commerce/core/use_cases/payment"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type PaymentController struct {
	processPayment     *useCasesPayment.ProcessPaymentUseCase
	readPaymentByOrder *useCasesPayment.ReadPaymentByOrderUseCase
	webhookPayment     *useCasesPayment.WebhookPaymentUseCase
}

func NewPaymentController(
	processPayment *useCasesPayment.ProcessPaymentUseCase,
	readPaymentByOrder *useCasesPayment.ReadPaymentByOrderUseCase,
	webhookPayment *useCasesPayment.WebhookPaymentUseCase,
) *PaymentController {
	return &PaymentController{
		processPayment:     processPayment,
		readPaymentByOrder: readPaymentByOrder,
		webhookPayment:     webhookPayment,
	}
}

func (ctrl *PaymentController) getClaims(c *gin.Context) (*auth.Claims, error) {
	val, exists := c.Get("claims")
	if !exists {
		return nil, apperrors.ErrUnauthorized
	}
	claims, ok := val.(*auth.Claims)
	if !ok {
		return nil, errors.New("Error converting declarations")
	}
	return claims, nil
}

// ProcessPayment processa pagamento Pix ou Cartao de Credito exigindo Idempotency-Key no header HTTP.
func (ctrl *PaymentController) ProcessPayment(c *gin.Context) {
	claims, err := ctrl.getClaims(c)
	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		helpers.HelperResponse(nil, http.StatusBadRequest, "Header Idempotency-Key é obrigatório", c)
		return
	}

	var dtoRequest dto.ProcessPaymentRequestDTO
	if err = c.ShouldBindJSON(&dtoRequest); err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	if listErrField := dtoRequest.Validator(); listErrField != nil {
		helpers.HelperErrResponse(listErrField, c, apperrors.ErrInvalidInput)
		return
	}

	payment, err := ctrl.processPayment.UseCase(useCasesPayment.ProcessPaymentInput{
		OrderId:         c.Param("id"),
		UserId:          claims.Id.String(),
		Role:            claims.Role,
		IdempotencyKey:  idempotencyKey,
		Method:          dtoRequest.Method,
		Installments:    dtoRequest.Installments,
		PayerDocType:    dtoRequest.PayerDocType,
		PayerDocNumber:  dtoRequest.PayerDocNumber,
		CardToken:       dtoRequest.CardToken,
		PaymentMethodId: dtoRequest.PaymentMethodId,
		CardNumber:      dtoRequest.CardNumber,
		CardHolderName:  dtoRequest.CardHolderName,
		CardExpMonth:    dtoRequest.CardExpMonth,
		CardExpYear:     dtoRequest.CardExpYear,
		CardCVV:         dtoRequest.CardCVV,
	})
	if err != nil {
		log.Printf("[ERROR] [PaymentController.ProcessPayment] order=%s method=%s: %v", c.Param("id"), dtoRequest.Method, err)
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(mapper.PaymentToDto(*payment), http.StatusOK, "payment processed", c)
}

// ReadPaymentByOrder consulta o ultimo pagamento de um pedido (com verificacao automatica de expiracao).
func (ctrl *PaymentController) ReadPaymentByOrder(c *gin.Context) {
	claims, err := ctrl.getClaims(c)
	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	payment, err := ctrl.readPaymentByOrder.UseCase(c.Param("id"), claims.Id.String(), claims.Role)
	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(mapper.PaymentToDto(*payment), http.StatusOK, "found payment", c)
}

// ConfirmOrderPayment permite confirmar/recusar/expirar o pagamento pendente de um pedido (usado na simulacao Pix / callback autenticado).
func (ctrl *PaymentController) ConfirmOrderPayment(c *gin.Context) {
	claims, err := ctrl.getClaims(c)
	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	var dtoRequest dto.WebhookPaymentRequestDTO
	if err = c.ShouldBindJSON(&dtoRequest); err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	if listErrField := dtoRequest.Validator(); listErrField != nil {
		helpers.HelperErrResponse(listErrField, c, apperrors.ErrInvalidInput)
		return
	}

	currentPayment, err := ctrl.readPaymentByOrder.UseCase(c.Param("id"), claims.Id.String(), claims.Role)
	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	payment, err := ctrl.webhookPayment.UseCase(currentPayment.Id.String(), "", dtoRequest.Action)
	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(mapper.PaymentToDto(*payment), http.StatusOK, "payment updated", c)
}

// WebhookPayment recebe callbacks externos (Mercado Pago nativo ou payload interno) de forma idempotente.
func (ctrl *PaymentController) WebhookPayment(c *gin.Context) {
	var raw map[string]any
	_ = c.ShouldBindJSON(&raw)

	// Formato nativo do Mercado Pago: {"type":"payment","data":{"id":"123456"}} ou ?data.id=123456&type=payment
	mpExternalId := strings.TrimSpace(c.Query("data.id"))
	if mpExternalId == "" && raw != nil {
		if dataMap, ok := raw["data"].(map[string]any); ok {
			if idVal, exists := dataMap["id"]; exists && idVal != nil {
				mpExternalId = strings.TrimSpace(toString(idVal))
			}
		}
	}

	if mpExternalId != "" {
		payment, err := ctrl.webhookPayment.SyncByExternalId(mpExternalId)
		if err != nil {
			helpers.HelperErrResponse(nil, c, err)
			return
		}
		helpers.HelperResponse(mapper.PaymentToDto(*payment), http.StatusOK, "mercado pago webhook synced", c)
		return
	}

	var dtoRequest dto.WebhookPaymentRequestDTO
	if raw != nil {
		dtoRequest.PaymentId = toString(raw["payment_id"])
		dtoRequest.ExternalPaymentId = toString(raw["external_payment_id"])
		dtoRequest.Action = toString(raw["action"])
	}

	if listErrField := dtoRequest.Validator(); listErrField != nil {
		helpers.HelperErrResponse(listErrField, c, apperrors.ErrInvalidInput)
		return
	}

	payment, err := ctrl.webhookPayment.UseCase(dtoRequest.PaymentId, dtoRequest.ExternalPaymentId, dtoRequest.Action)
	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(mapper.PaymentToDto(*payment), http.StatusOK, "webhook processed", c)
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
