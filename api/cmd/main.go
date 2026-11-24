package main

import (
	"e-commerce/adapters/database"
	"e-commerce/adapters/rabbitmq"
	"e-commerce/adapters/redis"
	"e-commerce/builder"
	"e-commerce/core/ports"
	"log"
	"net/http"
	"time"

	"e-commerce/adapters/server/router"
	"e-commerce/config"
)

func init() {
	config.LoadConfig()
	database.NewMySQLConnection()
	rabbitmq.NewRabbitMQConnection()
	redis.NewRedisConnection()
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

	paymentSyncConsumer := builder.BuilderPaymentSyncConsumerRabbitMQ()
	if err := paymentSyncConsumer.DeclareQueue(ports.QueuePaymentSync); err != nil {
		log.Fatalf("[ERROR] falha ao declarar fila de sincronizacao de pagamentos: %v", err)
	}
	if err := paymentSyncConsumer.ConsumePaymentSync(); err != nil {
		log.Fatalf("[ERROR] falha ao iniciar consumer de sincronizacao de pagamentos: %v", err)
	}

	r := router.NewGinRouter()
	srv := &http.Server{
		Addr:              config.ApiPort,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("[ERROR] falha ao iniciar servidor HTTP: %v", err)
	}

}
