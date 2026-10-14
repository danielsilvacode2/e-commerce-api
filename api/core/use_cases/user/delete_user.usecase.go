package use_cases

import "e-commerce/core/entities/user"

func DeleteUser(ID string, userRepository entities.UserRepository) (error) {

	err := userRepository.DeleteUserById(ID)
	if err != nil {
		return err
	}

	return nil
}