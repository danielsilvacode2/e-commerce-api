package use_cases

import (
	entities "e-commerce/core/entities/user"
	"e-commerce/core/ports"
)

type ResetPasswordUseCase struct {
	repository entities.UserRepository
	hashService ports.HashService
}