package entities

import (
	entitiesAddress "e-commerce/core/entities/address"
	entitiesOrderItem "e-commerce/core/entities/order_item"
	entitiesUser "e-commerce/core/entities/user"
	"math"
	"time"

	"github.com/google/uuid"
)

const (
	OrderStatusPendingPayment  = "PENDING_PAYMENT"
	OrderStatusPreparing       = "PREPARING"
	OrderStatusShipped         = "SHIPPED"
	OrderStatusDelivered       = "DELIVERED"
	OrderStatusCanceled        = "CANCELED"
	OrderStatusRefundRequested = "REFUND_REQUESTED"
	OrderStatusRefunded        = "REFUNDED"
)

// orderStatusTransitions define as transicoes validas de status do pedido.
// CANCELED e REFUNDED sao status finais (sem transicoes).
var orderStatusTransitions = map[string][]string{
	OrderStatusPendingPayment:  {OrderStatusPreparing, OrderStatusCanceled},
	OrderStatusPreparing:       {OrderStatusShipped, OrderStatusCanceled, OrderStatusRefunded},
	OrderStatusShipped:         {OrderStatusDelivered, OrderStatusCanceled},
	OrderStatusDelivered:       {OrderStatusRefundRequested},
	OrderStatusRefundRequested: {OrderStatusRefunded, OrderStatusDelivered},
}

type Order struct {
	Id                    uuid.UUID
	User                  *entitiesUser.User
	Address               *entitiesAddress.Address
	Status                string
	TrackingCode          string
	RefundReason          string
	RefundRejectionReason string
	Items                 []entitiesOrderItem.OrderItem
	UpdateAt              time.Time
	CreateAt              time.Time
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
	case OrderStatusPendingPayment,
		OrderStatusPreparing,
		OrderStatusShipped,
		OrderStatusDelivered,
		OrderStatusCanceled,
		OrderStatusRefundRequested,
		OrderStatusRefunded:
		return true
	}
	return false
}

// IsFinalized indica se o pedido nao pode mais ser alterado
func (o *Order) IsFinalized() bool {
	return o.Status == OrderStatusCanceled || o.Status == OrderStatusRefunded
}

// IsEditable indica se o endereco ou cancelamento pelo cliente ainda sao permitidos
func (o *Order) IsEditable() bool {
	return o.Status == OrderStatusPendingPayment || o.Status == OrderStatusPreparing
}

// CanModifyItems indica se os itens do pedido ainda podem ser adicionados, alterados ou removidos (somente antes do pagamento)
func (o *Order) CanModifyItems() bool {
	return o.Status == OrderStatusPendingPayment
}

// HasReservedStock indica se o estoque dos itens ainda esta reservado (nao entregue nem devolvido)
func (o *Order) HasReservedStock() bool {
	return o.Status == OrderStatusPendingPayment || o.Status == OrderStatusPreparing || o.Status == OrderStatusShipped
}

// IsAutoRefundable indica se o estorno solicitado pelo cliente e aprovado automaticamente (sem aprovacao manual do admin)
func (o *Order) IsAutoRefundable() bool {
	return o.Status == OrderStatusPreparing
}

// RequiresAdminRefundApproval indica se o estorno solicitado pelo cliente exige aprovacao manual do admin
func (o *Order) RequiresAdminRefundApproval() bool {
	return o.Status == OrderStatusDelivered
}

// CanCustomerRequestRefund indica se o cliente pode solicitar estorno no status atual.
// Bloqueado explicitamente em SHIPPED (em transito), PENDING_PAYMENT, REFUND_REQUESTED, CANCELED e REFUNDED.
func (o *Order) CanCustomerRequestRefund() bool {
	return o.IsAutoRefundable() || o.RequiresAdminRefundApproval()
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
	CreateOrder(order Order, cartProductIds []string, useExistingAddress bool) error
	ReadOrderById(ID string) (*Order, error)
	ReadAllOrders(filter OrderFilter) (*PaginatedOrders, error)
	UpdateOrderAddress(address entitiesAddress.Address, ID string, useExistingAddress bool) error
	UpdateOrderStatus(ID string, currentStatus string, newStatus string, restoreStock bool) error
	UpdateOrderStatusWithDetails(ID string, currentStatus string, newStatus string, restoreStock bool, trackingCode *string, refundReason *string, refundRejectionReason *string) error
	DeleteOrder(ID string, restoreStock bool) error
}
