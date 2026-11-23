package mapper

import (
	"e-commerce/adapters/controller/order/dto"
	entitiesOrder "e-commerce/core/entities/order"
	entitiesOrderItem "e-commerce/core/entities/order_item"
	entitiesOrderStatusHistory "e-commerce/core/entities/order_status_history"
	entitiesProduct "e-commerce/core/entities/product"
	"math"

	"github.com/google/uuid"
)

func OrderItemToDto(item entitiesOrderItem.OrderItem) dto.OrderItemResponseDTO {

	dto := dto.OrderItemResponseDTO{
		Id:        item.Id,
		Quantity:  item.Quantity,
		UnitPrice: item.UnitPrice,
		Subtotal:  math.Round(item.Subtotal()*100) / 100,
	}

	if item.Product != nil {
		dto.ProductId = item.Product.Id
		dto.Name = item.Product.Name
		dto.Sku = item.Product.Sku
		dto.Image = item.Product.Image
	}

	return dto
}

func OrderToDto(order entitiesOrder.Order) dto.OrderResponseDTO {

	listItemsDTO := make([]dto.OrderItemResponseDTO, 0, len(order.Items))

	for _, item := range order.Items {
		listItemsDTO = append(listItemsDTO, OrderItemToDto(item))
	}

	dto := dto.OrderResponseDTO{
		Id:                    order.Id,
		Status:                order.Status,
		ZipCode:               order.ZipCode,
		TrackingCode:          order.TrackingCode,
		RefundReason:          order.RefundReason,
		RefundRejectionReason: order.RefundRejectionReason,
		Total:                 order.Total(),
		Items:                 listItemsDTO,
		UpdateAt:              order.UpdateAt,
		CreateAt:              order.CreateAt,
	}

	if order.User != nil {
		dto.User.Id = order.User.ID
		dto.User.Name = order.User.Name
		dto.User.Email = order.User.Email
	}

	return dto
}

func PaginatedOrdersToDto(paginated entitiesOrder.PaginatedOrders) dto.PaginatedOrdersDTO {

	listOrdersDTO := make([]dto.OrderResponseDTO, 0, len(paginated.Data))

	for _, order := range paginated.Data {
		listOrdersDTO = append(listOrdersDTO, OrderToDto(order))
	}

	return dto.PaginatedOrdersDTO{
		Orders:     listOrdersDTO,
		Page:       paginated.Page,
		Limit:      paginated.Limit,
		TotalRows:  paginated.TotalRows,
		TotalPages: paginated.TotalPages,
	}
}

func OrderStatusHistoryToDto(history entitiesOrderStatusHistory.OrderStatusHistory) dto.OrderStatusHistoryResponseDTO {

	return dto.OrderStatusHistoryResponseDTO{
		Id:       history.Id,
		Status:   history.Status,
		CreateAt: history.CreateAt,
	}
}

func DtoRequestToOrderItem(dto dto.OrderItemRequestDTO) (*entitiesOrderItem.OrderItem, error) {

	productUUID, err := uuid.Parse(dto.ProductId)

	if err != nil {
		return nil, err
	}

	orderItem := entitiesOrderItem.OrderItem{
		Product:  &entitiesProduct.Product{Id: productUUID},
		Quantity: dto.Quantity,
	}

	return &orderItem, nil
}
