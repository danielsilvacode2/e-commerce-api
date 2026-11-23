package use_cases

import (
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/user"
	"e-commerce/core/ports"
	"time"
)

type VerifyUserCodePasswordUseCase struct {
	repository  entities.UserRepository
	readUser    ReadUserByEmailUseCase
	hashService ports.HashService
}

func NewVerifyUserCodePasswordUseCase ( repository entities.UserRepository, readUser ReadUserByEmailUseCase, hashService ports.HashService) *VerifyUserCodePasswordUseCase{
	return &VerifyUserCodePasswordUseCase{
		hashService: hashService,
		readUser: readUser,
		repository: repository,
	}
}

func (useCase *VerifyUserCodePasswordUseCase) UseCase(email string, code string, newPass string) error {

	user, err := useCase.readUser.UseCase(email)
	if err != nil {
		return err
	}

	if user.PasswordResetCode == "" || user.PasswordResetExpires.Before(time.Now()) {
		return apperrors.ErrUnprocessableEntity
	}

	if err = useCase.hashService.CompareHash(user.PasswordResetCode, code); err != nil {
		return apperrors.ErrUnauthorized
	}

	hashPass, err := useCase.hashService.HashGenerate(newPass)
	if err != nil {
		return err
	}

	user.Password = hashPass
	user.PasswordResetCode = ""
	user.PasswordResetExpires = time.Date(1970, 1, 1, 0, 0, 1, 0, time.UTC)

	if err := useCase.repository.UpdatePasswordUser(*user, user.ID.String()); err != nil {
		return err
	}

	return nil

}
