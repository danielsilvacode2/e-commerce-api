package main

import (
	controllerLogin "e-commerce/adapters/controller/login"
	controllerUser "e-commerce/adapters/controller/user"
	"e-commerce/adapters/database"
	repository "e-commerce/adapters/repository/user"
	"e-commerce/adapters/router"
	"e-commerce/config"
)

func init() {
	config.LoadConfig()
	database.NewMySQLConnection()
}

func main() {

	userRepository := repository.NewUserRepository(database.DB_MySQL)
	userController := controllerUser.NewUserController(userRepository)
	loginController := controllerLogin.NewLoginController(userRepository)

	r := router.NewGinRouter(userController,loginController)
	r.Run(config.ApiPort)
}
