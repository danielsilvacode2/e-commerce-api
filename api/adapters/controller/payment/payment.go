package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/controller/payment/dto"
	"e-commerce/adapters/controller/payment/mapper"
	"e-commerce/adapters/security/auth"
	"e-commerce/config"
	"e-commerce/core/apperrors"
	useCasesPayment "e-commerce/core/use_cases/payment"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

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
// @Summary      Processar pagamento de pedido
// @Description  Processa o pagamento de um pedido via PIX ou CREDIT_CARD. Exige o header Idempotency-Key para garantir idempotencia.
// @Tags         Payments
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Param        id               path      string                        true  "ID do pedido (UUID)"
// @Param        Idempotency-Key  header    string                        true  "Chave unica de idempotencia"
// @Param        request          body      dto.ProcessPaymentRequestDTO  true  "Dados do pagamento"
// @Success      200              {object}  helpers.SwaggerDataResponse[dto.PaymentResponseDTO]
// @Failure      400              {object}  helpers.SwaggerValidationErrorResponse
// @Failure      401              {object}  helpers.SwaggerMessageResponse
// @Failure      403              {object}  helpers.SwaggerMessageResponse
// @Failure      404              {object}  helpers.SwaggerMessageResponse
// @Failure      409              {object}  helpers.SwaggerMessageResponse
// @Failure      422              {object}  helpers.SwaggerMessageResponse
// @Failure      500              {object}  helpers.SwaggerMessageResponse
// @Router       /orders/{id}/payments [post]
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
// @Summary      Consultar pagamento do pedido
// @Description  Retorna o ultimo pagamento associado a um pedido, verificando automaticamente se houve expiracao.
// @Tags         Payments
// @Produce      json
// @Security     CookieAuth
// @Param        id   path      string  true  "ID do pedido (UUID)"
// @Success      200  {object}  helpers.SwaggerDataResponse[dto.PaymentResponseDTO]
// @Failure      401  {object}  helpers.SwaggerMessageResponse
// @Failure      403  {object}  helpers.SwaggerMessageResponse
// @Failure      404  {object}  helpers.SwaggerMessageResponse
// @Failure      500  {object}  helpers.SwaggerMessageResponse
// @Router       /orders/{id}/payment [get]
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

// ConfirmOrderPayment permite simular confirmacao/recusa/expiracao apenas quando o gateway mock e a simulacao estiverem explicitamente habilitados.
// @Summary      Simular confirmacao de pagamento (mock)
// @Description  Simula aprovacao (APPROVE), falha (FAIL) ou expiracao (EXPIRE) de pagamento. Disponivel apenas quando ALLOW_PAYMENT_SIMULATION=true e sem ACCESS_TOKEN_PAYMENT.
// @Tags         Payments
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Param        id       path      string                        true  "ID do pedido (UUID)"
// @Param        request  body      dto.WebhookPaymentRequestDTO  true  "Acao de simulacao (APPROVE, FAIL ou EXPIRE)"
// @Success      200      {object}  helpers.SwaggerDataResponse[dto.PaymentResponseDTO]
// @Failure      400      {object}  helpers.SwaggerValidationErrorResponse
// @Failure      401      {object}  helpers.SwaggerMessageResponse
// @Failure      403      {object}  helpers.SwaggerMessageResponse
// @Failure      404      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /orders/{id}/payment/confirm [post]
func (ctrl *PaymentController) ConfirmOrderPayment(c *gin.Context) {
	if !config.AllowPaymentSimulation || config.AccessTokenPayment != "" {
		helpers.HelperErrResponse(nil, c, apperrors.ErrForbidden)
		return
	}

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

// WebhookPayment recebe callbacks externos do Mercado Pago validando assinatura HMAC e sincronizando o status real no gateway.
// @Summary      Webhook do Mercado Pago
// @Description  Recebe notificacoes de pagamento do Mercado Pago, valida assinatura HMAC-SHA256 no header x-signature e sincroniza o status do pagamento.
// @Tags         Payments
// @Accept       json
// @Produce      json
// @Param        x-signature   header    string                  true   "Assinatura HMAC no formato ts=<timestamp>,v1=<hex>"
// @Param        x-request-id  header    string                  true   "Identificador da requisicao enviado pelo Mercado Pago"
// @Param        data.id       query     string                  false  "ID externo do pagamento no Mercado Pago (alternativa ao campo data.id do body)"
// @Param        request       body      map[string]interface{}  false  "Payload nativo de notificacao do Mercado Pago"
// @Success      200           {object}  helpers.SwaggerDataResponse[dto.PaymentResponseDTO]
// @Failure      400           {object}  helpers.SwaggerMessageResponse
// @Failure      401           {object}  helpers.SwaggerMessageResponse
// @Failure      404           {object}  helpers.SwaggerMessageResponse
// @Failure      429           {object}  helpers.SwaggerMessageResponse
// @Failure      500           {object}  helpers.SwaggerMessageResponse
// @Router       /payments/webhook [post]
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

	if !isValidExternalPaymentID(mpExternalId) {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	if !verifyMercadoPagoSignature(c.GetHeader("x-signature"), c.GetHeader("x-request-id"), mpExternalId, config.WebhookSecret) {
		helpers.HelperErrResponse(nil, c, apperrors.ErrUnauthorized)
		return
	}

	payment, err := ctrl.webhookPayment.SyncByExternalId(mpExternalId)
	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}
	helpers.HelperResponse(mapper.PaymentToDto(*payment), http.StatusOK, "mercado pago webhook synced", c)
}

func isValidExternalPaymentID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, r := range id {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

const webhookMaxTimestampSkew = 5 * time.Minute

// verifyMercadoPagoSignature valida o header x-signature (ts=...,v1=...) usando HMAC-SHA256 com MERCADO_PAGO_WEBHOOK_SECRET
// e rejeita timestamps fora da janela toleravel para mitigar ataques de replay.
func verifyMercadoPagoSignature(xSignature, xRequestID, dataID, secret string) bool {
	if strings.TrimSpace(secret) == "" {
		return false
	}
	var ts, v1 string
	for _, part := range strings.Split(xSignature, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch strings.TrimSpace(kv[0]) {
		case "ts":
			ts = strings.TrimSpace(kv[1])
		case "v1":
			v1 = strings.TrimSpace(kv[1])
		}
	}
	if ts == "" || v1 == "" {
		return false
	}

	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || tsInt <= 0 {
		return false
	}
	// Suporta timestamp em milissegundos ou segundos
	if tsInt > 1_000_000_000_000 {
		tsInt = tsInt / 1000
	}
	delta := time.Since(time.Unix(tsInt, 0))
	if delta < -webhookMaxTimestampSkew || delta > webhookMaxTimestampSkew {
		return false
	}

	manifest := fmt.Sprintf("id:%s;request-id:%s;ts:%s;", strings.ToLower(dataID), strings.TrimSpace(xRequestID), ts)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(manifest))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(strings.ToLower(v1)), []byte(expected))
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%.0f", val)
	default:
		return ""
	}
}
