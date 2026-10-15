package router

import (
	controllerLogin "e-commerce/adapters/controller/login"
	controllerProduct "e-commerce/adapters/controller/product"
	controllerUser "e-commerce/adapters/controller/user"
	"e-commerce/adapters/server/middleware"

	"github.com/gin-gonic/gin"
)

func NewGinRouter(
	userController *controllerUser.UserController,
	loginController *controllerLogin.LoginController,
	product *controllerProduct.ProductController,
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
				usersProtecd.GET("", middleware.AuthMiddleware("admin"), userController.ReadAllUsers)

				usersProtecdMe := usersProtecd.Group("/me")
				{
					usersProtecdMe.GET("", middleware.AuthMiddleware("user"), userController.ReadMe)
				}

			}
		}

		login := v1.Group("/login")
		{
			login.POST("", loginController.Login)
		}

		products := v1.Group("/products")
		{
			products.POST("", product.CreateProduct)
			products.GET("/:id", product.ReadProductById)
			products.GET("", product.ReadAllProducts)
			products.DELETE("/:id", product.DeleteProductById)
		}

	}

	return r

}
