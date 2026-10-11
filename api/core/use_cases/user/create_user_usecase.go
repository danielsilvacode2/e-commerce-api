package use_cases

import (
	"e-commerce/adapters/security"
	"e-commerce/core/entities/user"
	"github.com/google/uuid"
)

func CreateUser(user entities.User, userRepository entities.UserInterface) (*entities.User, error) {

	user.ID = uuid.New()
	passwordHash, err := security.HashPassword(user.Password)

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
