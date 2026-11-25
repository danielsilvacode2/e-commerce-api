package controller

import (
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/controller/order/dto"
	"e-commerce/adapters/controller/order/mapper"
	redisAdapter "e-commerce/adapters/redis"
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

// CreateOrderMe faz o checkout dos itens selecionados do carrinho do usuario logado.
// @Summary      Criar pedido (checkout)
// @Description  Cria um novo pedido a partir dos itens selecionados no carrinho do usuario logado.
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Param        request  body      dto.OrderCreateRequestDTO  true  "Dados para criacao do pedido (CEP)"
// @Success      201      {object}  helpers.SwaggerDataResponse[dto.OrderResponseDTO]
// @Failure      400      {object}  helpers.SwaggerValidationErrorResponse
// @Failure      401      {object}  helpers.SwaggerMessageResponse
// @Failure      422      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /orders [post]
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

	redisAdapter.InvalidateProductCache(c.Request.Context())
	helpers.HelperResponse(mapper.OrderToDto(*order), http.StatusCreated, "create order", c)
}

// ReadAllOrdersMe lista os pedidos do usuario autenticado de forma paginada.
// @Summary      Listar meus pedidos
// @Description  Retorna os pedidos do usuario logado com suporte a paginacao e filtro por status.
// @Tags         Orders
// @Produce      json
// @Security     CookieAuth
// @Param        page    query     int     false  "Numero da pagina"  default(1)
// @Param        limit   query     int     false  "Itens por pagina"  default(10)
// @Param        status  query     string  false  "Filtro por status do pedido"
// @Success      200     {object}  helpers.SwaggerDataResponse[dto.PaginatedOrdersDTO]
// @Failure      401     {object}  helpers.SwaggerMessageResponse
// @Failure      500     {object}  helpers.SwaggerMessageResponse
// @Router       /orders/me [get]
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

// ReadAllOrders lista todos os pedidos da plataforma (somente admin).
// @Summary      Listar todos os pedidos (admin)
// @Description  Retorna todos os pedidos com suporte a paginacao e filtros por status e user_id. Requer perfil admin.
// @Tags         Orders
// @Produce      json
// @Security     CookieAuth
// @Param        page     query     int     false  "Numero da pagina"       default(1)
// @Param        limit    query     int     false  "Itens por pagina"       default(10)
// @Param        status   query     string  false  "Filtro por status"
// @Param        user_id  query     string  false  "Filtro por ID do usuario (UUID)"
// @Success      200      {object}  helpers.SwaggerDataResponse[dto.PaginatedOrdersDTO]
// @Failure      401      {object}  helpers.SwaggerMessageResponse
// @Failure      403      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /orders [get]
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

// ReadOrderById retorna um pedido especifico pelo ID.
// @Summary      Buscar pedido por ID
// @Description  Retorna os detalhes de um pedido especifico (proprietario ou admin).
// @Tags         Orders
// @Produce      json
// @Security     CookieAuth
// @Param        id   path      string  true  "ID do pedido (UUID)"
// @Success      200  {object}  helpers.SwaggerDataResponse[dto.OrderResponseDTO]
// @Failure      401  {object}  helpers.SwaggerMessageResponse
// @Failure      403  {object}  helpers.SwaggerMessageResponse
// @Failure      404  {object}  helpers.SwaggerMessageResponse
// @Failure      500  {object}  helpers.SwaggerMessageResponse
// @Router       /orders/{id} [get]
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

// UpdateOrder atualiza o endereco (CEP) de um pedido.
// @Summary      Atualizar pedido
// @Description  Atualiza o CEP de entrega de um pedido elegivel (proprietario ou admin).
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Param        id       path      string                     true  "ID do pedido (UUID)"
// @Param        request  body      dto.OrderUpdateRequestDTO  true  "Novo CEP do pedido"
// @Success      204      "Pedido atualizado com sucesso"
// @Failure      400      {object}  helpers.SwaggerValidationErrorResponse
// @Failure      401      {object}  helpers.SwaggerMessageResponse
// @Failure      403      {object}  helpers.SwaggerMessageResponse
// @Failure      404      {object}  helpers.SwaggerMessageResponse
// @Failure      422      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /orders/{id} [put]
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

// UpdateOrderStatus atualiza o status e codigo de rastreamento de um pedido (somente admin).
// @Summary      Atualizar status do pedido (admin)
// @Description  Altera o status de um pedido e opcionalmente informa o codigo de rastreio. Requer perfil admin.
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Param        id       path      string                     true  "ID do pedido (UUID)"
// @Param        request  body      dto.OrderStatusRequestDTO  true  "Novo status e codigo de rastreamento"
// @Success      204      "Status atualizado com sucesso"
// @Failure      400      {object}  helpers.SwaggerValidationErrorResponse
// @Failure      401      {object}  helpers.SwaggerMessageResponse
// @Failure      403      {object}  helpers.SwaggerMessageResponse
// @Failure      404      {object}  helpers.SwaggerMessageResponse
// @Failure      422      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /orders/{id}/status [put]
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

