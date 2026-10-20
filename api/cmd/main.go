package main

import (
	"e-commerce/adapters/database"
	"e-commerce/adapters/server/router"
	"e-commerce/config"
)

func init() {
	config.LoadConfig()
	database.NewMySQLConnection()
}

func main() {

	r := router.NewGinRouter()
	r.Run(config.ApiPort)

}
