package use_cases

import (
	"e-commerce/core/entities/user"
	"e-commerce/core/ports"

	"github.com/google/uuid"
)

type CreateUserUseCase struct {
	repository      entities.UserRepository
	passwordService ports.PasswordService
}

func NewCreateUserUseCase(repository entities.UserRepository, passwordService ports.PasswordService) *CreateUserUseCase {
	return &CreateUserUseCase{
		repository:      repository,
		passwordService: passwordService,
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

	err = useCase.repository.CreateUser(user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
