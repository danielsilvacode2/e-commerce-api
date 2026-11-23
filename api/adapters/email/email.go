package email

import (
	"e-commerce/core/ports"

	"fmt"
	"html"
	"net/smtp"

	"github.com/jordan-wright/email"
)

type Mailer struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

func NewMailer(host, port, user, password, from string) ports.EmailService {
	return &Mailer{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		From:     from,
	}
}

func (m *Mailer) SendEmail(to, title, text, html string) error {

	e := email.NewEmail()
	e.From = m.From
	e.To = []string{to}
	e.Subject = title
	e.Text = []byte(text)

	if html != "" {
		e.HTML = []byte(html)
	}

	addr := fmt.Sprintf("%s:%s", m.Host, m.Port)
	auth := smtp.PlainAuth("", m.User, m.Password, m.Host)

	if err := e.Send(addr, auth); err != nil {
		return fmt.Errorf("falha ao enviar email: %w", err)
	}
	return nil
}



func (m *Mailer) SendVerificationCode(to, code string, textType string) error {
	htmlBody := fmt.Sprintf(verificationHTML, html.EscapeString(textType), html.EscapeString(code))
	text := "Seu código de verificação NERO: " + code
	return m.SendEmail(to, "NERO — Seu código de verificação", text, htmlBody)
}


func (m *Mailer) SendSuspiciousLogin(to string) error {
	text := "Tentativa de acesso suspeita na sua conta NERO. Se não foi você, troque sua senha."
	return m.SendEmail(to, "NERO — Tentativa de acesso suspeita", text, accountExistsHTML)
}

func (m *Mailer) SendOrderNotification(event ports.OrderNotificationEvent) error {
	subject, text, htmlBody := buildOrderNotificationEmail(event)
	return m.SendEmail(event.UserEmail, subject, text, htmlBody)
}