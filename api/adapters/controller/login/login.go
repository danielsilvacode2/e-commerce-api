package login

import (
	entities "e-commerce/core/entities/user"
)

type LoginController struct {
	repository entities.UserInterface
}

func NewLoginController( repository entities.UserInterface)*LoginController{
	return &LoginController{
		repository: repository,
	}
}
