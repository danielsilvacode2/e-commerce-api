package use_cases

import (
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/user"
	"time"
)

type VerifyUserCodeUseCase struct {
	repository      entities.UserRepository
	readUserByEmail *ReadUserByEmailUseCase
}

func NewVerifyUserCodeUseCase(repository entities.UserRepository, readUserByEmail *ReadUserByEmailUseCase) *VerifyUserCodeUseCase {
	return &VerifyUserCodeUseCase{
		repository:      repository,
		readUserByEmail: readUserByEmail,
	}
}

func (useCase *VerifyUserCodeUseCase) UseCase(email string, verificationCode string) error {

	user, err := useCase.readUserByEmail.UseCase(email)

	if err != nil {
		return err
	}

	if user.CodeExpires.Before(time.Now()) {
		return apperrors.ErrUnprocessableEntity
	}

	if user.VerificationCode != verificationCode {
		return apperrors.ErrUnprocessableEntity
	}

	user.IsVerified = true

	if err := useCase.repository.UpdateUser(*user, user.ID.String()); err != nil {
		return err
	}

	return nil

}
