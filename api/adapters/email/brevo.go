package email

import (
	"bytes"
	"e-commerce/core/ports"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"
)

const (
	defaultBrevoBaseURL   = "https://api.brevo.com/v3"
	defaultSenderName     = "NERO"
	brevoRequestTimeout   = 15 * time.Second
	maxBrevoResponseBytes = 1 << 20 // 1 MB
)

type BrevoMailer struct {
	baseURL    string
	apiKey     string
	from       string
	httpClient *http.Client
}

type brevoAddress struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email"`
}

type brevoEmailRequest struct {
	Sender      brevoAddress   `json:"sender"`
	To          []brevoAddress `json:"to"`
	Subject     string         `json:"subject"`
	HTMLContent string         `json:"htmlContent,omitempty"`
	TextContent string         `json:"textContent,omitempty"`
}

type brevoErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewBrevoMailer(apiKey, from string) ports.EmailService {
	return newBrevoMailerWithBaseURL(defaultBrevoBaseURL, apiKey, from)
}

func newBrevoMailerWithBaseURL(baseURL, apiKey, from string) *BrevoMailer {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = defaultBrevoBaseURL
	}

	return &BrevoMailer{
		baseURL: baseURL,
		apiKey:  strings.TrimSpace(apiKey),
		from:    strings.TrimSpace(from),
		httpClient: &http.Client{
			Timeout: brevoRequestTimeout,
		},
	}
}

func parseBrevoAddress(raw string, fallbackName string) (brevoAddress, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return brevoAddress{}, errors.New("endereco de email vazio")
	}

	parsed, err := mail.ParseAddress(raw)
	if err != nil {
		return brevoAddress{}, err
	}

	name := strings.TrimSpace(parsed.Name)
	if name == "" {
		name = fallbackName
	}

	return brevoAddress{
		Name:  name,
		Email: strings.TrimSpace(parsed.Address),
	}, nil
}

func (b *BrevoMailer) SendEmail(to, title, text, htmlBody string) error {
	if b.apiKey == "" {
		return errors.New("falha ao enviar email: BREVO_API_KEY nao configurada")
	}

	sender, err := parseBrevoAddress(b.from, defaultSenderName)
	if err != nil {
		return fmt.Errorf("falha ao enviar email: remetente invalido (%w)", err)
	}

	recipient, err := parseBrevoAddress(to, "")
	if err != nil {
		return fmt.Errorf("falha ao enviar email: destinatario invalido (%w)", err)
	}

	payload := brevoEmailRequest{
		Sender:      sender,
		To:          []brevoAddress{recipient},
		Subject:     title,
		HTMLContent: htmlBody,
		TextContent: text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("falha ao serializar email para Brevo: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, b.baseURL+"/smtp/email", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("falha ao criar requisicao para Brevo: %w", err)
	}

	req.Header.Set("api-key", b.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("falha ao enviar email via Brevo: %w", err)
	}
	defer res.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(res.Body, maxBrevoResponseBytes))
	if err != nil {
		return fmt.Errorf("falha ao ler resposta da Brevo: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		var brevoErr brevoErrorResponse
		errMsg := ""
		if json.Unmarshal(respBytes, &brevoErr) == nil && (brevoErr.Message != "" || brevoErr.Code != "") {
			if brevoErr.Code != "" && brevoErr.Message != "" {
				errMsg = fmt.Sprintf("%s: %s", brevoErr.Code, brevoErr.Message)
			} else if brevoErr.Message != "" {
				errMsg = brevoErr.Message
			} else {
				errMsg = brevoErr.Code
			}
		}
		if errMsg == "" {
			errMsg = strings.TrimSpace(string(respBytes))
			if len(errMsg) > 500 {
				errMsg = errMsg[:500]
			}
		}
		return fmt.Errorf("falha ao enviar email via Brevo (status %d): %s", res.StatusCode, errMsg)
	}

	return nil
}

func (b *BrevoMailer) SendVerificationCode(to, code string, textType string) error {
	subject, text, htmlBody := buildVerificationCodeEmail(code, textType)
	return b.SendEmail(to, subject, text, htmlBody)
}

func (b *BrevoMailer) SendSuspiciousLogin(to string) error {
	subject, text, htmlBody := buildSuspiciousLoginEmail()
	return b.SendEmail(to, subject, text, htmlBody)
}

func (b *BrevoMailer) SendOrderNotification(event ports.OrderNotificationEvent) error {
	subject, text, htmlBody := buildOrderNotificationEmail(event)
	return b.SendEmail(event.UserEmail, subject, text, htmlBody)
}
