package controller

import (
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/controller/user/dto"
	"e-commerce/adapters/controller/user/mapper"
	"e-commerce/adapters/security/auth"
	"e-commerce/core/apperrors"
	"e-commerce/core/entities/user"
	"e-commerce/core/ports"
	usecase "e-commerce/core/use_cases/user"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	repository      entities.UserRepository
	passwordService ports.PasswordService
}

func NewUserController(repository entities.UserRepository, passwordService ports.PasswordService) *UserController {
	return &UserController{
		repository:      repository,
		passwordService: passwordService,
	}
}

// CreateUser registra um usuario na api
func (userCtrl *UserController) CreateUser(c *gin.Context) {
	var userRequestDto dto.UserRequestDTO

	if err := c.ShouldBindJSON(&userRequestDto); err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	if listErrValidator := userRequestDto.Validator(); listErrValidator != nil {
		helpers.HelperErrResponse(listErrValidator, c, apperrors.ErrInvalidInput)
		return
	}

	potentialUser := mapper.DtoRequestToUser(userRequestDto)

	user, err := usecase.CreateUser(potentialUser, userCtrl.repository, userCtrl.passwordService)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	userResponseDto := mapper.UserToResponseDTO(*user)

	helpers.HelperResponse(userResponseDto, http.StatusCreated, "registered user", c)
}

// ReadUser recebe um Id como parametro e retorna o usuario caso ele exista
func (userCtrl *UserController) ReadUser(c *gin.Context) {
	userId := c.Param("id")

	user, err := usecase.ReadUserById(userId, userCtrl.repository)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	userDto := mapper.UserToResponseDTO(*user)
	helpers.HelperResponse(userDto, http.StatusOK, "user found", c)
}

// DeleteUser recebe um Id como parametro e deleta o usuario
func (userCtrl *UserController) DeleteUser(c *gin.Context) {
	userId := c.Param("id")
	if err := usecase.DeleteUser(userId, userCtrl.repository); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, http.StatusNoContent, "user successfully deleted ", c)
}

func (userCtrl *UserController) ReadAllUsers(c *gin.Context) {

	listUsers, err := usecase.ReadAllUsers(userCtrl.repository)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
	}

	var listUserDTO []dto.UserResponseDTO

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

	if !ok{
		helpers.HelperErrResponse(nil, c, errors.New("Error converting declarations"))
		return
	} 
		 
	userId := claims.Id.String()

	user, err := usecase.ReadUserById( userId, userCtrl.repository)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	dto := mapper.UserToResponseDTO(*user)

	helpers.HelperResponse(dto, http.StatusOK, "user found", c)

}
