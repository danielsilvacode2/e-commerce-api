package use_cases

import entities "e-commerce/core/entities/user"

type ReadUserByEmailUseCase struct {
	repository entities.UserRepository
}

func NewReadUserByEmailUseCase(repository entities.UserRepository) *ReadUserByEmailUseCase {
	return &ReadUserByEmailUseCase{
		repository: repository,
	}
}

func (useCase *ReadUserByEmailUseCase) UseCase(email string) (*entities.User, error) {
	user, err := useCase.repository.ReadUserByEmail(email)

	if err != nil {
		return nil, err
	}

	return user, nil
}
