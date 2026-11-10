package routes

import (
	controllerLogin "e-commerce/adapters/controller/auth"
	controllerCart "e-commerce/adapters/controller/cart"
	controllerProduct "e-commerce/adapters/controller/product"
	controllerUser "e-commerce/adapters/controller/user"
	"e-commerce/adapters/server/middleware"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func NewRoutes(
	userController *controllerUser.UserController,
	authController *controllerLogin.AuthController,
	productController *controllerProduct.ProductController,
	cartController *controllerCart.CartController,
) *gin.Engine {

	r := gin.Default()

	
	if err := r.SetTrustedProxies(nil); err != nil {
		panic(err)
	}

	
	globalLimiter := middleware.NewRateLimiter(rate.Every(100*time.Millisecond), 20)
	
	createEmailLimiter := middleware.NewEmailRateLimiter(rate.Every(1*time.Minute), 3)

	loginEmailLimiter := middleware.NewEmailRateLimiter(rate.Every(5*time.Minute), 4)
	resendCodeLimiter := middleware.NewEmailRateLimiter(rate.Every(1*time.Minute), 3)
	verifyCodeEmailLimiter := middleware.NewEmailRateLimiter(rate.Every(20*time.Minute), 3)

	r.Use(middleware.CorsMiddleware())
	r.Use(middleware.RateLimiterMiddleware(globalLimiter))

	v1 := r.Group("/api/v1")
	{
		
		users := v1.Group("/users")
		{
			users.POST("", middleware.EmailRateLimiterMiddleware(createEmailLimiter), userController.CreateUser)
			users.POST("/verify", middleware.EmailRateLimiterMiddleware(resendCodeLimiter), userController.VerifyEmailMe)
			users.POST("/new-verify-code", middleware.EmailRateLimiterMiddleware(verifyCodeEmailLimiter), userController.NewVerifyCode)
		}

		
		usersProtected := users.Group("")
		{
			usersProtected.GET("", middleware.AuthMiddleware("admin"), userController.ReadAllUsers)
			usersProtected.GET("/me", middleware.AuthMiddleware("user"), userController.ReadMe)
			usersProtected.GET("/:id", middleware.AuthMiddleware("admin"), userController.ReadUser)
			usersProtected.PUT("/:id", middleware.AuthMiddleware("user"), userController.UpdateUser)
			usersProtected.DELETE("/:id", middleware.AuthMiddleware("admin"), userController.DeleteUser)
		}

	
		auth := v1.Group("")
		{
			auth.POST("/login", middleware.EmailRateLimiterLoginMiddleware(loginEmailLimiter), authController.Login)
			auth.POST("/logout", authController.Logout)
		}

		
		products := v1.Group("/products")
		{
			products.GET("", productController.ReadAllProducts)
			products.GET("/:id", middleware.AuthMiddleware("user"), productController.ReadProductById)
			products.POST("", middleware.AuthMiddleware("admin"), productController.CreateProduct)
			products.PUT("/:id", middleware.AuthMiddleware("admin"), productController.UpdateProduct)
			products.DELETE("/:id", middleware.AuthMiddleware("admin"), productController.DeleteProductById)
		}

		
		cart := v1.Group("/cart", middleware.AuthMiddleware("user"))
		{
			cart.GET("", cartController.ReadAllProductCartMe)
			cart.POST("", cartController.AddProductCartMe)
			cart.PUT("", cartController.UpdateIsAllSelectProductMe)
			cart.PUT("/:id", cartController.UpdateIsSelectProductMe)
			cart.PUT("/decrease/:id", cartController.DecreaseProductCartMe)
			cart.DELETE("/:id", cartController.RemoveProductCartMe)
		}
	}

	return r
}
