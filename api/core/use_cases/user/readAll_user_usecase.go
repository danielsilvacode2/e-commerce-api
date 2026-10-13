package use_cases

import entities "e-commerce/core/entities/user"

func ReadAllUsers(userRepository entities.UserInterface) ([]entities.User, error){

	listUsers, err := userRepository.ReadAllUsers() 

	if err != nil {
		return nil, err
	}

 	return 	listUsers, err
}