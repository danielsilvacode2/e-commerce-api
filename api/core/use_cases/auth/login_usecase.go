package use_cases

import (
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/user"
	"e-commerce/core/ports"
	"errors"
)

// dummyBcryptHash e usado quando o e-mail nao existe para equalizar o tempo de resposta (mitigacao de timing attack).
const dummyBcryptHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

func LoginUseCase(email string, password string, repository entities.UserRepository, hashService ports.HashService) (*entities.User, error) {

	user, err := repository.ReadUserByEmail(email)

	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			_ = hashService.CompareHash(dummyBcryptHash, password)
			return nil, apperrors.ErrUnauthorized
		}
		return nil, err
	}

	err = hashService.CompareHash(user.Password, password)

	if err != nil {
		return nil, apperrors.ErrUnauthorized
	}

	return user, nil
}

