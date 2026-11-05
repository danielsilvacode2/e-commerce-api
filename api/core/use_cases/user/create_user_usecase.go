package use_cases

import (
	"e-commerce/core/entities/user"
	"e-commerce/core/ports"
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
)

type CreateUserUseCase struct {
	repository      entities.UserRepository
	passwordService ports.PasswordService
	emailService    ports.EmailService
}

func NewCreateUserUseCase(repository entities.UserRepository, passwordService ports.PasswordService, emailService ports.EmailService) *CreateUserUseCase {
	return &CreateUserUseCase{
		repository:      repository,
		passwordService: passwordService,
		emailService:    emailService,
	}
}

func (useCase CreateUserUseCase) UseCase(user entities.User) (*entities.User, error) {

	user.ID = uuid.New()
	passwordHash, err := useCase.passwordService.HashPassword(user.Password)

	if err != nil {
		return nil, err
	}

	user.Password = passwordHash

	if user.Role == "" {
		user.Role = "user"
	}

	user.NormalizeData()

	code := rand.Int64N(900000)
	user.VerificationCode = fmt.Sprintf("%06d", code)

	timeExpiresCode := time.Now().Add(15 * time.Minute)

	user.CodeExpires = timeExpiresCode

	err = useCase.repository.CreateUser(user)
	if err != nil {
		return nil, err
	}

	go func() {

		if err := useCase.emailService.SendEmail(user.Email, "Olá! Seu codigo de verificacao: "+user.VerificationCode, "Codigo de verificacao NERO"); err != nil {
			log.Printf("ERRO AO ENVIAR EMAIL: %s - ERRO: %s", user.Email, err.Error())
		} else {
			log.Printf("EMAIL ENVIADO COM SUCESSO PARA: %s", user.Email)
		}

	}()

	return &user, nil
}
