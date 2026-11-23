package ports

type EmailService interface {
	SendEmail(string, string, string, string) error
	SendVerificationCode(to, code string, textType string) error
	SendSuspiciousLogin(to string) error
	SendOrderNotification(event OrderNotificationEvent) error
}
