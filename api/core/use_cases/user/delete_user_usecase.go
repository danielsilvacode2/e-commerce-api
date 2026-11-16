package use_cases

import "e-commerce/core/entities/user"

type DeleteUserUseCase struct {
	repository entities.UserRepository
}

func NewDeleteUserUseCase(repository entities.UserRepository) *DeleteUserUseCase {
	return &DeleteUserUseCase{
		repository: repository,
	}
}

func (useCase DeleteUserUseCase) UseCase(ID string) error {

	err := useCase.repository.DeleteUserById(ID)
	if err != nil {
		return err
	}

	return nil
}
