package main

import (
	controller "e-commerce/adapters/controller/user"
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
	userController := controller.NewUserController(userRepository)

	r := router.NewGinRouter(*userController)
	r.Run(config.ApiPort)
}
