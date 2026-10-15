package main

import (
	controllerLogin "e-commerce/adapters/controller/login"
	controllerProduct "e-commerce/adapters/controller/product"
	controllerUser "e-commerce/adapters/controller/user"
	"e-commerce/adapters/database"
	repositoryProduct "e-commerce/adapters/repository/product"
	repositoryUser "e-commerce/adapters/repository/user"
	"e-commerce/adapters/security"
	"e-commerce/adapters/server/router"
	"e-commerce/config"
)

func init() {
	config.LoadConfig()
	database.NewMySQLConnection()
}

func main() {
	
	userRepository := repositoryUser.NewUserRepository(database.DB_MySQL)

	passwordService := security.NewPasswordBcript()

	userController := controllerUser.NewUserController(userRepository, passwordService)
	loginController := controllerLogin.NewLoginController(userRepository, passwordService)

	productRepository := repositoryProduct.NewProductRepository(database.DB_MySQL)
	productController := controllerProduct.NewProductController(productRepository)

	r := router.NewGinRouter(userController, loginController, productController)
	r.Run(config.ApiPort)

}
