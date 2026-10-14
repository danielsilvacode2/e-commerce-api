package use_cases

import (
	"e-commerce/core/entities/user"
	"e-commerce/core/ports"

	"github.com/google/uuid"
)

func CreateUser(user entities.User, userRepository entities.UserRepository, passwordService ports.PasswordService) (*entities.User, error) {

	user.ID = uuid.New()
	passwordHash, err := passwordService.HashPassword(user.Password)

	if err != nil {
		return nil, err
	}

	user.Password = passwordHash

	if user.Role == "" {
		user.Role = "user"
	}

	user.NormalizeData()

	err = userRepository.CreateUser(user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
