package use_cases

import (
	"e-commerce/adapters/controller/apperrors"
	"e-commerce/adapters/security"
	entities "e-commerce/core/entities/user"
)

func LoginUseCase(email string, password string, repository entities.UserInterface) ( *entities.User ,error){

	user, err := repository.ReadUserByEmail(email)

	if err != nil {
		return nil, err
	}

	err = security.ComparePassword(user.Password, password)

	if err != nil{
		return nil, apperrors.ErrUnauthorized
	}	
	
	return user, nil
}

