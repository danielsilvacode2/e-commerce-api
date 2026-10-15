package use_cases

import entities "e-commerce/core/entities/user"


type UpdateUseUseCase struct{
  	repository entities.UserRepository
}

func NewUpdateUseUseCase(repository entities.UserRepository) *UpdateUseUseCase{
	return &UpdateUseUseCase{
		repository: repository,
	}
}


func (useCase *UpdateUseUseCase) UseCase(user entities.User, id string) error {
	user.NormalizeData()

	if err := useCase.repository.UpdateUser(user, id); err != nil {
		return err
	}

	return nil
}
