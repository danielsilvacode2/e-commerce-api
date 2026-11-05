package rabbitmq

import (
	"e-commerce/adapters/rabbitmq/dto"
	"e-commerce/adapters/rabbitmq/mapper"
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/user"
	"e-commerce/core/ports"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type UserRabbitMQConsumerDeclare struct {
	channel      *amqp.Channel
	queue        string
	repository   entities.UserRepository
	serviceEmail ports.EmailService
	mu           *sync.Mutex
}

func NewUserRabbitMQConsumer(
	channel *amqp.Channel,
	queue string,
	repository entities.UserRepository,
	serviceEmail ports.EmailService,
) ports.EventeUserQueueConsumerDeclare {
	return &UserRabbitMQConsumerDeclare{
		channel:      channel,
		queue:        queue,
		repository:   repository,
		serviceEmail: serviceEmail,
		mu:           &sync.Mutex{},
	}
}

func (consumer *UserRabbitMQConsumerDeclare) DeclareQueue(eventName string) error {

	consumer.mu.Lock()
	defer consumer.mu.Unlock()

	_, err := consumer.channel.QueueDeclare(
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

func (consumer *UserRabbitMQConsumerDeclare) Consumer() error {

	if err := consumer.channel.Qos(10, 0, false); err != nil {
		return err
	}

	msg, err := consumer.channel.Consume(consumer.queue, "", false, false, false, false, nil)

	if err != nil {
		return err
	}

	go func() {

		for q := range msg {

			var dto dto.RabbitMQUserDto

			if err := json.Unmarshal(q.Body, &dto); err != nil {
				q.Nack(false, false)
				continue
			}

			user, err := mapper.DtoToRabbitUser(dto)

			if err != nil {
				q.Nack(false, false)
				continue
			}

			if err := consumer.repository.CreateUser(*user); err != nil {

				switch {
				case errors.Is(err, apperrors.ErrDatabaseUnavailable):
					// único caso de retry: banco fora do ar
					log.Printf("banco indisponível, reenfileirando: %s", user.Email)
					time.Sleep(2 * time.Second)
					q.Nack(false, true)

				case errors.Is(err, apperrors.ErrConflict):
					// já existe: trata como sucesso (idempotência)
					log.Printf("usuário já existe, ignorando: %s", user.Email)
					q.Ack(false)

					go func() {
						if err := consumer.serviceEmail.SendEmail(user.Email, "Alguém tentou criar uma conta com seu email. Se foi você, entre aqui ou recupere sua senha. Se não foi, ignore", "Tentativa de registro"); err != nil {
							log.Printf("ERRO AO ENVIAR EMAIL: %s - ERRO: %s", user.Email, err.Error())
						} else {
							log.Printf("EMAIL ENVIADO COM SUCESSO PARA: %s", user.Email)
						}
					}()

				default:
					// erro inesperado: descarta, mas deixa registrado
					log.Printf("erro inesperado ao criar usuário %s: %v", user.Email, err)
					q.Nack(false, false)
				}

				continue
			}

			if err := q.Ack(false); err != nil {
				log.Printf("erro ao dar ack: %v", err)
			}

			go func() {
				if err := consumer.serviceEmail.SendEmail(user.Email, "Olá! Seu codigo de verificacao: "+user.VerificationCode, "Codigo de verificacao NERO"); err != nil {
					log.Printf("ERRO AO ENVIAR EMAIL: %s - ERRO: %s", user.Email, err.Error())
				} else {
					log.Printf("EMAIL ENVIADO COM SUCESSO PARA: %s", user.Email)
				}
			}()

		}

	}()

	return nil
}
