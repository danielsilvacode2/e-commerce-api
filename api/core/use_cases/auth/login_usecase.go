package use_cases

import (
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/user"
	"e-commerce/core/ports"
)



func LoginUseCase(email string, password string, repository entities.UserRepository,hashService ports.HashService) ( *entities.User ,error){

	user, err := repository.ReadUserByEmail(email)

	if err != nil {
		return nil, err
	}

	err = hashService.CompareHash(user.Password, password)

	if err != nil{
		return nil, apperrors.ErrUnauthorized
	}	
	
	return user, nil
}

