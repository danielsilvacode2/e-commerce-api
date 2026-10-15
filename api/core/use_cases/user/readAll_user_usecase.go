package use_cases

import entities "e-commerce/core/entities/user"


type ReadAllUsersUseCase struct{
  	repository entities.UserRepository
}

func NewReadAllUsersUseCase(repository entities.UserRepository) *ReadAllUsersUseCase {
	return &ReadAllUsersUseCase{
		repository: repository,
	}
}


func (useCase *ReadAllUsersUseCase) UseCase() ([]entities.User, error){

	listUsers, err := useCase.repository.ReadAllUsers() 

	if err != nil {
		return nil, err
	}

 	return 	listUsers, err
}