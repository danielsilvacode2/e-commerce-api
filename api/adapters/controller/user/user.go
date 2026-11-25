package controller

import (
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/controller/user/dto"
	"e-commerce/adapters/controller/user/mapper"
	"e-commerce/adapters/security/auth"
	"e-commerce/core/apperrors"
	"e-commerce/core/ports"

	use_cases "e-commerce/core/use_cases/user"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	createUser             *use_cases.CreateUserUseCase
	deleteUser             *use_cases.DeleteUserUseCase
	readAllUsers           *use_cases.ReadAllUsersUseCase
	updateUser             *use_cases.UpdateUseUseCase
	readUserById           *use_cases.ReadUserByIdUseCase
	readUserByEmail        *use_cases.ReadUserByEmailUseCase
	resetPassword          *use_cases.ResetPasswordUseCase
	verifyUserCodePassword *use_cases.VerifyUserCodePasswordUseCase
	verifyUserCode         *use_cases.VerifyUserCodeUseCase
	newUserCode            *use_cases.NewUserCodeUseCase
}

func NewUserController(
	createUser *use_cases.CreateUserUseCase,
	deleteUser *use_cases.DeleteUserUseCase,
	readAllUsers *use_cases.ReadAllUsersUseCase,
	updateUser *use_cases.UpdateUseUseCase,
	readUserById *use_cases.ReadUserByIdUseCase,
	verifyUserCode *use_cases.VerifyUserCodeUseCase,
	emailService ports.EmailService,
	readUserByEmail *use_cases.ReadUserByEmailUseCase,
	newUserCode *use_cases.NewUserCodeUseCase,
	resetPassword *use_cases.ResetPasswordUseCase,
	verifyUserCodePassword *use_cases.VerifyUserCodePasswordUseCase,
) *UserController {
	return &UserController{
		createUser:             createUser,
		deleteUser:             deleteUser,
		readAllUsers:           readAllUsers,
		updateUser:             updateUser,
		readUserById:           readUserById,
		verifyUserCodePassword: verifyUserCodePassword,
		resetPassword:          resetPassword,
		verifyUserCode:         verifyUserCode,
		readUserByEmail:        readUserByEmail,
		newUserCode:            newUserCode,
	}
}

// CreateUser registra um usuario na api.
// @Summary      Criar usuario
// @Description  Registra um novo usuario e envia codigo de verificacao por email.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        request  body      dto.UserCreateRequestDTO  true  "Dados de cadastro do usuario"
// @Success      202      {object}  helpers.SwaggerMessageResponse
// @Failure      400      {object}  helpers.SwaggerValidationErrorResponse
// @Failure      429      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /users [post]
func (userCtrl *UserController) CreateUser(c *gin.Context) {
	var userRequestDto dto.UserCreateRequestDTO

	if err := c.ShouldBindJSON(&userRequestDto); err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	if listErrValidator := userRequestDto.Validator(); listErrValidator != nil {
		helpers.HelperErrResponse(listErrValidator, c, apperrors.ErrInvalidInput)
		return
	}

	potentialUser := mapper.DtoCreateRequestToUser(userRequestDto)

	_, err := userCtrl.createUser.UseCase(potentialUser)

	if err != nil {

		if errors.Is(err, apperrors.ErrConflict) {
			helpers.HelperResponse(nil, 202, "We have sent the next instructions to your email.", c)
			return
		}

		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, 202, "We have sent the next instructions to your email.", c)
}

