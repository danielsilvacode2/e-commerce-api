package main

import (
	"e-commerce/adapters/database"
	"e-commerce/adapters/rabbitmq"
	"e-commerce/builder"
	"log"

	"e-commerce/adapters/server/router"
	"e-commerce/config"
)

func init() {
	config.LoadConfig()
	database.NewMySQLConnection()
	rabbitmq.NewRabbitMQConnection()
}

func main() {

	rabbitConsumerDeclare := builder.BuilderConsumerRabbitMQ()

	rabbitConsumerDeclare.DeclareQueue("createusers")

	if err := rabbitConsumerDeclare.Consumer(); err != nil {
		log.Fatalf("erro ao iniciar consumer: %v", err)
	}

	r := router.NewGinRouter()
	r.Run(config.ApiPort)

}
