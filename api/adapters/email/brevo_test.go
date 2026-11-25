package email

import (
	"bytes"
	"e-commerce/core/ports"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestBrevoMailer(apiKey, from string, fn roundTripFunc) *BrevoMailer {
	m := newBrevoMailerWithBaseURL("https://api.brevo.test/v3", apiKey, from)
	m.httpClient = &http.Client{Transport: fn}
	return m
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

func TestBrevoMailer_SendEmailSuccess(t *testing.T) {
	var gotMethod, gotURL, gotAPIKey, gotContentType, gotAccept string
	var gotPayload brevoEmailRequest

	mailer := newTestBrevoMailer("xkeysib-test-key", "noreply@nero.com", func(r *http.Request) (*http.Response, error) {
		gotMethod = r.Method
		gotURL = r.URL.String()
		gotAPIKey = r.Header.Get("api-key")
		gotContentType = r.Header.Get("Content-Type")
		gotAccept = r.Header.Get("Accept")
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("erro ao decodificar payload: %v", err)
		}
		return jsonResponse(http.StatusCreated, `{"messageId":"<msg-123@brevo>"}`), nil
	})

	err := mailer.SendEmail("cliente@example.com", "Assunto Teste", "Texto simples", "<p>HTML</p>")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if gotMethod != http.MethodPost || gotURL != "https://api.brevo.test/v3/smtp/email" {
		t.Fatalf("method/url inesperados: %s %s", gotMethod, gotURL)
	}
	if gotAPIKey != "xkeysib-test-key" {
		t.Fatalf("api-key = %q", gotAPIKey)
	}
	if gotContentType != "application/json" || gotAccept != "application/json" {
		t.Fatalf("headers invalidos: Content-Type=%q Accept=%q", gotContentType, gotAccept)
	}
	if gotPayload.Sender.Name != defaultSenderName || gotPayload.Sender.Email != "noreply@nero.com" {
		t.Fatalf("sender inesperado: %+v", gotPayload.Sender)
	}
	if len(gotPayload.To) != 1 || gotPayload.To[0].Email != "cliente@example.com" {
		t.Fatalf("destinatario inesperado: %+v", gotPayload.To)
	}
	if gotPayload.Subject != "Assunto Teste" || gotPayload.TextContent != "Texto simples" || gotPayload.HTMLContent != "<p>HTML</p>" {
		t.Fatalf("conteudo inesperado: %+v", gotPayload)
	}
}

func TestBrevoMailer_SenderWithDisplayNameAndTemplates(t *testing.T) {
	var payloads []brevoEmailRequest

	mailer := newTestBrevoMailer("xkeysib-test-key", "NERO Store <contato@nero.com>", func(r *http.Request) (*http.Response, error) {
		var p brevoEmailRequest
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		payloads = append(payloads, p)
		return jsonResponse(http.StatusCreated, `{"messageId":"<ok>"}`), nil
	})

	if err := mailer.SendVerificationCode("user@example.com", "123456", "Verifique seu e-mail"); err != nil {
		t.Fatalf("SendVerificationCode falhou: %v", err)
	}
	if err := mailer.SendSuspiciousLogin("user@example.com"); err != nil {
		t.Fatalf("SendSuspiciousLogin falhou: %v", err)
	}
	if err := mailer.SendOrderNotification(ports.OrderNotificationEvent{
		EventType:  ports.EventOrderPaymentApproved,
		OrderId:    "12345678-aaaa-bbbb-cccc-1234567890ab",
		OrderTotal: 199.90,
		UserEmail:  "user@example.com",
		UserName:   "Daniel",
	}); err != nil {
		t.Fatalf("SendOrderNotification falhou: %v", err)
	}

	if len(payloads) != 3 {
		t.Fatalf("esperava 3 envios, got %d", len(payloads))
	}
	if payloads[0].Sender.Name != "NERO Store" || payloads[0].Sender.Email != "contato@nero.com" {
		t.Fatalf("sender com nome nao foi preservado: %+v", payloads[0].Sender)
	}
	if !strings.Contains(payloads[0].HTMLContent, "123456") || !strings.Contains(payloads[0].TextContent, "123456") {
		t.Fatalf("codigo de verificacao ausente no email")
	}
	if !strings.Contains(payloads[1].Subject, "suspeita") || !strings.Contains(payloads[1].HTMLContent, "NERO") {
		t.Fatalf("template de login suspeito invalido: subject=%q", payloads[1].Subject)
	}
	if !strings.Contains(payloads[2].Subject, "Pagamento aprovado") || !strings.Contains(payloads[2].HTMLContent, "199.90") {
		t.Fatalf("template de pedido invalido: subject=%q", payloads[2].Subject)
	}
}

func TestBrevoMailer_Errors(t *testing.T) {
	t.Run("Missing API key does not call HTTP", func(t *testing.T) {
		called := false
		mailer := newTestBrevoMailer("", "noreply@nero.com", func(r *http.Request) (*http.Response, error) {
			called = true
			return jsonResponse(http.StatusCreated, `{}`), nil
		})
		if err := mailer.SendEmail("a@b.com", "sub", "txt", ""); err == nil {
			t.Fatal("esperava erro sem API key")
		}
		if called {
			t.Fatal("nao deveria fazer chamada HTTP sem API key")
		}
	})

	t.Run("Invalid sender returns error", func(t *testing.T) {
		mailer := newTestBrevoMailer("key", "", func(r *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusCreated, `{}`), nil
		})
		if err := mailer.SendEmail("a@b.com", "sub", "txt", ""); err == nil || !strings.Contains(err.Error(), "remetente") {
			t.Fatalf("esperava erro de remetente invalido, got %v", err)
		}
	})

	t.Run("Brevo HTTP 401 returns parsed message", func(t *testing.T) {
		mailer := newTestBrevoMailer("bad-key", "noreply@nero.com", func(r *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusUnauthorized, `{"code":"unauthorized","message":"Key not found"}`), nil
		})
		err := mailer.SendEmail("a@b.com", "sub", "txt", "")
		if err == nil || !strings.Contains(err.Error(), "status 401") || !strings.Contains(err.Error(), "Key not found") {
			t.Fatalf("erro inesperado: %v", err)
		}
	})

	t.Run("Transport error is propagated", func(t *testing.T) {
		mailer := newTestBrevoMailer("key", "noreply@nero.com", func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("connection refused")
		})
		if err := mailer.SendEmail("a@b.com", "sub", "txt", ""); err == nil || !strings.Contains(err.Error(), "connection refused") {
			t.Fatalf("erro de rede inesperado: %v", err)
		}
	})
}