// VerifyEmailMe valida o codigo de verificacao de email do usuario.
// @Summary      Verificar email do usuario
// @Description  Confirma o cadastro do usuario utilizando o codigo enviado por email.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        request  body      dto.UserVerifyCodeRequestDTO  true  "Email e codigo de verificacao"
// @Success      200      {object}  helpers.SwaggerMessageResponse
// @Failure      400      {object}  helpers.SwaggerValidationErrorResponse
// @Failure      401      {object}  helpers.SwaggerMessageResponse
// @Failure      429      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /users/verify [post]
func (userCtrl *UserController) VerifyEmailMe(c *gin.Context) {

	var userVerifyCodeRequest dto.UserVerifyCodeRequestDTO

	if err := c.ShouldBindJSON(&userVerifyCodeRequest); err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	if listErrValidator := userVerifyCodeRequest.Validator(); listErrValidator != nil {
		helpers.HelperErrResponse(listErrValidator, c, apperrors.ErrInvalidInput)
		return
	}

	err := userCtrl.verifyUserCode.UseCase(userVerifyCodeRequest.Email, userVerifyCodeRequest.VerificationCode)

	if err != nil {

		if errors.Is(err, apperrors.ErrNotFound) || errors.Is(err, apperrors.ErrUnprocessableEntity) {
			helpers.HelperErrResponse(nil, c, apperrors.ErrUnauthorized)
			return
		}

		helpers.HelperErrResponse(nil, c, err)
		return

	}

	helpers.HelperResponse(nil, http.StatusOK, "If your email is registered, it has been successfully verified.", c)

}

// NewVerifyCode reenvia um novo codigo de verificacao para o email informado.
// @Summary      Reenviar codigo de verificacao
// @Description  Gera e envia um novo codigo de verificacao para o email cadastrado ainda nao verificado.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        request  body      dto.UserNewVerifyCodeDTO  true  "Email do usuario"
// @Success      200      {object}  helpers.SwaggerMessageResponse
// @Failure      400      {object}  helpers.SwaggerValidationErrorResponse
// @Failure      429      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /users/new-verify-code [post]
func (userCtrl *UserController) NewVerifyCode(c *gin.Context) {
	var userNewVerifyCodeRequest dto.UserNewVerifyCodeDTO

	if err := c.ShouldBindJSON(&userNewVerifyCodeRequest); err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	if listErrValidator := userNewVerifyCodeRequest.Validator(); listErrValidator != nil {
		helpers.HelperErrResponse(listErrValidator, c, apperrors.ErrInvalidInput)
		return
	}

	user, err := userCtrl.readUserByEmail.UseCase(userNewVerifyCodeRequest.Email)

	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			helpers.HelperResponse(nil, http.StatusOK, "If your email is registered, a new verification code has been sent.", c)
			return
		}
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	if user.IsVerified {
		helpers.HelperResponse(nil, http.StatusOK, "If your email is registered, a new verification code has been sent.", c)
		return
	}

	err = userCtrl.newUserCode.UseCase(user.Email)

	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			helpers.HelperResponse(nil, http.StatusOK, "If your email is registered, a new verification code has been sent.", c)
			return
		}
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, http.StatusOK, "If your email is registered, a new verification code has been sent.", c)

}

