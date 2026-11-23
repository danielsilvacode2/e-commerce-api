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

	if err := rabbitConsumerDeclare.DeclareQueue(ports.QueueCreateUser); err != nil {
		log.Fatalf("[ERROR] falha ao declarar fila de criacao de usuarios: %v", err)
	}

	if err := rabbitConsumerDeclare.ConsumerUser(); err != nil {
		log.Fatalf("[ERROR] falhar ao iniciar consumer de criacao de usuarios: %v", err)
	}

	orderNotificationConsumer := builder.BuilderOrderNotificationConsumerRabbitMQ()
	if err := orderNotificationConsumer.DeclareQueue(ports.QueueOrderNotifications); err != nil {
		log.Fatalf("[ERROR] falha ao declarar fila de notificacoes de pedidos: %v", err)
	}
	if err := orderNotificationConsumer.ConsumeOrderNotifications(); err != nil {
		log.Fatalf("[ERROR] falha ao iniciar consumer de notificacoes de pedidos: %v", err)
	}

	r := router.NewGinRouter()
	r.Run(config.ApiPort)

}