// CancelOrder cancela um pedido elegivel.
// @Summary      Cancelar pedido
// @Description  Cancela um pedido pendente (proprietario ou admin).
// @Tags         Orders
// @Produce      json
// @Security     CookieAuth
// @Param        id   path  string  true  "ID do pedido (UUID)"
// @Success      204  "Pedido cancelado com sucesso"
// @Failure      401  {object}  helpers.SwaggerMessageResponse
// @Failure      403  {object}  helpers.SwaggerMessageResponse
// @Failure      404  {object}  helpers.SwaggerMessageResponse
// @Failure      422  {object}  helpers.SwaggerMessageResponse
// @Failure      500  {object}  helpers.SwaggerMessageResponse
// @Router       /orders/{id}/cancel [put]
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

	redisAdapter.InvalidateProductCache(c.Request.Context())
	helpers.HelperResponse(nil, http.StatusNoContent, "order successfully canceled", c)
}

// RequestRefund aplica a regra de estorno: automatico em PREPARING, bloqueado em SHIPPED e pendente de aprovacao admin em DELIVERED.
// @Summary      Solicitar estorno de pedido
// @Description  Solicita estorno de um pedido: automatico quando em PREPARING, bloqueado quando SHIPPED e pendente de aprovacao admin quando DELIVERED (motivo obrigatorio).
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Param        id       path      string                     true   "ID do pedido (UUID)"
// @Param        request  body      dto.OrderRefundRequestDTO  false  "Motivo do estorno (obrigatorio para pedidos entregues)"
// @Success      200      {object}  helpers.SwaggerDataResponse[dto.OrderResponseDTO]
// @Failure      400      {object}  helpers.SwaggerMessageResponse
// @Failure      401      {object}  helpers.SwaggerMessageResponse
// @Failure      403      {object}  helpers.SwaggerMessageResponse
// @Failure      404      {object}  helpers.SwaggerMessageResponse
// @Failure      422      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /orders/{id}/refund [post]
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

	redisAdapter.InvalidateProductCache(c.Request.Context())
	helpers.HelperResponse(mapper.OrderToDto(*order), http.StatusOK, "refund processed", c)
}

// ApproveRefund efetiva um estorno solicitado pelo cliente (somente admin).
// @Summary      Aprovar estorno de pedido (admin)
// @Description  Aprova e efetiva uma solicitacao de estorno pendente. Requer perfil admin.
// @Tags         Orders
// @Produce      json
// @Security     CookieAuth
// @Param        id   path      string  true  "ID do pedido (UUID)"
// @Success      200  {object}  helpers.SwaggerDataResponse[dto.OrderResponseDTO]
// @Failure      401  {object}  helpers.SwaggerMessageResponse
// @Failure      403  {object}  helpers.SwaggerMessageResponse
// @Failure      404  {object}  helpers.SwaggerMessageResponse
// @Failure      422  {object}  helpers.SwaggerMessageResponse
// @Failure      500  {object}  helpers.SwaggerMessageResponse
// @Router       /orders/{id}/refund/approve [put]
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

	redisAdapter.InvalidateProductCache(c.Request.Context())
	helpers.HelperResponse(mapper.OrderToDto(*order), http.StatusOK, "refund approved", c)
}

// RejectRefund recusa um estorno solicitado pelo cliente com justificativa e retorna o pedido para DELIVERED (somente admin).
// @Summary      Recusar estorno de pedido (admin)
// @Description  Recusa uma solicitacao de estorno pendente com justificativa e retorna o pedido para DELIVERED. Requer perfil admin.
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Param        id       path      string                           true  "ID do pedido (UUID)"
// @Param        request  body      dto.OrderRefundRejectRequestDTO  true  "Justificativa da recusa"
// @Success      200      {object}  helpers.SwaggerDataResponse[dto.OrderResponseDTO]
// @Failure      400      {object}  helpers.SwaggerValidationErrorResponse
// @Failure      401      {object}  helpers.SwaggerMessageResponse
// @Failure      403      {object}  helpers.SwaggerMessageResponse
// @Failure      404      {object}  helpers.SwaggerMessageResponse
// @Failure      422      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /orders/{id}/refund/reject [put]
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

// DeleteOrder remove um pedido pelo ID (somente admin).
// @Summary      Remover pedido (admin)
// @Description  Exclui um pedido pelo ID. Requer perfil admin.
// @Tags         Orders
// @Produce      json
// @Security     CookieAuth
// @Param        id   path  string  true  "ID do pedido (UUID)"
// @Success      204  "Pedido removido com sucesso"
// @Failure      401  {object}  helpers.SwaggerMessageResponse
// @Failure      403  {object}  helpers.SwaggerMessageResponse
// @Failure      404  {object}  helpers.SwaggerMessageResponse
// @Failure      500  {object}  helpers.SwaggerMessageResponse
// @Router       /orders/{id} [delete]
func (orderCtrl *OrderController) DeleteOrder(c *gin.Context) {

	if err := orderCtrl.deleteOrder.UseCase(c.Param("id")); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	redisAdapter.InvalidateProductCache(c.Request.Context())
	helpers.HelperResponse(nil, http.StatusNoContent, "order successfully deleted", c)
}