// ReadUser recebe um Id como parametro e retorna o usuario caso ele exista.
// @Summary      Buscar usuario por ID
// @Description  Retorna os dados de um usuario especifico. Requer perfil admin.
// @Tags         Users
// @Produce      json
// @Security     CookieAuth
// @Param        id   path      string  true  "ID do usuario (UUID)"
// @Success      200  {object}  helpers.SwaggerDataResponse[dto.UserResponseDTO]
// @Failure      401  {object}  helpers.SwaggerMessageResponse
// @Failure      403  {object}  helpers.SwaggerMessageResponse
// @Failure      404  {object}  helpers.SwaggerMessageResponse
// @Failure      500  {object}  helpers.SwaggerMessageResponse
// @Router       /users/{id} [get]
func (userCtrl *UserController) ReadUser(c *gin.Context) {

	id, err := userCtrl.validatorPermissionId(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	user, err := userCtrl.readUserById.UseCase(id)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	userDto := mapper.UserToResponseDTO(*user)
	helpers.HelperResponse(userDto, http.StatusOK, "user found", c)
}

// DeleteUser recebe um Id como parametro e deleta o usuario.
// @Summary      Remover usuario
// @Description  Remove um usuario pelo ID e revoga seus tokens ativos. Requer perfil admin.
// @Tags         Users
// @Produce      json
// @Security     CookieAuth
// @Param        id   path  string  true  "ID do usuario (UUID)"
// @Success      204  "Usuario removido com sucesso"
// @Failure      401  {object}  helpers.SwaggerMessageResponse
// @Failure      403  {object}  helpers.SwaggerMessageResponse
// @Failure      404  {object}  helpers.SwaggerMessageResponse
// @Failure      500  {object}  helpers.SwaggerMessageResponse
// @Router       /users/{id} [delete]
func (userCtrl *UserController) DeleteUser(c *gin.Context) {
	id, err := userCtrl.validatorPermissionId(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	if err = userCtrl.deleteUser.UseCase(id); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	auth.RevokeUserTokensBefore(id)

	helpers.HelperResponse(nil, http.StatusNoContent, "user successfully deleted ", c)
}

// ReadAllUsers lista todos os usuarios cadastrados (somente admin).
// @Summary      Listar usuarios
// @Description  Retorna todos os usuarios cadastrados na plataforma. Requer perfil admin.
// @Tags         Users
// @Produce      json
// @Security     CookieAuth
// @Success      200  {object}  helpers.SwaggerDataResponse[[]dto.UserResponseDTO]
// @Failure      401  {object}  helpers.SwaggerMessageResponse
// @Failure      403  {object}  helpers.SwaggerMessageResponse
// @Failure      500  {object}  helpers.SwaggerMessageResponse
// @Router       /users [get]
func (userCtrl *UserController) ReadAllUsers(c *gin.Context) {

	listUsers, err := userCtrl.readAllUsers.UseCase()

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	var listUserDTO []dto.UserResponseDTO

	if listUsers == nil {
		helpers.HelperResponse(listUserDTO, http.StatusOK, "list is empty", c)
		return
	}

	for _, user := range listUsers {
		userDTO := mapper.UserToResponseDTO(user)
		listUserDTO = append(listUserDTO, userDTO)
	}

	helpers.HelperResponse(listUserDTO, http.StatusOK, "list users", c)

}

// ReadMe retorna o perfil do usuario autenticado.
// @Summary      Obter perfil do usuario logado
// @Description  Retorna os dados do usuario atualmente autenticado via cookie JWT.
// @Tags         Users
// @Produce      json
// @Security     CookieAuth
// @Success      200  {object}  helpers.SwaggerDataResponse[dto.UserResponseDTO]
// @Failure      401  {object}  helpers.SwaggerMessageResponse
// @Failure      404  {object}  helpers.SwaggerMessageResponse
// @Failure      500  {object}  helpers.SwaggerMessageResponse
// @Router       /users/me [get]
func (userCtrl *UserController) ReadMe(c *gin.Context) {

	val, exists := c.Get("claims")

	if !exists {
		helpers.HelperErrResponse(nil, c, apperrors.ErrUnauthorized)
		return
	}

	claims, ok := val.(*auth.Claims)

	if !ok {
		helpers.HelperErrResponse(nil, c, errors.New("Error converting declarations"))
		return
	}

	userId := claims.Id.String()

	user, err := userCtrl.readUserById.UseCase(userId)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	dto := mapper.UserToResponseDTO(*user)

	helpers.HelperResponse(dto, http.StatusOK, "user found", c)

}

// UpdateUser atualiza nome e email do usuario (proprio usuario ou admin).
// @Summary      Atualizar usuario
// @Description  Atualiza o perfil de um usuario. Permitido para o proprio usuario ou admin.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Param        id       path      string                    true  "ID do usuario (UUID)"
// @Param        request  body      dto.UserUpdateRequestDTO  true  "Dados atualizados do perfil"
// @Success      204      "Usuario atualizado com sucesso"
// @Failure      400      {object}  helpers.SwaggerValidationErrorResponse
// @Failure      401      {object}  helpers.SwaggerMessageResponse
// @Failure      404      {object}  helpers.SwaggerMessageResponse
// @Failure      409      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /users/{id} [put]
func (userCtrl *UserController) UpdateUser(c *gin.Context) {

	id, err := userCtrl.validatorPermissionId(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	var dtorequest dto.UserUpdateRequestDTO

	if err := c.ShouldBindJSON(&dtorequest); err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	if listFieldErrors := dtorequest.Validator(); listFieldErrors != nil {
		helpers.HelperErrResponse(listFieldErrors, c, apperrors.ErrInvalidInput)
		return
	}

	if err := userCtrl.updateUser.UpdateProfile(dtorequest.Name, dtorequest.Email, id); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, http.StatusNoContent, "user  successfully update ", c)

}

// ResetPassword inicia o fluxo de redefinicao de senha enviando codigo por email.
// @Summary      Solicitar redefinicao de senha
// @Description  Envia instrucoes e codigo de redefinicao de senha para o email informado, caso esteja cadastrado.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        request  body      dto.UserResetPasswordDTO  true  "Email do usuario"
// @Success      200      {object}  helpers.SwaggerMessageResponse
// @Failure      400      {object}  helpers.SwaggerMessageResponse
// @Failure      429      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /users/reset-password [post]
func (userCtrl *UserController) ResetPassword(c *gin.Context) {

	var requestDTO dto.UserResetPasswordDTO

	if err := c.ShouldBindJSON(&requestDTO); err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	if err := userCtrl.resetPassword.UseCase(requestDTO.Email); err != nil && !errors.Is(err, apperrors.ErrNotFound) {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, http.StatusOK, "If your email is registered, reset instructions have been sent.", c)

}

// ConfirmeResetPassword confirma a redefinicao de senha com o codigo enviado por email.
// @Summary      Confirmar redefinicao de senha
// @Description  Valida o codigo de recuperacao, atualiza a senha do usuario e revoga sessoes anteriores.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        request  body      dto.UserConfirmResetPasswordDTO  true  "Email, codigo e nova senha"
// @Success      200      {object}  helpers.SwaggerMessageResponse
// @Failure      400      {object}  helpers.SwaggerValidationErrorResponse
// @Failure      401      {object}  helpers.SwaggerMessageResponse
// @Failure      429      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /users/confirm-reset-password [post]
func (userCtrl *UserController) ConfirmeResetPassword(c *gin.Context) {

	var requestDTO dto.UserConfirmResetPasswordDTO

	if err := c.ShouldBindJSON(&requestDTO); err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	if listFieldErrors := requestDTO.Validator(); listFieldErrors != nil {
		helpers.HelperErrResponse(listFieldErrors, c, apperrors.ErrInvalidInput)
		return
	}

	if err := userCtrl.verifyUserCodePassword.UseCase(requestDTO.Email, requestDTO.Code, requestDTO.NewPassword); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) || errors.Is(err, apperrors.ErrUnprocessableEntity) {
			helpers.HelperErrResponse(nil, c, apperrors.ErrUnauthorized)
			return
		}
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	if updatedUser, err := userCtrl.readUserByEmail.UseCase(requestDTO.Email); err == nil && updatedUser != nil {
		auth.RevokeUserTokensBefore(updatedUser.ID.String())
	}

	helpers.HelperResponse(nil, http.StatusOK, "successfully request", c)

}

func (userCtrl *UserController) validatorPermissionId(c *gin.Context) (string, error) {

	valClaims, exists := c.Get("claims")

	if !exists {
		return "", apperrors.ErrUnauthorized
	}

	claims, ok := valClaims.(*auth.Claims)

	if !ok {
		return "", errors.New("Error converting declarations")
	}

	userClaimsId := claims.Id.String()

	id := c.Param("id")

	if userClaimsId != id && claims.Role != "admin" {
		return "", apperrors.ErrUnauthorized
	}

	return id, nil
}
