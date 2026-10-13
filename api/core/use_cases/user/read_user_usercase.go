package use_cases

import  "e-commerce/core/entities/user"

func ReadUserById( ID string, userRepository entities.UserInterface) ( *entities.User, error) {
	
	user, err := userRepository.ReadUserById(ID)
	if err != nil {
		return nil, err
	}

	return user, nil
}