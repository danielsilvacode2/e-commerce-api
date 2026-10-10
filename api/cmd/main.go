package main

import (
	"e-commerce/adapters/database/mysql"
	"e-commerce/adapters/router"
	"e-commerce/config"
	
)

func init() {
	config.LoadConfig()
	mysql.NewMySQLConnection()
}

func main() {

 	r := router.NewGinRouter()
	r.Run(config.ApiPort)
}