// ReadOrderItems lista os itens de um pedido.
// @Summary      Listar itens do pedido
// @Description  Retorna todos os itens vinculados a um pedido (proprietario ou admin).
// @Tags         Orders
// @Produce      json
// @Security     CookieAuth
// @Param        id   path      string  true  "ID do pedido (UUID)"
// @Success      200  {object}  helpers.SwaggerDataResponse[[]dto.OrderItemResponseDTO]
// @Failure      401  {object}  helpers.SwaggerMessageResponse
// @Failure      403  {object}  helpers.SwaggerMessageResponse
// @Failure      404  {object}  helpers.SwaggerMessageResponse
// @Failure      500  {object}  helpers.SwaggerMessageResponse
// @Router       /orders/{id}/items [get]
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

// AddOrderItem adiciona um item a um pedido existente.
// @Summary      Adicionar item ao pedido
// @Description  Adiciona um produto e quantidade a um pedido elegivel (proprietario ou admin).
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Param        id       path      string                   true  "ID do pedido (UUID)"
// @Param        request  body      dto.OrderItemRequestDTO  true  "Produto e quantidade"
// @Success      201      {object}  helpers.SwaggerMessageResponse
// @Failure      400      {object}  helpers.SwaggerValidationErrorResponse
// @Failure      401      {object}  helpers.SwaggerMessageResponse
// @Failure      403      {object}  helpers.SwaggerMessageResponse
// @Failure      404      {object}  helpers.SwaggerMessageResponse
// @Failure      422      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /orders/{id}/items [post]
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

	redisAdapter.InvalidateProductCache(c.Request.Context())
	helpers.HelperResponse(nil, http.StatusCreated, "product add order", c)
}

// UpdateOrderItem atualiza a quantidade de um item do pedido.
// @Summary      Atualizar item do pedido
// @Description  Altera a quantidade de um item especifico do pedido (proprietario ou admin).
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Param        id       path      string                         true  "ID do pedido (UUID)"
// @Param        itemId   path      string                         true  "ID do item do pedido (UUID)"
// @Param        request  body      dto.OrderItemUpdateRequestDTO  true  "Nova quantidade do item"
// @Success      204      "Item atualizado com sucesso"
// @Failure      400      {object}  helpers.SwaggerValidationErrorResponse
// @Failure      401      {object}  helpers.SwaggerMessageResponse
// @Failure      403      {object}  helpers.SwaggerMessageResponse
// @Failure      404      {object}  helpers.SwaggerMessageResponse
// @Failure      422      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /orders/{id}/items/{itemId} [put]
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

	redisAdapter.InvalidateProductCache(c.Request.Context())
	helpers.HelperResponse(nil, http.StatusNoContent, "order item successfully update", c)
}

// RemoveOrderItem remove um item de um pedido.
// @Summary      Remover item do pedido
// @Description  Remove um item especifico de um pedido elegivel (proprietario ou admin).
// @Tags         Orders
// @Produce      json
// @Security     CookieAuth
// @Param        id      path  string  true  "ID do pedido (UUID)"
// @Param        itemId  path  string  true  "ID do item do pedido (UUID)"
// @Success      204     "Item removido com sucesso"
// @Failure      401     {object}  helpers.SwaggerMessageResponse
// @Failure      403     {object}  helpers.SwaggerMessageResponse
// @Failure      404     {object}  helpers.SwaggerMessageResponse
// @Failure      422     {object}  helpers.SwaggerMessageResponse
// @Failure      500     {object}  helpers.SwaggerMessageResponse
// @Router       /orders/{id}/items/{itemId} [delete]
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

	redisAdapter.InvalidateProductCache(c.Request.Context())
	helpers.HelperResponse(nil, http.StatusNoContent, "remove order item", c)
}

// ReadOrderHistory lista o historico de alteracoes de status de um pedido.
// @Summary      Consultar historico de status do pedido
// @Description  Retorna o historico cronologico de mudancas de status de um pedido (proprietario ou admin).
// @Tags         Orders
// @Produce      json
// @Security     CookieAuth
// @Param        id   path      string  true  "ID do pedido (UUID)"
// @Success      200  {object}  helpers.SwaggerDataResponse[[]dto.OrderStatusHistoryResponseDTO]
// @Failure      401  {object}  helpers.SwaggerMessageResponse
// @Failure      403  {object}  helpers.SwaggerMessageResponse
// @Failure      404  {object}  helpers.SwaggerMessageResponse
// @Failure      500  {object}  helpers.SwaggerMessageResponse
// @Router       /orders/{id}/history [get]
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
