package routes

import (
	controllerCart "e-commerce/adapters/controller/cart"
	controllerLogin "e-commerce/adapters/controller/login"
	controllerProduct "e-commerce/adapters/controller/product"
	controllerUser "e-commerce/adapters/controller/user"
	"e-commerce/adapters/server/middleware"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func NewRoutes(
	userController *controllerUser.UserController,
	loginController *controllerLogin.LoginController,
	productController *controllerProduct.ProductController,
	cartController *controllerCart.CartController,
) *gin.Engine {

	r := gin.Default()

	ipRateLimiter := middleware.NewRateLimiter(rate.Every(time.Millisecond*100), 20)

	r.Use(middleware.CorsMiddleware())
	r.Use(middleware.RateLimiterMiddleware(ipRateLimiter))

	v1 := r.Group("/api/v1")
	{
		users := v1.Group("/users")
		{

			users.POST("", userController.CreateUser)

			usersProtecd := users.Group("")
			{
				usersProtecd.GET("/:id", middleware.AuthMiddleware("admin"), userController.ReadUser)
				usersProtecd.DELETE("/:id", middleware.AuthMiddleware("admin"), userController.DeleteUser)
				usersProtecd.GET("", middleware.AuthMiddleware("admin"), userController.ReadAllUsers)
				usersProtecd.PUT("/:id", middleware.AuthMiddleware("user"), userController.UpdateUser)

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
			products.POST("", middleware.AuthMiddleware("admin"), productController.CreateProduct)
			products.GET("/:id", middleware.AuthMiddleware("user"), productController.ReadProductById)
			products.GET("", productController.ReadAllProducts)
			products.DELETE("/:id", middleware.AuthMiddleware("admin"), productController.DeleteProductById)
			products.PUT("/:id", middleware.AuthMiddleware("admin"), productController.UpdateProduct)
		}

		cart := v1.Group("/cart")
		{
			cart.POST("", middleware.AuthMiddleware("user"), cartController.AddProductCartMe)
			cart.GET("", middleware.AuthMiddleware("user"), cartController.ReadAllProductCartMe)
			cart.DELETE("/:id", middleware.AuthMiddleware("user"), cartController.RemoveProductCartMe)
			cart.PUT("/:id", middleware.AuthMiddleware("user"), cartController.UpdateIsSelectProductMe)
			cart.PUT("", middleware.AuthMiddleware("user"), cartController.UpdateIsAllSelectProductMe)
			cart.PUT("/decrease/:id", middleware.AuthMiddleware("user"), cartController.DecreaseProductCartMe)
		}

	}

	return r

}
