package controller

import (
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/controller/order/dto"
	"e-commerce/adapters/controller/order/mapper"
	"e-commerce/adapters/security/auth"
	"e-commerce/core/apperrors"
	entitiesOrder "e-commerce/core/entities/order"
	useCasesOrder "e-commerce/core/use_cases/order"
	useCasesOrderItem "e-commerce/core/use_cases/order_item"
	useCasesOrderStatusHistory "e-commerce/core/use_cases/order_status_history"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type OrderController struct {
	createOrder        *useCasesOrder.CreateOrderUseCase
	readOrderById      *useCasesOrder.ReadOrderByIdUseCase
	readAllOrders      *useCasesOrder.ReadAllOrdersUseCase
	readOrdersByUserId *useCasesOrder.ReadOrdersByUserIdUseCase
	updateOrder        *useCasesOrder.UpdateOrderUseCase
	updateOrderStatus  *useCasesOrder.UpdateOrderStatusUseCase
	cancelOrder        *useCasesOrder.CancelOrderUseCase
	requestRefund      *useCasesOrder.RequestRefundUseCase
	approveRefund      *useCasesOrder.ApproveRefundUseCase
	rejectRefund       *useCasesOrder.RejectRefundUseCase
	deleteOrder        *useCasesOrder.DeleteOrderUseCase
	readOrderItems     *useCasesOrderItem.ReadOrderItemsUseCase
	addOrderItem       *useCasesOrderItem.AddOrderItemUseCase
	updateOrderItem    *useCasesOrderItem.UpdateOrderItemUseCase
	removeOrderItem    *useCasesOrderItem.RemoveOrderItemUseCase
	readOrderHistory   *useCasesOrderStatusHistory.ReadOrderStatusHistoryUseCase
}

func NewOrderController(
	createOrder *useCasesOrder.CreateOrderUseCase,
	readOrderById *useCasesOrder.ReadOrderByIdUseCase,
	readAllOrders *useCasesOrder.ReadAllOrdersUseCase,
	readOrdersByUserId *useCasesOrder.ReadOrdersByUserIdUseCase,
	updateOrder *useCasesOrder.UpdateOrderUseCase,
	updateOrderStatus *useCasesOrder.UpdateOrderStatusUseCase,
	cancelOrder *useCasesOrder.CancelOrderUseCase,
	requestRefund *useCasesOrder.RequestRefundUseCase,
	approveRefund *useCasesOrder.ApproveRefundUseCase,
	rejectRefund *useCasesOrder.RejectRefundUseCase,
	deleteOrder *useCasesOrder.DeleteOrderUseCase,
	readOrderItems *useCasesOrderItem.ReadOrderItemsUseCase,
	addOrderItem *useCasesOrderItem.AddOrderItemUseCase,
	updateOrderItem *useCasesOrderItem.UpdateOrderItemUseCase,
	removeOrderItem *useCasesOrderItem.RemoveOrderItemUseCase,
	readOrderHistory *useCasesOrderStatusHistory.ReadOrderStatusHistoryUseCase,
) *OrderController {

	return &OrderController{
		createOrder:        createOrder,
		readOrderById:      readOrderById,
		readAllOrders:      readAllOrders,
		readOrdersByUserId: readOrdersByUserId,
		updateOrder:        updateOrder,
		updateOrderStatus:  updateOrderStatus,
		cancelOrder:        cancelOrder,
		requestRefund:      requestRefund,
		approveRefund:      approveRefund,
		rejectRefund:       rejectRefund,
		deleteOrder:        deleteOrder,
		readOrderItems:     readOrderItems,
		addOrderItem:       addOrderItem,
		updateOrderItem:    updateOrderItem,
		removeOrderItem:    removeOrderItem,
		readOrderHistory:   readOrderHistory,
	}
}

func (orderCtrl *OrderController) getClaims(c *gin.Context) (*auth.Claims, error) {

	val, exists := c.Get("claims")

	if !exists {
		return nil, apperrors.ErrUnauthorized
	}

	claims, ok := val.(*auth.Claims)

	if !ok {
		return nil, errors.New("Error converting declarations")
	}

	return claims, nil
}

func (orderCtrl *OrderController) getPagination(c *gin.Context) (int, int) {

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	return page, limit
}

// CreateOrderMe faz o checkout dos itens selecionados do carrinho do usuario logado
func (orderCtrl *OrderController) CreateOrderMe(c *gin.Context) {

	claims, err := orderCtrl.getClaims(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	var dtoRequest dto.OrderCreateRequestDTO

	if err = c.ShouldBindJSON(&dtoRequest); err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	if listErrField := dtoRequest.Validator(); listErrField != nil {
		helpers.HelperErrResponse(listErrField, c, apperrors.ErrInvalidInput)
		return
	}

	order, err := orderCtrl.createOrder.UseCase(strings.TrimSpace(dtoRequest.ZipCode), claims.Id.String())

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(mapper.OrderToDto(*order), http.StatusCreated, "create order", c)
}

func (orderCtrl *OrderController) ReadAllOrdersMe(c *gin.Context) {

	claims, err := orderCtrl.getClaims(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	page, limit := orderCtrl.getPagination(c)
	status := strings.ToUpper(c.Query("status"))

	paginated, err := orderCtrl.readOrdersByUserId.UseCase(claims.Id.String(), page, limit, status)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(mapper.PaginatedOrdersToDto(*paginated), http.StatusOK, "list orders", c)
}

func (orderCtrl *OrderController) ReadAllOrders(c *gin.Context) {

	page, limit := orderCtrl.getPagination(c)

	filter := entitiesOrder.OrderFilter{
		Page:   page,
		Limit:  limit,
		Status: strings.ToUpper(c.Query("status")),
		UserId: c.Query("user_id"),
	}

	paginated, err := orderCtrl.readAllOrders.UseCase(filter)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(mapper.PaginatedOrdersToDto(*paginated), http.StatusOK, "list orders", c)
}

func (orderCtrl *OrderController) ReadOrderById(c *gin.Context) {

	claims, err := orderCtrl.getClaims(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	order, err := orderCtrl.readOrderById.UseCase(c.Param("id"), claims.Id.String(), claims.Role)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(mapper.OrderToDto(*order), http.StatusOK, "found order", c)
}

func (orderCtrl *OrderController) UpdateOrder(c *gin.Context) {

	claims, err := orderCtrl.getClaims(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	var dtoRequest dto.OrderUpdateRequestDTO

	if err = c.ShouldBindJSON(&dtoRequest); err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	if listErrField := dtoRequest.Validator(); listErrField != nil {
		helpers.HelperErrResponse(listErrField, c, apperrors.ErrInvalidInput)
		return
	}

	if err = orderCtrl.updateOrder.UseCase(strings.TrimSpace(dtoRequest.ZipCode), c.Param("id"), claims.Id.String(), claims.Role); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, http.StatusNoContent, "order successfully update", c)
}

func (orderCtrl *OrderController) UpdateOrderStatus(c *gin.Context) {

	claims, err := orderCtrl.getClaims(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	var dtoRequest dto.OrderStatusRequestDTO

	if err = c.ShouldBindJSON(&dtoRequest); err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	if listErrField := dtoRequest.Validator(); listErrField != nil {
		helpers.HelperErrResponse(listErrField, c, apperrors.ErrInvalidInput)
		return
	}

	status := strings.ToUpper(strings.TrimSpace(dtoRequest.Status))

	if err = orderCtrl.updateOrderStatus.UseCase(status, dtoRequest.TrackingCode, c.Param("id"), claims.Id.String(), claims.Role); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, http.StatusNoContent, "order status successfully update", c)
}

func (orderCtrl *OrderController) CancelOrder(c *gin.Context) {

	claims, err := orderCtrl.getClaims(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	if err = orderCtrl.cancelOrder.UseCase(c.Param("id"), claims.Id.String(), claims.Role); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, http.StatusNoContent, "order successfully canceled", c)
}

// RequestRefund aplica a regra de estorno: automatico em PREPARING, bloqueado em SHIPPED e pendente de aprovacao admin em DELIVERED.
func (orderCtrl *OrderController) RequestRefund(c *gin.Context) {

	claims, err := orderCtrl.getClaims(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	var dtoRequest dto.OrderRefundRequestDTO
	_ = c.ShouldBindJSON(&dtoRequest)

	order, err := orderCtrl.requestRefund.UseCase(c.Param("id"), dtoRequest.Reason, claims.Id.String(), claims.Role)

	if err != nil {
		if errors.Is(err, apperrors.ErrUnprocessableEntity) {
			helpers.HelperResponse(nil, http.StatusUnprocessableEntity, "Não é possível solicitar estorno para o status atual deste pedido (pedidos enviados/em trânsito não permitem estorno).", c)
			return
		}
		if errors.Is(err, apperrors.ErrInvalidInput) {
			helpers.HelperResponse(nil, http.StatusBadRequest, "Informe o motivo para solicitar o estorno de um pedido entregue.", c)
			return
		}
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(mapper.OrderToDto(*order), http.StatusOK, "refund processed", c)
}

// ApproveRefund efetiva um estorno solicitado pelo cliente (somente admin).
func (orderCtrl *OrderController) ApproveRefund(c *gin.Context) {

	claims, err := orderCtrl.getClaims(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	order, err := orderCtrl.approveRefund.UseCase(c.Param("id"), claims.Id.String(), claims.Role)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(mapper.OrderToDto(*order), http.StatusOK, "refund approved", c)
}

// RejectRefund recusa um estorno solicitado pelo cliente com justificativa e retorna o pedido para DELIVERED (somente admin).
func (orderCtrl *OrderController) RejectRefund(c *gin.Context) {

	claims, err := orderCtrl.getClaims(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	var dtoRequest dto.OrderRefundRejectRequestDTO

	if err = c.ShouldBindJSON(&dtoRequest); err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	if listErrField := dtoRequest.Validator(); listErrField != nil {
		helpers.HelperErrResponse(listErrField, c, apperrors.ErrInvalidInput)
		return
	}

	order, err := orderCtrl.rejectRefund.UseCase(c.Param("id"), dtoRequest.Reason, claims.Id.String(), claims.Role)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(mapper.OrderToDto(*order), http.StatusOK, "refund rejected", c)
}

func (orderCtrl *OrderController) DeleteOrder(c *gin.Context) {

	if err := orderCtrl.deleteOrder.UseCase(c.Param("id")); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, http.StatusNoContent, "order successfully deleted", c)
}

func (orderCtrl *OrderController) ReadOrderItems(c *gin.Context) {

	claims, err := orderCtrl.getClaims(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	listItems, err := orderCtrl.readOrderItems.UseCase(c.Param("id"), claims.Id.String(), claims.Role)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	var listItemsDTO []dto.OrderItemResponseDTO

	if listItems == nil {
		helpers.HelperResponse(listItemsDTO, http.StatusOK, "list is empty", c)
		return
	}

	for _, item := range listItems {
		listItemsDTO = append(listItemsDTO, mapper.OrderItemToDto(item))
	}

	helpers.HelperResponse(listItemsDTO, http.StatusOK, "list order items", c)
}

func (orderCtrl *OrderController) AddOrderItem(c *gin.Context) {

	claims, err := orderCtrl.getClaims(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	var dtoRequest dto.OrderItemRequestDTO

	if err = c.ShouldBindJSON(&dtoRequest); err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	if listErrField := dtoRequest.Validator(); listErrField != nil {
		helpers.HelperErrResponse(listErrField, c, apperrors.ErrInvalidInput)
		return
	}

	orderItem, err := mapper.DtoRequestToOrderItem(dtoRequest)

	if err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	if err = orderCtrl.addOrderItem.UseCase(*orderItem, c.Param("id"), claims.Id.String(), claims.Role); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, http.StatusCreated, "product add order", c)
}

func (orderCtrl *OrderController) UpdateOrderItem(c *gin.Context) {

	claims, err := orderCtrl.getClaims(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	var dtoRequest dto.OrderItemUpdateRequestDTO

	if err = c.ShouldBindJSON(&dtoRequest); err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	if listErrField := dtoRequest.Validator(); listErrField != nil {
		helpers.HelperErrResponse(listErrField, c, apperrors.ErrInvalidInput)
		return
	}

	if err = orderCtrl.updateOrderItem.UseCase(dtoRequest.Quantity, c.Param("id"), c.Param("itemId"), claims.Id.String(), claims.Role); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, http.StatusNoContent, "order item successfully update", c)
}

func (orderCtrl *OrderController) RemoveOrderItem(c *gin.Context) {

	claims, err := orderCtrl.getClaims(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	if err = orderCtrl.removeOrderItem.UseCase(c.Param("id"), c.Param("itemId"), claims.Id.String(), claims.Role); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, http.StatusNoContent, "remove order item", c)
}

func (orderCtrl *OrderController) ReadOrderHistory(c *gin.Context) {

	claims, err := orderCtrl.getClaims(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	listHistory, err := orderCtrl.readOrderHistory.UseCase(c.Param("id"), claims.Id.String(), claims.Role)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	var listHistoryDTO []dto.OrderStatusHistoryResponseDTO

	if listHistory == nil {
		helpers.HelperResponse(listHistoryDTO, http.StatusOK, "list is empty", c)
		return
	}

	for _, history := range listHistory {
		listHistoryDTO = append(listHistoryDTO, mapper.OrderStatusHistoryToDto(history))
	}

	helpers.HelperResponse(listHistoryDTO, http.StatusOK, "list order history", c)
}
