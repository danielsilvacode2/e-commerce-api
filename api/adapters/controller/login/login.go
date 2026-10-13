package controller

import (
	"e-commerce/adapters/controller/apperrors"
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/controller/login/dto"
	"e-commerce/adapters/security/auth"
	entities "e-commerce/core/entities/user"
	usecases "e-commerce/core/use_cases/login"
	"net/http"

	"github.com/gin-gonic/gin"
)

type LoginController struct {
	repository entities.UserInterface
}

func NewLoginController(repository entities.UserInterface) *LoginController {
	return &LoginController{
		repository: repository,
	}
}

func (l *LoginController) Login(c *gin.Context) {

	var loginRequest dto.LoginRequestDTO

	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	user, err := usecases.LoginUseCase(loginRequest.Email, loginRequest.Password, l.repository)

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
	 }

	 helpers.HelperResponse(loginResponse, http.StatusOK, "login authorized", c)

}
