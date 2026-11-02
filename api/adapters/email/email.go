package email

import (
	"e-commerce/core/ports"

	"fmt"
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

func (m *Mailer) SendEmail(to string, msg string, title string) error {
	e := email.NewEmail()
	e.From = m.From // ex: "NERO <no-reply@nero.com>"
	e.To = []string{to}
	e.Subject = title
	e.Text = []byte(msg)

	addr := fmt.Sprintf("%s:%s", m.Host, m.Port)
	auth := smtp.PlainAuth("", m.User, m.Password, m.Host)

	if err := e.Send(addr, auth); err != nil {
		return fmt.Errorf("falha ao enviar email: %w", err)
	}
	return nil
}
