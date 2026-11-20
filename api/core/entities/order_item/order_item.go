package entities

import (
	entitiesProduct "e-commerce/core/entities/product"

	"github.com/google/uuid"
)

// OrderItem representa a tabela order_items (sem timestamps no schema)
type OrderItem struct {
	Id        uuid.UUID
	OrderId   uuid.UUID
	Product   *entitiesProduct.Product
	Quantity  int64
	UnitPrice float64
}

// Subtotal retorna quantity * unit_price do item
func (item *OrderItem) Subtotal() float64 {
	return float64(item.Quantity) * item.UnitPrice
}

type OrderItemRepository interface {
	AddOrderItem(item OrderItem) error
	ReadOrderItemById(ID string) (*OrderItem, error)
	ReadOrderItemsByOrderId(orderId string) ([]OrderItem, error)
	UpdateOrderItemQuantity(orderId string, orderItemId string, quantity int64) error
	RemoveOrderItem(orderId string, orderItemId string) error
}
