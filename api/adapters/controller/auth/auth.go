package controller

import (
	"e-commerce/adapters/controller/auth/dto"
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/security/auth"
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/user"
	"e-commerce/core/ports"
	usecases "e-commerce/core/use_cases/auth"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	repository  entities.UserRepository
	hashService ports.HashService
}

func NewAuthController(repository entities.UserRepository, hashService ports.HashService) *AuthController {
	return &AuthController{
		repository:  repository,
		hashService: hashService,
	}
}

func (a *AuthController) Login(c *gin.Context) {

	var loginRequest dto.LoginRequestDTO

	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	user, err := usecases.LoginUseCase(loginRequest.Email, loginRequest.Password, a.repository, a.hashService)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	if !user.IsVerified {
		helpers.HelperErrResponse(nil, c, apperrors.ErrUnauthorized)
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

	c.SetSameSite(http.SameSiteLaxMode)

	c.SetCookie(
		"token", // nome (o mesmo que o getToken lê com c.Cookie("token"))
		token,   // valor
		2*60*60, // duração em segundos (2h, igual ao JWT)
		"/",     // path
		"",      // domínio (vazio = só o host atual)
		false,   // secure: false em dev local (HTTP), true em produção
		true,    // httpOnly
	)

	helpers.HelperResponse(loginResponse, http.StatusOK, "login authorized", c)

}
func (a *AuthController) Logout(c *gin.Context) {

	c.SetCookie(
		"token",     // Nome do cookie usado no login
		"",          // Valor vazio para apagar
		-1,          // MaxAge < 0 remove o cookie imediatamente
		"/",         // Path
		"localhost", // Domínio (ajuste para seu ambiente de produção)
		false,       // Secure (deve ser true em produção com HTTPS)
		true,        // HttpOnly (protege contra XSS)
	)

	helpers.HelperResponse(nil, http.StatusOK, "login realizado com sucesso", c)
}
