package use_cases

import (
	"e-commerce/core/entities/user"
	"fmt"
	"math/rand/v2"
	"time"
)

type NewUserCodeUseCase struct {
	repository  entities.UserRepository
	updateUser  *UpdateUseUseCase
	readByEmail *ReadUserByEmailUseCase
}

func NewNewUserCodeUseCase(repository entities.UserRepository, updateUser *UpdateUseUseCase, readByEmail *ReadUserByEmailUseCase) *NewUserCodeUseCase {
	return &NewUserCodeUseCase{
		repository:  repository,
		updateUser:  updateUser,
		readByEmail: readByEmail,
	}
}

func (useCase *NewUserCodeUseCase) UseCase(email string) (string, error) {
	user, err := useCase.readByEmail.UseCase(email)
	if err != nil {
		return "", err
	}

	code := rand.Int64N(900000)
	user.VerificationCode = fmt.Sprintf("%06d", code)

	timeExpiresCode := time.Now().Add(15 * time.Minute)

	user.CodeExpires = timeExpiresCode

	 if err = useCase.updateUser.UseCase(*user, user.ID.String()); err != nil {
		return "", err
	 }

	 return user.VerificationCode, nil
}
