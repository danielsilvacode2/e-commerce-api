package controller

import (
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/controller/login/dto"
	"e-commerce/adapters/security/auth"
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/user"
	"e-commerce/core/ports"
	usecases "e-commerce/core/use_cases/login"
	"net/http"

	"github.com/gin-gonic/gin"
)

type LoginController struct {
	repository      entities.UserRepository
	passwordService ports.PasswordService
}

func NewLoginController(repository entities.UserRepository, passwordService ports.PasswordService) *LoginController {
	return &LoginController{
		repository:      repository,
		passwordService: passwordService,
	}
}

func (l *LoginController) Login(c *gin.Context) {

	var loginRequest dto.LoginRequestDTO

	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	user, err := usecases.LoginUseCase(loginRequest.Email, loginRequest.Password, l.repository, l.passwordService)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Role)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	loginResponse := dto.LoginResponseDTO{
		WebToken: token,
		UserRole: user.Role,
	}

	helpers.HelperResponse(loginResponse, http.StatusOK, "login authorized", c)

}
