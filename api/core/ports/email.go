package ports

type EmailService interface {
	SendEmail(string, string, string) error
}
