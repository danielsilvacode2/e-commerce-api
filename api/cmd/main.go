package main

import (
	controller "e-commerce/adapters/controller/user"
	"e-commerce/adapters/database/mysql"
	repository "e-commerce/adapters/database/repository/user"
	"e-commerce/adapters/router"
	"e-commerce/config"
)

func init() {
	config.LoadConfig()
	mysql.NewMySQLConnection()
}

func main() {

	userRepository := repository.NewUserRepository(mysql.DbConnection)
	userController := controller.NewUserController(userRepository)

 	r := router.NewGinRouter(*userController)
	r.Run(config.ApiPort)
}
