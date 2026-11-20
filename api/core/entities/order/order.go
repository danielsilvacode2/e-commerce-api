package entities

import (
	entitiesOrderItem "e-commerce/core/entities/order_item"
	entitiesUser "e-commerce/core/entities/user"
	"math"
	"time"

	"github.com/google/uuid"
)

const (
	OrderStatusPreparing = "PREPARING"
	OrderStatusShipped   = "SHIPPED"
	OrderStatusDelivered = "DELIVERED"
	OrderStatusCanceled  = "CANCELED"
)

// orderStatusTransitions define as transicoes validas de status do pedido.
// DELIVERED e CANCELED sao status finais (sem transicoes).
var orderStatusTransitions = map[string][]string{
	OrderStatusPreparing: {OrderStatusShipped, OrderStatusCanceled},
	OrderStatusShipped:   {OrderStatusDelivered, OrderStatusCanceled},
}

type Order struct {
	Id       uuid.UUID
	User     *entitiesUser.User
	Status   string
	ZipCode  string
	Items    []entitiesOrderItem.OrderItem
	UpdateAt time.Time
	CreateAt time.Time
}

type OrderFilter struct {
	Page   int
	Limit  int
	Status string
	UserId string
}

type PaginatedOrders struct {
	Data       []Order
	Page       int
	Limit      int
	TotalRows  int64
	TotalPages int
}

// IsValidOrderStatus verifica se o status existe no ENUM da tabela orders
func IsValidOrderStatus(status string) bool {
	switch status {
	case OrderStatusPreparing, OrderStatusShipped, OrderStatusDelivered, OrderStatusCanceled:
		return true
	}
	return false
}

// IsFinalized indica se o pedido nao pode mais ser alterado
func (o *Order) IsFinalized() bool {
	return o.Status == OrderStatusDelivered || o.Status == OrderStatusCanceled
}

// IsEditable indica se itens e CEP do pedido ainda podem ser alterados
func (o *Order) IsEditable() bool {
	return o.Status == OrderStatusPreparing
}

// HasReservedStock indica se o estoque dos itens ainda esta reservado (nao entregue nem devolvido)
func (o *Order) HasReservedStock() bool {
	return o.Status == OrderStatusPreparing || o.Status == OrderStatusShipped
}

// CanTransitionTo verifica se a transicao do status atual para o novo status e valida
func (o *Order) CanTransitionTo(status string) bool {
	for _, next := range orderStatusTransitions[o.Status] {
		if next == status {
			return true
		}
	}
	return false
}

// BelongsTo verifica se o pedido pertence ao usuario informado
func (o *Order) BelongsTo(userId string) bool {
	return o.User != nil && o.User.ID.String() == userId
}

// Total retorna a soma dos subtotais dos itens, arredondada em 2 casas
func (o *Order) Total() float64 {
	var total float64
	for _, item := range o.Items {
		total += item.Subtotal()
	}
	return math.Round(total*100) / 100
}

type OrderRepository interface {
	CreateOrder(order Order, cartProductIds []string) error
	ReadOrderById(ID string) (*Order, error)
	ReadAllOrders(filter OrderFilter) (*PaginatedOrders, error)
	UpdateOrderZipCode(zipCode string, ID string) error
	UpdateOrderStatus(ID string, currentStatus string, newStatus string, restoreStock bool) error
	DeleteOrder(ID string, restoreStock bool) error
}
