package main

import (
	"e-commerce/adapters/database"
	"e-commerce/adapters/server/router"
	"e-commerce/config"
)



func main() {

	config.LoadConfig()
	database.NewMySQLConnection()

	r := router.NewGinRouter()
	r.Run(config.ApiPort)

}



