package main

import (
	controllerLogin "e-commerce/adapters/controller/login"
	controllerUser "e-commerce/adapters/controller/user"
	"e-commerce/adapters/database"
	repository "e-commerce/adapters/repository/user"
	"e-commerce/adapters/security"
	"e-commerce/adapters/server/router"
	"e-commerce/config"
)

func init() {
	config.LoadConfig()
	database.NewMySQLConnection()
}

func main() {
	
	userRepository := repository.NewUserRepository(database.DB_MySQL)

	passwordService := security.NewPasswordBcript()

	userController := controllerUser.NewUserController(userRepository, passwordService)
	loginController := controllerLogin.NewLoginController(userRepository, passwordService)

	r := router.NewGinRouter(userController, loginController)
	r.Run(config.ApiPort)

}
