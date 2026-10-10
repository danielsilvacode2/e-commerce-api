package use_cases

import "e-commerce/core/entities/user"

func DeleteUser(ID string, userRepository entities.UserInterface) (error) {

	err := userRepository.DeleteUser(ID)
	if err != nil {
		return err
	}

	return nil
}