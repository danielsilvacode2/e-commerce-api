package use_cases

import entities "e-commerce/core/entities/user"

func UpdateUser(user entities.User, id string ,repository entities.UserRepository) error {
	user.NormalizeData()

	if err := repository.UpdateUser(user, id); err != nil {
		return err
	}

	return nil
}
