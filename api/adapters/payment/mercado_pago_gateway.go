package payment

import (
	"bytes"
	entitiesPayment "e-commerce/core/entities/payment"
	"e-commerce/core/ports"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strings"
	"time"
)

const mercadoPagoBaseURL = "https://api.mercadopago.com"

type MercadoPagoGateway struct {
	accessToken string
	httpClient  *http.Client
}

func NewMercadoPagoGateway(accessToken string) ports.PaymentGateway {
	return &MercadoPagoGateway{
		accessToken: strings.TrimSpace(accessToken),
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

type mpPaymentResponse struct {
	ID              json.Number `json:"id"`
	Status          string      `json:"status"`
	StatusDetail    string      `json:"status_detail"`
	PaymentMethodID string      `json:"payment_method_id"`
	DateOfExpiration string     `json:"date_of_expiration"`
	Card            struct {
		LastFourDigits string `json:"last_four_digits"`
	} `json:"card"`
	PointOfInteraction struct {
		TransactionData struct {
			QrCode       string `json:"qr_code"`
			QrCodeBase64 string `json:"qr_code_base64"`
		} `json:"transaction_data"`
	} `json:"point_of_interaction"`
	Message string `json:"message"`
	Cause   []struct {
		Code        any    `json:"code"`
		Description string `json:"description"`
	} `json:"cause"`
}

func roundCurrency(amount float64) float64 {
	return math.Round(amount*100) / 100
}

func (g *MercadoPagoGateway) CreatePixPayment(orderId string, amount float64, payerEmail string, idempotencyKey string) (*ports.PixGatewayResult, error) {
	if strings.TrimSpace(payerEmail) == "" {
		payerEmail = "cliente@nero.com.br"
	}

	expiresAt := time.Now().Add(30 * time.Minute)
	payload := map[string]any{
		"transaction_amount": roundCurrency(amount),
		"description":        fmt.Sprintf("Pedido NERO #%s", orderId),
		"payment_method_id":  "pix",
		"external_reference": orderId,
		"date_of_expiration": expiresAt.Format("2006-01-02T15:04:05.000-07:00"),
		"payer": map[string]any{
			"email": payerEmail,
		},
	}

	mpRes, err := g.doPaymentRequest(http.MethodPost, "/v1/payments", idempotencyKey, payload)
	if err != nil {
		return nil, err
	}

	qrBase64 := mpRes.PointOfInteraction.TransactionData.QrCodeBase64
	qrDataURI := ""
	if qrBase64 != "" {
		qrDataURI = "data:image/png;base64," + qrBase64
	}

	if mpRes.DateOfExpiration != "" {
		if parsed, parseErr := time.Parse(time.RFC3339, mpRes.DateOfExpiration); parseErr == nil {
			expiresAt = parsed
		}
	}

	return &ports.PixGatewayResult{
		ExternalPaymentId: mpRes.ID.String(),
		PixQrCode:         qrDataURI,
		PixCopyPaste:      mpRes.PointOfInteraction.TransactionData.QrCode,
		ExpiresAt:         expiresAt,
	}, nil
}

func (g *MercadoPagoGateway) ProcessCreditCardPayment(input ports.CreditCardGatewayInput) (*ports.CardGatewayResult, error) {
	payerEmail := strings.TrimSpace(input.PayerEmail)
	if payerEmail == "" {
		payerEmail = "cliente@nero.com.br"
	}

	docType := strings.ToUpper(strings.TrimSpace(input.PayerDocType))
	if docType == "" {
		docType = "CPF"
	}

	payer := map[string]any{
		"email": payerEmail,
	}
	if docDigits := cleanDigits(input.PayerDocNumber); docDigits != "" {
		payer["identification"] = map[string]any{
			"type":   docType,
			"number": docDigits,
		}
	}

	installments := input.Installments
	if installments < 1 {
		installments = 1
	}

	payload := map[string]any{
		"transaction_amount": roundCurrency(input.Amount),
		"token":              strings.TrimSpace(input.CardToken),
		"description":        fmt.Sprintf("Pedido NERO #%s", input.OrderId),
		"installments":       installments,
		"payment_method_id":  strings.ToLower(strings.TrimSpace(input.PaymentMethodId)),
		"external_reference": input.OrderId,
		"payer":              payer,
	}

	mpRes, err := g.doPaymentRequest(http.MethodPost, "/v1/payments", input.IdempotencyKey, payload)
	if err != nil {
		return nil, err
	}

	approved := strings.EqualFold(mpRes.Status, "approved")
	brand := strings.ToUpper(mpRes.PaymentMethodID)
	if brand == "" {
		brand = strings.ToUpper(input.PaymentMethodId)
	}

	return &ports.CardGatewayResult{
		ExternalPaymentId: mpRes.ID.String(),
		Approved:          approved,
		CardLast4:         mpRes.Card.LastFourDigits,
		CardBrand:         brand,
		FailureReason:     mpRes.StatusDetail,
	}, nil
}

func isSafeExternalPaymentID(id string) bool {
	id = strings.TrimSpace(id)
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

func (g *MercadoPagoGateway) RefundPayment(externalPaymentId string, amount float64) error {
	externalPaymentId = strings.TrimSpace(externalPaymentId)
	if externalPaymentId == "" {
		return nil
	}
	if !isSafeExternalPaymentID(externalPaymentId) {
		return fmt.Errorf("external_payment_id invalido")
	}

	path := fmt.Sprintf("/v1/payments/%s/refunds", externalPaymentId)
	idempotencyKey := fmt.Sprintf("refund:%s", externalPaymentId)

	_, err := g.doPaymentRequest(http.MethodPost, path, idempotencyKey, map[string]any{})
	return err
}

func (g *MercadoPagoGateway) GetPaymentStatus(externalPaymentId string) (string, error) {
	externalPaymentId = strings.TrimSpace(externalPaymentId)
	if externalPaymentId == "" {
		return "", nil
	}
	if !isSafeExternalPaymentID(externalPaymentId) {
		return "", fmt.Errorf("external_payment_id invalido")
	}

	path := fmt.Sprintf("/v1/payments/%s", externalPaymentId)
	mpRes, err := g.doPaymentRequest(http.MethodGet, path, "", nil)
	if err != nil {
		return "", err
	}

	switch strings.ToLower(mpRes.Status) {
	case "approved":
		return entitiesPayment.PaymentStatusApproved, nil
	case "rejected", "cancelled":
		return entitiesPayment.PaymentStatusFailed, nil
	case "refunded", "charged_back":
		return entitiesPayment.PaymentStatusRefunded, nil
	default:
		return entitiesPayment.PaymentStatusPending, nil
	}
}

func (g *MercadoPagoGateway) doPaymentRequest(method, path, idempotencyKey string, bodyPayload any) (*mpPaymentResponse, error) {
	var bodyReader io.Reader
	if bodyPayload != nil {
		raw, err := json.Marshal(bodyPayload)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, mercadoPagoBaseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+g.accessToken)
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(idempotencyKey) != "" {
		req.Header.Set("X-Idempotency-Key", idempotencyKey)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao comunicar com Mercado Pago: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var mpRes mpPaymentResponse
	_ = json.Unmarshal(respBytes, &mpRes)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("[ERROR] [MercadoPagoGateway] %s %s status=%d body=%s", method, path, resp.StatusCode, string(respBytes))
		msg := mpRes.Message
		if len(mpRes.Cause) > 0 {
			causes := make([]string, 0, len(mpRes.Cause))
			for _, c := range mpRes.Cause {
				if strings.TrimSpace(c.Description) != "" {
					causes = append(causes, fmt.Sprintf("%v: %s", c.Code, c.Description))
				}
			}
			if len(causes) > 0 {
				if msg != "" {
					msg = fmt.Sprintf("%s (%s)", msg, strings.Join(causes, "; "))
				} else {
					msg = strings.Join(causes, "; ")
				}
			}
		}
		if msg == "" {
			msg = string(respBytes)
		}
		return nil, fmt.Errorf("mercado pago retornou status %d: %s", resp.StatusCode, msg)
	}

	return &mpRes, nil
}
