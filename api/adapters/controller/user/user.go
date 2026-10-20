package controller

import (
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/controller/user/dto"
	"e-commerce/adapters/controller/user/mapper"
	"e-commerce/adapters/security/auth"
	"e-commerce/core/apperrors"

	use_cases "e-commerce/core/use_cases/user"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	createUser   *use_cases.CreateUserUseCase
	deleteUser   *use_cases.DeleteUserUseCase
	readAllUsers *use_cases.ReadAllUsersUseCase
	updateUser   *use_cases.UpdateUseUseCase
	readUserById *use_cases.ReadUserByIdUseCase
}

func NewUserController(
	createUser *use_cases.CreateUserUseCase,
	deleteUser *use_cases.DeleteUserUseCase,
	readAllUsers *use_cases.ReadAllUsersUseCase,
	updateUser *use_cases.UpdateUseUseCase,
	readUserById *use_cases.ReadUserByIdUseCase,
) *UserController {
	return &UserController{
		createUser:   createUser,
		deleteUser:   deleteUser,
		readAllUsers: readAllUsers,
		updateUser:   updateUser,
		readUserById: readUserById,
	}
}

// CreateUser registra um usuario na api
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

	user, err := userCtrl.createUser.UseCase(potentialUser)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	userResponseDto := mapper.UserToResponseDTO(*user)

	helpers.HelperResponse(userResponseDto, http.StatusCreated, "registered user", c)
}

// ReadUser recebe um Id como parametro e retorna o usuario caso ele exista
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

// DeleteUser recebe um Id como parametro e deleta o usuario
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

	helpers.HelperResponse(nil, http.StatusNoContent, "user successfully deleted ", c)
}

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

func (userCtrl *UserController) UpdateUser(c *gin.Context) {

	id, err := userCtrl.validatorPermissionId(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	var dtorequest dto.UserUpdateRequestDTO

	if err := c.ShouldBindJSON(&dtorequest); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	if listFieldErrors := dtorequest.Validator(); listFieldErrors != nil {
		helpers.HelperErrResponse(listFieldErrors, c, apperrors.ErrInvalidInput)
		return
	}

	user := mapper.DtoUpdateRequestToUser(dtorequest)

	if err := userCtrl.updateUser.UseCase(user, id); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, http.StatusNoContent, "user  successfully update ", c)

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
