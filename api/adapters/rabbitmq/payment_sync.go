package rabbitmq

import (
	"context"
	"e-commerce/core/apperrors"
	entitiesPayment "e-commerce/core/entities/payment"
	"e-commerce/core/ports"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type PaymentSyncRabbitMQ struct {
	channel *amqp.Channel
	mu      *sync.Mutex
}

func NewPaymentSyncRabbitMQ(channel *amqp.Channel) ports.EventPaymentSyncQueue {
	return &PaymentSyncRabbitMQ{
		channel: channel,
		mu:      &sync.Mutex{},
	}
}

func (rb *PaymentSyncRabbitMQ) DeclareQueue(eventName string) error {
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

// PublishPaymentSync enfileira o resultado retornado pelo gateway quando o banco de dados falha apos a cobranca.
func (rb *PaymentSyncRabbitMQ) PublishPaymentSync(event ports.PaymentSyncEvent) error {
	if rb.channel == nil {
		return errors.New("rabbitmq channel indisponivel para fallback de pagamento")
	}

	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	rb.mu.Lock()
	defer rb.mu.Unlock()

	return rb.channel.PublishWithContext(
		context.Background(),
		"",
		ports.QueuePaymentSync,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}

type PaymentSyncConsumer struct {
	channel         *amqp.Channel
	paymentRepo     entitiesPayment.PaymentRepository
	orderEventQueue ports.EventOrderQueue
	mu              *sync.Mutex
}

func NewPaymentSyncConsumer(
	channel *amqp.Channel,
	paymentRepo entitiesPayment.PaymentRepository,
	orderEventQueue ports.EventOrderQueue,
) ports.EventPaymentSyncConsumer {
	return &PaymentSyncConsumer{
		channel:         channel,
		paymentRepo:     paymentRepo,
		orderEventQueue: orderEventQueue,
		mu:              &sync.Mutex{},
	}
}

func (c *PaymentSyncConsumer) DeclareQueue(eventName string) error {
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

func (c *PaymentSyncConsumer) ConsumePaymentSync() error {
	if c.channel == nil {
		return nil
	}

	if err := c.channel.Qos(10, 0, false); err != nil {
		return err
	}

	msgs, err := c.channel.Consume(ports.QueuePaymentSync, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			var event ports.PaymentSyncEvent
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				log.Printf("[ERROR] [PaymentSyncConsumer] Payload invalido: %v", err)
				_ = msg.Nack(false, false)
				continue
			}

			if err := c.HandleEvent(event); err != nil {
				if errors.Is(err, apperrors.ErrDatabaseUnavailable) {
					log.Printf("[WARN] [PaymentSyncConsumer] Banco indisponivel, reenfileirando payment=%s order=%s action=%s", event.PaymentId, event.OrderId, event.Action)
					time.Sleep(2 * time.Second)
					_ = msg.Nack(false, true)
					continue
				}

				log.Printf("[ERROR] [PaymentSyncConsumer] Falha permanente ao sincronizar payment=%s order=%s action=%s: %v", event.PaymentId, event.OrderId, event.Action, err)
				_ = msg.Nack(false, false)
				continue
			}

			if err := msg.Ack(false); err != nil {
				log.Printf("[ERROR] [PaymentSyncConsumer] Falha ao dar ack para payment=%s: %v", event.PaymentId, err)
			}
		}
	}()

	return nil
}

// HandleEvent aplica o evento de fallback no banco de forma idempotente.
func (c *PaymentSyncConsumer) HandleEvent(event ports.PaymentSyncEvent) error {
	paymentId, err := uuid.Parse(event.PaymentId)
	if err != nil {
		return apperrors.ErrInvalidInput
	}
	orderId, err := uuid.Parse(event.OrderId)
	if err != nil {
		return apperrors.ErrInvalidInput
	}

	payment := entitiesPayment.Payment{
		Id:                paymentId,
		OrderId:           orderId,
		Provider:          event.Provider,
		Method:            event.Method,
		ExternalPaymentId: event.ExternalPaymentId,
		IdempotencyKey:    event.IdempotencyKey,
		Amount:            event.Amount,
		Installments:      event.Installments,
		Status:            event.Status,
		PixQrCode:         event.PixQrCode,
		PixCopyPaste:      event.PixCopyPaste,
		CardLast4:         event.CardLast4,
		CardBrand:         event.CardBrand,
		ExpiresAt:         event.ExpiresAt,
		PaidAt:            event.PaidAt,
	}

	// Garante que o registro base existe no banco (caso raro de perda ou inconsistencia)
	existing, readErr := c.paymentRepo.ReadPaymentById(payment.Id.String())
	if readErr != nil {
		if errors.Is(readErr, apperrors.ErrNotFound) {
			pendingPayment := payment
			pendingPayment.Status = entitiesPayment.PaymentStatusPending
			pendingPayment.PaidAt = nil
			if createErr := c.paymentRepo.CreatePayment(pendingPayment); createErr != nil && !errors.Is(createErr, apperrors.ErrConflict) {
				return createErr
			}
		} else {
			return readErr
		}
	} else if existing != nil && existing.Status != entitiesPayment.PaymentStatusPending {
		// Idempotencia: ja foi finalizado anteriormente (ex: webhook ou polling)
		return nil
	}

	switch event.Action {
	case ports.PaymentSyncActionUpdateGatewayData:
		if err := c.paymentRepo.UpdatePaymentGatewayData(payment); err != nil {
			return err
		}
		log.Printf("[INFO] [PaymentSyncConsumer] Dados do gateway sincronizados para payment=%s order=%s", event.PaymentId, event.OrderId)
		return nil

	case ports.PaymentSyncActionApprove:
		if payment.PaidAt == nil {
			now := time.Now()
			payment.PaidAt = &now
		}
		payment.Status = entitiesPayment.PaymentStatusApproved
		if err := c.paymentRepo.ApprovePaymentAndOrder(payment, event.OrderId); err != nil {
			if errors.Is(err, apperrors.ErrConflict) {
				// Pedido ja saiu de PENDING_PAYMENT
				return nil
			}
			return err
		}
		c.publishOrderNotification(event, ports.EventOrderPaymentApproved)
		log.Printf("[INFO] [PaymentSyncConsumer] Pagamento aprovado sincronizado para payment=%s order=%s", event.PaymentId, event.OrderId)
		return nil

	case ports.PaymentSyncActionFail:
		payment.Status = entitiesPayment.PaymentStatusFailed
		if err := c.paymentRepo.FailOrExpirePaymentAndCancelOrder(payment, event.OrderId); err != nil {
			if errors.Is(err, apperrors.ErrConflict) {
				return nil
			}
			return err
		}
		c.publishOrderNotification(event, ports.EventOrderPaymentFailed)
		log.Printf("[INFO] [PaymentSyncConsumer] Pagamento recusado sincronizado para payment=%s order=%s", event.PaymentId, event.OrderId)
		return nil

	default:
		return apperrors.ErrInvalidInput
	}
}

func (c *PaymentSyncConsumer) publishOrderNotification(event ports.PaymentSyncEvent, eventType string) {
	if c.orderEventQueue == nil || event.UserEmail == "" {
		return
	}
	if err := c.orderEventQueue.PublishOrderNotification(ports.OrderNotificationEvent{
		EventType:     eventType,
		OrderId:       event.OrderId,
		UserEmail:     event.UserEmail,
		UserName:      event.UserName,
		OrderTotal:    event.Amount,
		PaymentMethod: event.Method,
	}); err != nil {
		log.Printf("[ERROR] [PaymentSyncConsumer] Falha ao publicar notificacao %s para order=%s: %v", eventType, event.OrderId, err)
	}
}
