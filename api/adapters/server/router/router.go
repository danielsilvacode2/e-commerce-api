package router

import (
	controllerLogin "e-commerce/adapters/controller/login"
	controllerProduct "e-commerce/adapters/controller/product"
	controllerUser "e-commerce/adapters/controller/user"
	"e-commerce/adapters/server/middleware"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func NewGinRouter(
	userController *controllerUser.UserController,
	loginController *controllerLogin.LoginController,
	productController *controllerProduct.ProductController,
) *gin.Engine {

	r := gin.Default()

	ipRateLimiter := middleware.NewRateLimiter(rate.Every(time.Second*1), 5)

	v1 := r.Group("/api/v1")
	{
		users := v1.Group("/users")
		{

			users.POST("", middleware.RateLimiterMiddleware(ipRateLimiter), userController.CreateUser)

			usersProtecd := users.Group("")
			{
				usersProtecd.GET("/:id", middleware.RateLimiterMiddleware(ipRateLimiter), middleware.AuthMiddleware("admin"), userController.ReadUser)
				usersProtecd.DELETE("/:id", middleware.RateLimiterMiddleware(ipRateLimiter), middleware.AuthMiddleware("admin"), userController.DeleteUser)
				usersProtecd.GET("", middleware.RateLimiterMiddleware(ipRateLimiter), middleware.AuthMiddleware("admin"), userController.ReadAllUsers)
				usersProtecd.PUT("/:id", middleware.RateLimiterMiddleware(ipRateLimiter), middleware.AuthMiddleware("user"), userController.UpdateUser)

				usersProtecdMe := usersProtecd.Group("/me")
				{
					usersProtecdMe.GET("", middleware.RateLimiterMiddleware(ipRateLimiter), middleware.AuthMiddleware("user"), userController.ReadMe)
				}

			}
		}

		login := v1.Group("/login")
		{
			login.POST("", middleware.RateLimiterMiddleware(ipRateLimiter), loginController.Login)
		}

		products := v1.Group("/products")
		{
			products.POST("", middleware.RateLimiterMiddleware(ipRateLimiter), middleware.AuthMiddleware("admin"), productController.CreateProduct)
			products.GET("/:id", middleware.RateLimiterMiddleware(ipRateLimiter), middleware.AuthMiddleware("admin"), productController.ReadProductById)
			products.GET("", middleware.RateLimiterMiddleware(ipRateLimiter), middleware.AuthMiddleware("admin"), productController.ReadAllProducts)
			products.DELETE("/:id", middleware.RateLimiterMiddleware(ipRateLimiter), middleware.AuthMiddleware("admin"), productController.DeleteProductById)
			products.PUT("/:id", middleware.RateLimiterMiddleware(ipRateLimiter), middleware.AuthMiddleware("admin"), productController.UpdateProduct)
		}

	}

	return r

}
