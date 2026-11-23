package use_cases

import (
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/user"
	"e-commerce/core/ports"
	"time"
)

type VerifyUserCodeUseCase struct {
	repository      entities.UserRepository
	readUserByEmail *ReadUserByEmailUseCase
	hashService     ports.HashService
}

func NewVerifyUserCodeUseCase(repository entities.UserRepository, readUserByEmail *ReadUserByEmailUseCase, hashService ports.HashService) *VerifyUserCodeUseCase {
	return &VerifyUserCodeUseCase{
		repository:      repository,
		readUserByEmail: readUserByEmail,
		hashService:     hashService,
	}
}

func (useCase *VerifyUserCodeUseCase) UseCase(email string, verificationCode string) error {

	user, err := useCase.readUserByEmail.UseCase(email)

	if err != nil {
		return err
	}

	if user.VerificationCode == "" || user.CodeExpires.Before(time.Now()) {
		return apperrors.ErrUnprocessableEntity
	}

	if err = useCase.hashService.CompareHash(user.VerificationCode, verificationCode); err != nil {
		return apperrors.ErrUnauthorized
	}

	user.IsVerified = true
	user.VerificationCode = ""
	user.CodeExpires = time.Date(1970, 1, 1, 0, 0, 1, 0, time.UTC)

	if err := useCase.repository.UpdateUser(*user, user.ID.String()); err != nil {
		return err
	}

	return nil

}
