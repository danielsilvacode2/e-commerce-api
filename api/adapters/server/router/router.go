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
	product *controllerProduct.ProductController,
) *gin.Engine {

	r := gin.Default()

	IpRateLimiter := middleware.NewRateLimiter(rate.Every(time.Second*1), 5)

	v1 := r.Group("/api/v1")
	{
		users := v1.Group("/users")
		{

			users.POST("", middleware.RateLimiterMiddleware(IpRateLimiter), userController.CreateUser)

			usersProtecd := users.Group("")
			{
				usersProtecd.GET("/:id", middleware.RateLimiterMiddleware(IpRateLimiter), middleware.AuthMiddleware("admin"), userController.ReadUser)
				usersProtecd.DELETE("/:id", middleware.RateLimiterMiddleware(IpRateLimiter), middleware.AuthMiddleware("admin"), userController.DeleteUser)
				usersProtecd.GET("", middleware.RateLimiterMiddleware(IpRateLimiter), middleware.AuthMiddleware("admin"), userController.ReadAllUsers)

				usersProtecdMe := usersProtecd.Group("/me")
				{
					usersProtecdMe.GET("", middleware.RateLimiterMiddleware(IpRateLimiter), middleware.AuthMiddleware("user"), userController.ReadMe)
				}

			}
		}

		login := v1.Group("/login")
		{
			login.POST("", middleware.RateLimiterMiddleware(IpRateLimiter), loginController.Login)
		}

		products := v1.Group("/products")
		{
			products.POST("", middleware.RateLimiterMiddleware(IpRateLimiter), product.CreateProduct)
			products.GET("/:id", middleware.RateLimiterMiddleware(IpRateLimiter), product.ReadProductById)
			products.GET("", middleware.RateLimiterMiddleware(IpRateLimiter), product.ReadAllProducts)
			products.DELETE("/:id", middleware.RateLimiterMiddleware(IpRateLimiter), product.DeleteProductById)
		}

	}

	return r

}
