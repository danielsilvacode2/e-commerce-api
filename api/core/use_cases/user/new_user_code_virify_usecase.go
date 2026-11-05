package use_cases

import (
	"e-commerce/core/entities/user"
	"e-commerce/core/ports"
	"fmt"
	"math/rand/v2"
	"time"
)

type NewUserCodeUseCase struct {
	repository   entities.UserRepository
	updateUser   *UpdateUseUseCase
	readByEmail  *ReadUserByEmailUseCase
	emailService ports.EmailService
}

func NewNewUserCodeUseCase(repository entities.UserRepository, updateUser *UpdateUseUseCase, readByEmail *ReadUserByEmailUseCase, emailService ports.EmailService) *NewUserCodeUseCase {
	return &NewUserCodeUseCase{
		repository:   repository,
		updateUser:   updateUser,
		readByEmail:  readByEmail,
		emailService: emailService,
	}
}

func (useCase *NewUserCodeUseCase) UseCase(email string) error {
	user, err := useCase.readByEmail.UseCase(email)
	if err != nil {
		return  err
	}

	code := rand.Int64N(900000)
	user.VerificationCode = fmt.Sprintf("%06d", code)

	timeExpiresCode := time.Now().Add(15 * time.Minute)

	user.CodeExpires = timeExpiresCode

	if err = useCase.updateUser.UseCase(*user, user.ID.String()); err != nil {
		return  err
	}

	go func() {
		useCase.emailService.SendEmail(user.Email, "Olá! Seu novo codigo de verificacao: "+user.VerificationCode, "Codigo de verificacao NERO")
	}()
	return nil
}
