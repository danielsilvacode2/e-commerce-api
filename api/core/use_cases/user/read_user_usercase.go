package use_cases

import "e-commerce/core/entities/user"

type ReadUserByIdUseCase struct {
	repository entities.UserRepository
}

func NewReadUserByIdUseCase(repository entities.UserRepository) *ReadUserByIdUseCase {
	return &ReadUserByIdUseCase{
		repository: repository,
	}
}

func(useCase ReadUserByIdUseCase) UseCase(ID string) (*entities.User, error) {

	user, err := useCase.repository.ReadUserById(ID)
	if err != nil {
		return nil, err
	}

	return user, nil
}
