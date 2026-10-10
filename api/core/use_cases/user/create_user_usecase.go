package use_cases

import  (
	"e-commerce/core/entities/user"
	"e-commerce/adapters/security"
	"github.com/google/uuid"
)

func CreateUser(user entities.User, userRepository entities.UserInterface) (error) {

	user.ID = uuid.New()	
	passwordHash, err := security.HashPassword(user.Password)

	if err != nil {
		return err
	}

	user.Password = passwordHash

	err = userRepository.CreateUSer(user)
	if err != nil {
		return err
	}

	return nil
}