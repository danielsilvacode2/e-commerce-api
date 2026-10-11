package controller

import (
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/controller/user/dto"
	"e-commerce/core/entities/user"
	usecase "e-commerce/core/use_cases/user"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	repository entities.UserInterface
}

func NewUserController(repository entities.UserInterface) *UserController {
	return &UserController{
		repository: repository,
	}
}

func (userCtrl *UserController) CreateUser(c *gin.Context) {

	var userDto dto.UserRequestDTO

	if err := c.ShouldBindJSON(&userDto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"mensagem": "erro ao ler corpo da requisicao",
		})
		return
	}

	user := entities.User{
		Name:     userDto.Name,
		Email:    userDto.Email,
		Password: userDto.Password,
	}

	if err := usecase.CreateUser(user, userCtrl.repository); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"mensagem": "erro ao criar usuario",
			"erro":     err,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"mensagem":         "usuario criado com sucesso",
		"dados do usuario": userDto,
	})
}

func (userCtrl *UserController) ReadUser(c *gin.Context) {

	userId := c.Param("id")

	user, err := usecase.ReadUser(userId, userCtrl.repository)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	userDto := dto.UserResponseDTO{
		Id:         user.ID,
		Name:       user.Name,
		Email:      user.Email,
		Role:       user.Role,
		Update_at:  user.UpdatedAt,
		Created_at: user.CreatedAt,
	}

	
	helpers.HelperResponse(userDto, http.StatusOK, "user found", c)

}
