package routes

import (
	controllerAddress "e-commerce/adapters/controller/address"
	controllerLogin "e-commerce/adapters/controller/auth"
	controllerCart "e-commerce/adapters/controller/cart"
	controllerChat "e-commerce/adapters/controller/chat"
	controllerOrder "e-commerce/adapters/controller/order"
	controllerPayment "e-commerce/adapters/controller/payment"
	controllerProduct "e-commerce/adapters/controller/product"
	controllerUser "e-commerce/adapters/controller/user"
	redisAdapter "e-commerce/adapters/redis"
	"e-commerce/adapters/server/middleware"
	"e-commerce/config"
	_ "e-commerce/docs"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/time/rate"
)

func NewRoutes(
	userController *controllerUser.UserController,
	authController *controllerLogin.AuthController,
	productController *controllerProduct.ProductController,
	cartController *controllerCart.CartController,
	orderController *controllerOrder.OrderController,
	paymentController *controllerPayment.PaymentController,
	chatController *controllerChat.ChatController,
	addressController *controllerAddress.AddressController,
) *gin.Engine {

	r := gin.Default()

	if err := r.SetTrustedProxies(config.TrustedProxies); err != nil {
		panic(err)
	}

	globalLimiter := middleware.NewRateLimiter(redisAdapter.Client, "rl:ip:global", rate.Every(20*time.Millisecond), 60)
	webhookLimiter := middleware.NewRateLimiter(redisAdapter.Client, "rl:ip:webhook", rate.Every(200*time.Millisecond), 10)
	addressLookupLimiter := middleware.NewRateLimiter(redisAdapter.Client, "rl:ip:address_lookup", rate.Every(250*time.Millisecond), 20)

	createEmailLimiter := middleware.NewEmailRateLimiter(redisAdapter.Client, "rl:email:create", rate.Every(1*time.Minute), 3)

	loginEmailLimiter := middleware.NewEmailRateLimiter(redisAdapter.Client, "rl:email:login", rate.Every(5*time.Minute), 4)
	resendCodeLimiter := middleware.NewEmailRateLimiter(redisAdapter.Client, "rl:email:resend_code", rate.Every(1*time.Minute), 3)
	verifyCodeEmailLimiter := middleware.NewEmailRateLimiter(redisAdapter.Client, "rl:email:verify_code", rate.Every(20*time.Minute), 3)
	verifyCodePasswordEmailLimiter := middleware.NewEmailRateLimiter(redisAdapter.Client, "rl:email:verify_password_code", rate.Every(40*time.Minute), 3)

	r.Use(middleware.SecurityHeadersMiddleware())
	r.Use(middleware.MaxBodySizeMiddlewareWithRouteLimits(1<<20, map[string]int64{
		"POST /api/v1/products/image": 6 << 20, // 5 MB de imagem + overhead multipart
	}))
	r.Use(middleware.CorsMiddleware())
	r.Use(middleware.GzipMiddleware())
	r.Use(middleware.CSRFOriginMiddleware())
	r.Use(middleware.RateLimiterMiddleware(globalLimiter))

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")
	{

		users := v1.Group("/users")
		{
			users.POST("", middleware.EmailRateLimiterMiddleware(createEmailLimiter), userController.CreateUser)
			users.POST("/verify", middleware.EmailRateLimiterMiddleware(verifyCodeEmailLimiter), userController.VerifyEmailMe)
			users.POST("/new-verify-code", middleware.EmailRateLimiterMiddleware(resendCodeLimiter), userController.NewVerifyCode)
			users.POST("/reset-password", middleware.EmailRateLimiterMiddleware(resendCodeLimiter), userController.ResetPassword)
			users.POST("/confirm-reset-password", middleware.EmailRateLimiterMiddleware(verifyCodePasswordEmailLimiter), userController.ConfirmeResetPassword)
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
			products.GET("/search", productController.SearchProducts)
			products.GET("/:id", productController.ReadProductById)
			products.POST("", middleware.AuthMiddleware("admin"), productController.CreateProduct)
			products.POST("/image", middleware.AuthMiddleware("admin"), productController.UploadProductImage)
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

		orders := v1.Group("/orders")
		{
			orders.POST("", middleware.AuthMiddleware("user"), orderController.CreateOrderMe)
			orders.GET("/me", middleware.AuthMiddleware("user"), orderController.ReadAllOrdersMe)
			orders.GET("/:id", middleware.AuthMiddleware("user"), orderController.ReadOrderById)
			orders.PUT("/:id", middleware.AuthMiddleware("user"), orderController.UpdateOrder)
			orders.PUT("/:id/cancel", middleware.AuthMiddleware("user"), orderController.CancelOrder)
			orders.POST("/:id/refund", middleware.AuthMiddleware("user"), orderController.RequestRefund)
			orders.GET("/:id/history", middleware.AuthMiddleware("user"), orderController.ReadOrderHistory)
			orders.GET("/:id/items", middleware.AuthMiddleware("user"), orderController.ReadOrderItems)
			orders.POST("/:id/items", middleware.AuthMiddleware("user"), orderController.AddOrderItem)
			orders.PUT("/:id/items/:itemId", middleware.AuthMiddleware("user"), orderController.UpdateOrderItem)
			orders.DELETE("/:id/items/:itemId", middleware.AuthMiddleware("user"), orderController.RemoveOrderItem)

			// Pagamentos vinculados ao pedido
			orders.POST("/:id/payments", middleware.AuthMiddleware("user"), paymentController.ProcessPayment)
			orders.GET("/:id/payment", middleware.AuthMiddleware("user"), paymentController.ReadPaymentByOrder)
			if config.AllowPaymentSimulation && config.AccessTokenPayment == "" {
				orders.POST("/:id/payment/confirm", middleware.AuthMiddleware("user"), paymentController.ConfirmOrderPayment)
			}

			// Rotas administrativas
			orders.GET("", middleware.AuthMiddleware("admin"), orderController.ReadAllOrders)
			orders.PUT("/:id/status", middleware.AuthMiddleware("admin"), orderController.UpdateOrderStatus)
			orders.PUT("/:id/refund/approve", middleware.AuthMiddleware("admin"), orderController.ApproveRefund)
			orders.PUT("/:id/refund/reject", middleware.AuthMiddleware("admin"), orderController.RejectRefund)
			orders.DELETE("/:id", middleware.AuthMiddleware("admin"), orderController.DeleteOrder)
		}

		addresses := v1.Group("/addresses", middleware.AuthMiddleware("user"))
		{
			addresses.POST("", addressController.CreateAddress)
			addresses.GET("/me", addressController.ReadMyAddresses)
			addresses.GET("/lookup/:cep", middleware.RateLimiterMiddleware(addressLookupLimiter), addressController.LookupByCep)
		}

		chat := v1.Group("/chat", middleware.AuthMiddleware("user"))
		{
			chat.POST("", chatController.SendMessage)
		}

		payments := v1.Group("/payments")
		{
			payments.POST("/webhook", middleware.RateLimiterMiddleware(webhookLimiter), paymentController.WebhookPayment)
		}
	}

	return r
}
