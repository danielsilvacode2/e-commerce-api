package router

import (
	controllerLogin "e-commerce/adapters/controller/login"
	controller "e-commerce/adapters/controller/user"
	"e-commerce/adapters/router/middleware"

	"github.com/gin-gonic/gin"
)

func NewGinRouter(
	userController *controller.UserController,
	loginController *controllerLogin.LoginController,

) *gin.Engine {

	r := gin.Default()

	v1 := r.Group("/api/v1")
	{
		users := v1.Group("/users")
		{

			users.POST("", userController.CreateUser)

			usersProtecd := users.Group("")
			{
				usersProtecd.GET("/:id", middleware.AuthMiddleware("user"), userController.ReadUser)
				usersProtecd.DELETE("/:id", middleware.AuthMiddleware("user"), userController.DeleteUser)
				usersProtecd.GET("", middleware.AuthMiddleware("user"), userController.ReadAllUsers)
			}
		}

		login := v1.Group("/login")
		{
			login.POST("",loginController.Login)
		}
	}

	return r

}
