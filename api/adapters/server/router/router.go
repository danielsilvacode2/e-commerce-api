package router

import (
	"e-commerce/adapters/server/router/routes"
	"e-commerce/builder"

	"github.com/gin-gonic/gin"
)

func NewGinRouter() *gin.Engine {

	userController := builder.BuilderUserController()
	loginController := builder.BuilderLoginController()
	productController := builder.BuilderProductController()
	cartController := builder.BuilderCartController()
	orderController := builder.BuilderOrderController()
	paymentController := builder.BuilderPaymentController()

	r := routes.NewRoutes(
		userController,
		loginController,
		productController,
		cartController,
		orderController,
		paymentController,
	)

	return r

}
