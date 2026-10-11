package router

import (
	controller "e-commerce/adapters/controller/user"

	"github.com/gin-gonic/gin"
)

func NewGinRouter(userController controller.UserController) *gin.Engine {

	r := gin.Default()

	v1 := r.Group("/api/v1")
	{
		users := v1.Group("/users")
		{
			users.POST("", userController.CreateUser)
			users.GET("/:id", userController.ReadUser)
			users.DELETE("/:id", userController.DeleteUser)
		}
	}

	return r

}
