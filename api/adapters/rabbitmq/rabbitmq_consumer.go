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
	repository   entities.UserRepository
	serviceEmail ports.EmailService
	mu           *sync.Mutex
	hashService  ports.HashService
}

func NewUserRabbitMQConsumer(
	channel *amqp.Channel,
	repository entities.UserRepository,
	serviceEmail ports.EmailService,
	hashService ports.HashService,
) ports.EventeUserQueueConsumerDeclare {
	return &UserRabbitMQConsumerDeclare{
		channel:      channel,
		repository:   repository,
		serviceEmail: serviceEmail,
		mu:           &sync.Mutex{},
		hashService:  hashService,
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

func (consumer *UserRabbitMQConsumerDeclare) ConsumerUser() error {

	if err := consumer.channel.Qos(10, 0, false); err != nil {
		return err
	}

	msg, err := consumer.channel.Consume(ports.QueueCreateUser, "", false, false, false, false, nil)

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

			codeHash, err := consumer.hashService.HashGenerate(user.VerificationCode)

			if err != nil {
				q.Nack(false, false)
				continue
			}

			code := user.VerificationCode
			user.VerificationCode = codeHash

			if err := consumer.repository.CreateUser(*user); err != nil {

				switch {
				case errors.Is(err, apperrors.ErrDatabaseUnavailable):

					log.Printf("[ERROR] banco indisponível, reenfileirando: %s", user.Email)
					time.Sleep(2 * time.Second)
					q.Nack(false, true)

				case errors.Is(err, apperrors.ErrConflict):

					log.Printf("[INFO] usuário já existe, ignorando: %s", user.Email)
					q.Ack(false)

					go func() {
						if err := consumer.serviceEmail.SendSuspiciousLogin(user.Email); err != nil {
							log.Printf("[ERROR] [SuspiciousLoginEmail] Falha ao enviar para userID=%s email=%s: %v", user.ID, user.Email, err)
						} else {
							log.Printf("[INFO] [SuspiciousLoginEmail] E-mail enviado com sucesso para userID=%s email=%s", user.ID, user.Email)
						}
					}()

				default:
					// erro inesperado: descarta, mas deixa registrado
					log.Printf("[ERROR] falha inesperada ao criar usuário %s: %v", user.Email, err)
					q.Nack(false, false)
				}

				continue
			}

			if err := q.Ack(false); err != nil {
				log.Printf("[ERROR] falha ao dar ack: %v", err)
			}

			go func() {

				if err := consumer.serviceEmail.SendVerificationCode(user.Email, code, "Verificar email"); err != nil {
					log.Printf("[ERROR] [VerificationCodeEmail] Falha ao enviar para userID=%s email=%s: %v", user.ID, user.Email, err)
				} else {
					log.Printf("[INFO] [VerificationCodeEmail] E-mail enviado com sucesso para userID=%s email=%s", user.ID, user.Email)
				}

			}()
		}

	}()

	return nil
}
