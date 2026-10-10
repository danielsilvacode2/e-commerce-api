package use_cases

import  "e-commerce/core/entities/user"

func ReadUser( ID string, userRepository entities.UserInterface) ( *entities.User, error) {
	
	user, err := userRepository.ReadUser(ID)
	if err != nil {
		return nil, err
	}

	return user, nil
}