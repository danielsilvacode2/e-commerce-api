package use_cases

import (
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/user"
	"strings"
)

type UpdateUseUseCase struct {
	repository entities.UserRepository
}

func NewUpdateUseUseCase(repository entities.UserRepository) *UpdateUseUseCase {
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

// UpdateProfile atualiza apenas os dados cadastrais do usuario preservando estado de verificacao e codigos de seguranca.
func (useCase *UpdateUseUseCase) UpdateProfile(name string, email string, id string) error {
	existing, err := useCase.repository.ReadUserById(id)
	if err != nil {
		return err
	}

	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	if normalizedEmail != "" && normalizedEmail != strings.ToLower(strings.TrimSpace(existing.Email)) {
		return apperrors.ErrForbidden
	}

	existing.Name = strings.TrimSpace(name)

	return useCase.repository.UpdateUser(*existing, id)
}
