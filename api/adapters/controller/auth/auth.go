package controller

import (
	"e-commerce/adapters/controller/auth/dto"
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/security/auth"
	"e-commerce/config"
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

func cookieSameSite() http.SameSite {
	if config.CookieSecure {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

func NewAuthController(repository entities.UserRepository, hashService ports.HashService) *AuthController {
	return &AuthController{
		repository:  repository,
		hashService: hashService,
	}
}

// Login autentica o usuario e define o cookie HTTP-Only "token".
// @Summary      Realizar login
// @Description  Autentica um usuario verificado por email e senha e define o cookie HTTP-Only "token" (JWT valido por 2 horas).
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.LoginRequestDTO  true  "Credenciais de acesso"
// @Success      200      {object}  helpers.SwaggerDataResponse[dto.LoginResponseDTO]
// @Failure      400      {object}  helpers.SwaggerMessageResponse
// @Failure      401      {object}  helpers.SwaggerMessageResponse
// @Failure      429      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /login [post]
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
		UserRole: user.Role,
	}

	c.SetSameSite(cookieSameSite())

	c.SetCookie(
		"token",             // nome (o mesmo que o getToken lê com c.Cookie("token"))
		token,               // valor
		2*60*60,             // duração em segundos (2h, igual ao JWT)
		"/",                 // path
		config.CookieDomain, // domínio (vazio = só o host atual)
		config.CookieSecure, // secure: true em produção com HTTPS
		true,                // httpOnly
	)

	helpers.HelperResponse(loginResponse, http.StatusOK, "login authorized", c)

}
// Logout revoga o token atual e remove o cookie de autenticacao.
// @Summary      Realizar logout
// @Description  Revoga o JWT atual (se presente e valido) e remove o cookie HTTP-Only "token".
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  helpers.SwaggerMessageResponse
// @Router       /logout [post]
func (a *AuthController) Logout(c *gin.Context) {
	if claims, err := auth.ValidateToken(c); err == nil && claims != nil {
		if claims.ExpiresAt != nil {
			auth.RevokeToken(claims.ID, claims.ExpiresAt.Time)
		} else {
			auth.RevokeUserTokensBefore(claims.Id.String())
		}
	}

	c.SetSameSite(cookieSameSite())

	c.SetCookie(
		"token",             // Nome do cookie usado no login
		"",                  // Valor vazio para apagar
		-1,                  // MaxAge < 0 remove o cookie imediatamente
		"/",                 // Path
		config.CookieDomain, // Mesmo domínio utilizado no login
		config.CookieSecure, // Secure (true em produção com HTTPS)
		true,                // HttpOnly (protege contra XSS)
	)

	helpers.HelperResponse(nil, http.StatusOK, "logout realizado com sucesso", c)
}
