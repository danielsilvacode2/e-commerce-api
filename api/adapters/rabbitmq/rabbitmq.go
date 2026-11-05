package rabbitmq

/*DeclareQueue(eventName string) error
PublishEvent(eventName string, user entities.User) error
*/

import (
	"context"
	"e-commerce/adapters/rabbitmq/mapper"
	entities "e-commerce/core/entities/user"
	"e-commerce/core/ports"
	"encoding/json"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type UserRabbitMQ struct {
	channel *amqp.Channel
	mu      *sync.Mutex
}

func NewUserRabbitMQ(channel *amqp.Channel) ports.EventUserQueue {
	return &UserRabbitMQ{
		channel: channel,
		mu:      &sync.Mutex{},
	}
}

func (rb *UserRabbitMQ) DeclareQueue(eventName string) error {

	rb.mu.Lock()
	defer rb.mu.Unlock()

	_, err := rb.channel.QueueDeclare(
		eventName,
		true,  // durable (sobrevive a reinício do broker)
		false, // autoDelete (apaga quando não há consumidores)
		false, // exclusive (só esta conexão acessa)
		false, // noWait
		nil)

	if err != nil {
		return err
	}

	return nil
}

func (rb *UserRabbitMQ) PublishEvent(eventName string, user entities.User) error {

	dto := mapper.UserToRabbitDTO(user)

	body, err := json.Marshal(dto)

	if err != nil {
		return err
	}

	rb.mu.Lock()
	defer rb.mu.Unlock()

	err = rb.channel.PublishWithContext(context.Background(), "", eventName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})

	if err != nil {
		return err
	}

	return nil
}
