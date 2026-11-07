package main

import (
	"e-commerce/adapters/database"
	"e-commerce/adapters/rabbitmq"
	"e-commerce/builder"
	"e-commerce/core/ports"
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

	rabbitConsumerDeclare.DeclareQueue(ports.QueueCreateUser)

	if err := rabbitConsumerDeclare.ConsumerUser(); err != nil {
		log.Fatalf("[ERROR] falhar ao iniciar consumer de criacao de usuarios: %v", err)
	}

	r := router.NewGinRouter()
	r.Run(config.ApiPort)

}
