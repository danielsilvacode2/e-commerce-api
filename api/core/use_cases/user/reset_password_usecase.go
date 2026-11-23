package use_cases

import (
	entities "e-commerce/core/entities/user"
	"e-commerce/core/ports"
	"log"
	"time"
)

type ResetPasswordUseCase struct {
	repository   entities.UserRepository
	hashService  ports.HashService
	emailService ports.EmailService
	readUser     ReadUserByEmailUseCase
}

func NewResetPasswordUseCase(repository entities.UserRepository, readUser ReadUserByEmailUseCase, hashService ports.HashService, emailService ports.EmailService) *ResetPasswordUseCase {
	return &ResetPasswordUseCase{
		hashService:  hashService,
		readUser:     readUser,
		repository:   repository,
		emailService: emailService,
	}
}

func (useCase *ResetPasswordUseCase) UseCase(email string) error {

	user, err := useCase.readUser.UseCase(email)
	if err != nil {
		return err
	}

	codeFortm, err := generateSecureVerificationCode()
	if err != nil {
		return err
	}

	codeHashPassword, err := useCase.hashService.HashGenerate(codeFortm)
	if err != nil {
		return err
	}
	user.PasswordResetCode = codeHashPassword

	timeExpiresCode := time.Now().Add(5 * time.Minute)
	user.PasswordResetExpires = timeExpiresCode

	if err = useCase.repository.UpdateUser(*user, user.ID.String()); err != nil {
		return err
	}

	go func() {
		if err := useCase.emailService.SendVerificationCode(user.Email, codeFortm,"Mudar senha"); err != nil {
			log.Printf("[ERROR] Falha ao enviar email para userID=%s userEmail=%s erro=(%v)", user.ID, user.Email, err)
		} else {
			log.Printf("[INFO] Email enviado com sucesso para userID=%s userEmail=%s", user.ID, user.Email)
		}
	}()

	return nil
}
