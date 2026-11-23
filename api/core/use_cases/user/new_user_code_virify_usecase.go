package use_cases

import (
	"e-commerce/core/entities/user"
	"e-commerce/core/ports"
	"time"
)

type NewUserCodeUseCase struct {
	repository   entities.UserRepository
	updateUser   *UpdateUseUseCase
	readByEmail  *ReadUserByEmailUseCase
	emailService ports.EmailService
	hashService  ports.HashService
}

func NewNewUserCodeUseCase(
	repository entities.UserRepository,
	updateUser *UpdateUseUseCase,
	readByEmail *ReadUserByEmailUseCase,
	emailService ports.EmailService,
	hashService ports.HashService,
) *NewUserCodeUseCase {
	return &NewUserCodeUseCase{
		repository:   repository,
		updateUser:   updateUser,
		readByEmail:  readByEmail,
		emailService: emailService,
		hashService:  hashService,
	}
}

func (useCase *NewUserCodeUseCase) UseCase(email string) error {
	user, err := useCase.readByEmail.UseCase(email)
	if err != nil {
		return err
	}

	codeFortm, err := generateSecureVerificationCode()
	if err != nil {
		return err
	}

	codeHash, err := useCase.hashService.HashGenerate(codeFortm)

	if err != nil {
		return err
	}

	user.VerificationCode = codeHash

	timeExpiresCode := time.Now().Add(15 * time.Minute)

	user.CodeExpires = timeExpiresCode

	if err = useCase.updateUser.UseCase(*user, user.ID.String()); err != nil {
		return err
	}

	go func() {
		useCase.emailService.SendVerificationCode(user.Email, codeFortm ,"Verificar email")
	}()
	return nil
}
