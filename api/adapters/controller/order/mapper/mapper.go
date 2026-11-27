package mapper

import (
	"e-commerce/adapters/controller/order/dto"
	entitiesAddress "e-commerce/core/entities/address"
	entitiesOrder "e-commerce/core/entities/order"
	entitiesOrderItem "e-commerce/core/entities/order_item"
	entitiesOrderStatusHistory "e-commerce/core/entities/order_status_history"
	entitiesProduct "e-commerce/core/entities/product"
	"math"

	"github.com/google/uuid"
)

func DtoRequestToAddress(req dto.OrderAddressRequestDTO) entitiesAddress.Address {
	return entitiesAddress.Address{
		Cep:          req.Cep,
		Street:       req.Street,
		Number:       req.Number,
		Complement:   req.Complement,
		Neighborhood: req.Neighborhood,
		City:         req.City,
		State:        req.State,
	}
}

func AddressToDto(address *entitiesAddress.Address) *dto.OrderAddressResponseDTO {
	if address == nil {
		return nil
	}
	return &dto.OrderAddressResponseDTO{
		Id:           address.Id,
		Cep:          address.Cep,
		Street:       address.Street,
		Number:       address.Number,
		Complement:   address.Complement,
		Neighborhood: address.Neighborhood,
		City:         address.City,
		State:        address.State,
	}
}

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
	return OrderToDtoWithWarning(order, "")
}

func OrderToDtoWithWarning(order entitiesOrder.Order, warning string) dto.OrderResponseDTO {

	listItemsDTO := make([]dto.OrderItemResponseDTO, 0, len(order.Items))

	for _, item := range order.Items {
		listItemsDTO = append(listItemsDTO, OrderItemToDto(item))
	}

	orderDTO := dto.OrderResponseDTO{
		Id:                       order.Id,
		Address:                  AddressToDto(order.Address),
		Status:                   order.Status,
		AddressValidationWarning: warning,
		TrackingCode:             order.TrackingCode,
		RefundReason:             order.RefundReason,
		RefundRejectionReason:    order.RefundRejectionReason,
		Total:                    order.Total(),
		Items:                    listItemsDTO,
		UpdateAt:                 order.UpdateAt,
		CreateAt:                 order.CreateAt,
	}

	if order.User != nil {
		orderDTO.User.Id = order.User.ID
		orderDTO.User.Name = order.User.Name
		orderDTO.User.Email = order.User.Email
	}

	return orderDTO
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
