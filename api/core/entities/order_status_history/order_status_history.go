package entities

import (
	"time"

	"github.com/google/uuid"
)

// OrderStatusHistory representa a tabela order_status_history.
// Os registros sao gravados pelo OrderRepository, na mesma transacao da mudanca de status.
type OrderStatusHistory struct {
	Id       uuid.UUID
	OrderId  uuid.UUID
	Status   string
	CreateAt time.Time
}

type OrderStatusHistoryRepository interface {
	ReadHistoryByOrderId(orderId string) ([]OrderStatusHistory, error)
}
