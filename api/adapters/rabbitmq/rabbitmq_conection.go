package rabbitmq

import (
	"e-commerce/config"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

var ChannelRabbitMQ *amqp.Channel

func NewRabbitMQConnection() {

	conn, err := amqp.Dial(config.RabbitMQUrl)

	if err != nil {
		log.Fatal("[ERROR] falhar ao abrir conexao com rabbitmq: ", err)
	}

	ch, err := conn.Channel()

	if err != nil {
		log.Fatal("[ERROR] falhar ao criar channel para o rabbitmq: ", err)
	}

	ChannelRabbitMQ = ch

}
