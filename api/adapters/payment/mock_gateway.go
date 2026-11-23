package payment

import (
	"crypto/sha256"
	"e-commerce/core/ports"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

type MockPaymentGateway struct{}

func NewMockPaymentGateway() ports.PaymentGateway {
	return &MockPaymentGateway{}
}

func (g *MockPaymentGateway) CreatePixPayment(orderId string, amount float64, payerEmail string, idempotencyKey string) (*ports.PixGatewayResult, error) {
	externalID := "pix_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:20]
	amountStr := fmt.Sprintf("%.2f", amount)

	// Payload estilo EMV BR Code (Copia e Cola)
	copyPaste := fmt.Sprintf(
		"00020126580014BR.GOV.BCB.PIX0136%s52040000530398654%02d%s5802BR5904NERO6009SAO PAULO62070503***6304%s",
		orderId,
		len(amountStr),
		amountStr,
		strings.ToUpper(externalID[len(externalID)-4:]),
	)

	qrDataURI := generateDeterministicQRCodeSVG(copyPaste)
	expiresAt := time.Now().Add(15 * time.Minute)

	return &ports.PixGatewayResult{
		ExternalPaymentId: externalID,
		PixQrCode:         qrDataURI,
		PixCopyPaste:      copyPaste,
		ExpiresAt:         expiresAt,
	}, nil
}

func (g *MockPaymentGateway) ProcessCreditCardPayment(input ports.CreditCardGatewayInput) (*ports.CardGatewayResult, error) {
	digits := cleanDigits(input.CardNumber)
	last4 := digits
	if len(digits) >= 4 {
		last4 = digits[len(digits)-4:]
	}

	brand := detectCardBrand(digits)
	externalID := "card_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:20]

	// Regra deterministica para testes: cartoes terminados em 0000 simulam recusa pela operadora
	if last4 == "0000" {
		return &ports.CardGatewayResult{
			ExternalPaymentId: externalID,
			Approved:          false,
			CardLast4:         last4,
			CardBrand:         brand,
			FailureReason:     "Transacao recusada pela operadora do cartao",
		}, nil
	}

	return &ports.CardGatewayResult{
		ExternalPaymentId: externalID,
		Approved:          true,
		CardLast4:         last4,
		CardBrand:         brand,
	}, nil
}

func (g *MockPaymentGateway) RefundPayment(externalPaymentId string, amount float64) error {
	log.Printf("[INFO] [MockPaymentGateway] Estorno processado para external_payment_id=%s amount=%.2f", externalPaymentId, amount)
	return nil
}

func (g *MockPaymentGateway) GetPaymentStatus(externalPaymentId string) (string, error) {
	return "", nil
}

func cleanDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func detectCardBrand(digits string) string {
	if len(digits) == 0 {
		return "OUTRO"
	}
	switch {
	case strings.HasPrefix(digits, "4"):
		return "VISA"
	case strings.HasPrefix(digits, "51"), strings.HasPrefix(digits, "52"),
		strings.HasPrefix(digits, "53"), strings.HasPrefix(digits, "54"),
		strings.HasPrefix(digits, "55"):
		return "MASTERCARD"
	case strings.HasPrefix(digits, "34"), strings.HasPrefix(digits, "37"):
		return "AMEX"
	case strings.HasPrefix(digits, "636368"), strings.HasPrefix(digits, "5067"), strings.HasPrefix(digits, "438935"):
		return "ELO"
	default:
		return "CARTAO"
	}
}

// generateDeterministicQRCodeSVG gera um QR Code visual em SVG (data URI) deterministico a partir do payload.
func generateDeterministicQRCodeSVG(payload string) string {
	const size = 21
	const cell = 10
	const margin = 20
	dim := size*cell + margin*2

	hash := sha256.Sum256([]byte(payload))
	hash2 := sha256.Sum256(hash[:])
	bits := append(hash[:], hash2[:]...)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d" shape-rendering="crispEdges">`, dim, dim, dim, dim))
	sb.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" fill="#ffffff"/>`, dim, dim))

	isFinder := func(r, c int) (bool, bool) {
		drawFinder := func(or, oc int) (bool, bool) {
			dr, dc := r-or, c-oc
			if dr < 0 || dr > 6 || dc < 0 || dc > 6 {
				return false, false
			}
			if dr == 0 || dr == 6 || dc == 0 || dc == 6 {
				return true, true
			}
			if dr >= 2 && dr <= 4 && dc >= 2 && dc <= 4 {
				return true, true
			}
			return true, false
		}
		if in, dark := drawFinder(0, 0); in {
			return true, dark
		}
		if in, dark := drawFinder(0, size-7); in {
			return true, dark
		}
		if in, dark := drawFinder(size-7, 0); in {
			return true, dark
		}
		return false, false
	}

	bitIdx := 0
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			dark := false
			if in, d := isFinder(r, c); in {
				dark = d
			} else if (r <= 7 && c <= 7) || (r <= 7 && c >= size-8) || (r >= size-8 && c <= 7) {
				dark = false
			} else {
				b := bits[(bitIdx/8)%len(bits)]
				dark = ((b >> uint(bitIdx%8)) & 1) == 1
				bitIdx++
			}
			if dark {
				sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" fill="#000000"/>`, margin+c*cell, margin+r*cell, cell, cell))
			}
		}
	}

	sb.WriteString(`</svg>`)
	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(sb.String()))
}
