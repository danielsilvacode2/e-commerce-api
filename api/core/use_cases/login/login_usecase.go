package use_cases

import (
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/user"
	"e-commerce/core/ports"
)



func LoginUseCase(email string, password string, repository entities.UserRepository,passwordService ports.PasswordService) ( *entities.User ,error){

	user, err := repository.ReadUserByEmail(email)

	if err != nil {
		return nil, err
	}

	err = passwordService.ComparePassword(user.Password, password)

	if err != nil{
		return nil, apperrors.ErrUnauthorized
	}	
	
	return user, nil
}

