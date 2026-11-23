package rabbitmq

import (
	"context"
	"e-commerce/core/ports"
	"encoding/json"
	"log"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type OrderNotificationRabbitMQ struct {
	channel      *amqp.Channel
	emailService ports.EmailService
	mu           *sync.Mutex
}

func NewOrderNotificationRabbitMQ(channel *amqp.Channel, emailService ports.EmailService) ports.EventOrderQueue {
	return &OrderNotificationRabbitMQ{
		channel:      channel,
		emailService: emailService,
		mu:           &sync.Mutex{},
	}
}

func (rb *OrderNotificationRabbitMQ) DeclareQueue(eventName string) error {
	if rb.channel == nil {
		return nil
	}

	rb.mu.Lock()
	defer rb.mu.Unlock()

	_, err := rb.channel.QueueDeclare(
		eventName,
		true,
		false,
		false,
		false,
		nil,
	)
	return err
}

// PublishOrderNotification publica o evento na fila RabbitMQ.
// Em caso de indisponibilidade do broker, aciona fallback assincrono via EmailService para nao perder notificacoes.
func (rb *OrderNotificationRabbitMQ) PublishOrderNotification(event ports.OrderNotificationEvent) error {
	if event.UserEmail == "" {
		return nil
	}

	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	if rb.channel != nil {
		rb.mu.Lock()
		pubErr := rb.channel.PublishWithContext(
			context.Background(),
			"",
			ports.QueueOrderNotifications,
			false,
			false,
			amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				Body:         body,
			},
		)
		rb.mu.Unlock()

		if pubErr == nil {
			return nil
		}

		log.Printf("[WARN] [OrderNotificationRabbitMQ] Falha ao publicar evento %s para order=%s, usando fallback de email: %v", event.EventType, event.OrderId, pubErr)
	}

	// Fallback resiliente quando RabbitMQ esta indisponivel
	if rb.emailService != nil {
		go func(ev ports.OrderNotificationEvent) {
			if err := rb.emailService.SendOrderNotification(ev); err != nil {
				log.Printf("[ERROR] [OrderNotificationFallback] Falha ao enviar email (%s) para %s: %v", ev.EventType, ev.UserEmail, err)
			}
		}(event)
	}

	return nil
}

type OrderNotificationConsumer struct {
	channel      *amqp.Channel
	emailService ports.EmailService
	mu           *sync.Mutex
}

func NewOrderNotificationConsumer(channel *amqp.Channel, emailService ports.EmailService) ports.EventOrderQueueConsumer {
	return &OrderNotificationConsumer{
		channel:      channel,
		emailService: emailService,
		mu:           &sync.Mutex{},
	}
}

func (c *OrderNotificationConsumer) DeclareQueue(eventName string) error {
	if c.channel == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	_, err := c.channel.QueueDeclare(
		eventName,
		true,
		false,
		false,
		false,
		nil,
	)
	return err
}

func (c *OrderNotificationConsumer) ConsumeOrderNotifications() error {
	if c.channel == nil {
		return nil
	}

	if err := c.channel.Qos(10, 0, false); err != nil {
		return err
	}

	msgs, err := c.channel.Consume(ports.QueueOrderNotifications, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			var event ports.OrderNotificationEvent
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				log.Printf("[ERROR] [OrderNotificationConsumer] Payload invalido: %v", err)
				_ = msg.Nack(false, false)
				continue
			}

			if err := c.emailService.SendOrderNotification(event); err != nil {
				log.Printf("[ERROR] [OrderNotificationConsumer] Falha ao enviar email (%s) para %s (order=%s): %v", event.EventType, event.UserEmail, event.OrderId, err)
				// Ack para evitar loop infinito de SMTP com credenciais invalidas em dev, mantendo log completo
				_ = msg.Ack(false)
				continue
			}

			log.Printf("[INFO] [OrderNotificationConsumer] Email (%s) enviado com sucesso para %s (order=%s)", event.EventType, event.UserEmail, event.OrderId)
			if err := msg.Ack(false); err != nil {
				log.Printf("[ERROR] [OrderNotificationConsumer] Falha ao dar ack: %v", err)
			}
		}
	}()

	return nil
}
