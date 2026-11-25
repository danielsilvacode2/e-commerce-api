package controller

import (
	"e-commerce/adapters/controller/cart/dto"
	"e-commerce/adapters/controller/cart/mapper"
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/security/auth"
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/cart"
	use_cases "e-commerce/core/use_cases/cart_product"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CartController struct {
	AddProductCase        use_cases.AddProductUseCase
	ReadAllCartCase       use_cases.ReadAllCartProductsUseCase
	RemoveProductCase     use_cases.RemoveProductCartUseCase
	UpdateIsSelectCase    use_cases.UpdateIsSelectUseCase
	UpdateIsAllSelectCase use_cases.UpdateAllIsSelectUseCase
	DecreaseProductCase   use_cases.DecreaseProductCartUseCase
}

func NewCartController(
	AddProductCase use_cases.AddProductUseCase,
	ReadAllCartCase use_cases.ReadAllCartProductsUseCase,
	RemoveProductCase use_cases.RemoveProductCartUseCase,
	UpdateIsSelectCase use_cases.UpdateIsSelectUseCase,
	UpdateIsAllSelectCase use_cases.UpdateAllIsSelectUseCase,
	DecreaseProductCase use_cases.DecreaseProductCartUseCase,
) *CartController {

	return &CartController{
		AddProductCase:        AddProductCase,
		ReadAllCartCase:       ReadAllCartCase,
		RemoveProductCase:     RemoveProductCase,
		UpdateIsSelectCase:    UpdateIsSelectCase,
		UpdateIsAllSelectCase: UpdateIsAllSelectCase,
		DecreaseProductCase:   DecreaseProductCase,
	}
}

func (cartCtrl *CartController) getClaims(c *gin.Context) (*auth.Claims, error) {

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

// AddProductCartMe adiciona um produto ao carrinho do usuario autenticado.
// @Summary      Adicionar produto ao carrinho
// @Description  Adiciona um produto (ou incrementa sua quantidade) no carrinho do usuario logado.
// @Tags         Cart
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Param        request  body      dto.CartRequestDTO  true  "Produto e quantidade"
// @Success      201      {object}  helpers.SwaggerMessageResponse
// @Failure      400      {object}  helpers.SwaggerValidationErrorResponse
// @Failure      401      {object}  helpers.SwaggerMessageResponse
// @Failure      404      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /cart [post]
func (cartCtrl *CartController) AddProductCartMe(c *gin.Context) {

	claims, err := cartCtrl.getClaims(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	userId := claims.Id.String()

	var cartRequest dto.CartRequestDTO

	if err := c.ShouldBindJSON(&cartRequest); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	if listErrField := cartRequest.Validator(); listErrField != nil {
		helpers.HelperErrResponse(listErrField, c, apperrors.ErrInvalidInput)
		return
	}

	cartProduct, err := mapper.DtoRequestToCartProduct(cartRequest)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	if err = cartCtrl.AddProductCase.UseCase(*cartProduct, userId); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, http.StatusCreated, "product add cart", c)

}

// DecreaseProductCartMe diminui a quantidade de um produto no carrinho do usuario autenticado.
// @Summary      Diminuir quantidade de item no carrinho
// @Description  Reduz a quantidade de um item especifico no carrinho do usuario logado.
// @Tags         Cart
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Param        id       path      string                      true  "ID do item no carrinho (UUID)"
// @Param        request  body      dto.CartRequestDecreaseDTO  true  "Quantidade a reduzir"
// @Success      200      {object}  helpers.SwaggerMessageResponse
// @Failure      400      {object}  helpers.SwaggerMessageResponse
// @Failure      401      {object}  helpers.SwaggerMessageResponse
// @Failure      404      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /cart/decrease/{id} [put]
func (cartCtrl *CartController) DecreaseProductCartMe(c *gin.Context) {

	claims, err := cartCtrl.getClaims(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	userId := claims.Id.String()

	var cartRequestDto dto.CartRequestDecreaseDTO

	if err = c.ShouldBindJSON(&cartRequestDto); err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return

	}

	id := c.Param("id")

	if err = cartCtrl.DecreaseProductCase.UseCase(id, cartRequestDto.Quantity, userId); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, http.StatusOK, "update decrease", c)

}

// ReadAllProductCartMe lista todos os produtos no carrinho do usuario autenticado.
// @Summary      Listar itens do carrinho
// @Description  Retorna o carrinho e todos os itens adicionados pelo usuario logado.
// @Tags         Cart
// @Produce      json
// @Security     CookieAuth
// @Success      200  {object}  helpers.SwaggerDataResponse[dto.CartResponseDTO]
// @Failure      401  {object}  helpers.SwaggerMessageResponse
// @Failure      500  {object}  helpers.SwaggerMessageResponse
// @Router       /cart [get]
func (cartCtrl *CartController) ReadAllProductCartMe(c *gin.Context) {

	claims, err := cartCtrl.getClaims(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	userId := claims.Id.String()

	listCartProducts, err := cartCtrl.ReadAllCartCase.UseCase(userId)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	var listCartProductsDTO []dto.ProcuctCartResponseDTO

	if listCartProducts == nil {
		helpers.HelperResponse(listCartProductsDTO, http.StatusOK, "list is empty", c)
		return
	}

	var cart *entities.Cart

	for _, cartProduct := range listCartProducts {

		if cart == nil {
			cart = cartProduct.Cart
		}

		cartProductDTO := mapper.ProductCartToDto(cartProduct)
		listCartProductsDTO = append(listCartProductsDTO, cartProductDTO)
	}

	userDTO := mapper.UserToResponseCartDTO(*cart)

	cartResponse := mapper.CartResponseToDTO(userDTO, cart.Id, listCartProductsDTO)
	helpers.HelperResponse(cartResponse, http.StatusOK, "read all products cart", c)
}

// RemoveProductCartMe remove um item do carrinho do usuario autenticado.
// @Summary      Remover item do carrinho
// @Description  Remove um produto do carrinho do usuario logado.
// @Tags         Cart
// @Produce      json
// @Security     CookieAuth
// @Param        id   path  string  true  "ID do item no carrinho (UUID)"
// @Success      204  "Item removido com sucesso"
// @Failure      401  {object}  helpers.SwaggerMessageResponse
// @Failure      404  {object}  helpers.SwaggerMessageResponse
// @Failure      500  {object}  helpers.SwaggerMessageResponse
// @Router       /cart/{id} [delete]
func (cartCtrl *CartController) RemoveProductCartMe(c *gin.Context) {
	claims, err := cartCtrl.getClaims(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	userId := claims.Id.String()

	cartProductID := c.Param("id")
	if err = cartCtrl.RemoveProductCase.UseCase(cartProductID, userId); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, http.StatusNoContent, "remove product", c)
}

// UpdateIsSelectProductMe marca ou desmarca um item do carrinho para checkout.
// @Summary      Selecionar/desmarcar item do carrinho
// @Description  Atualiza o estado de selecao de um item especifico no carrinho para o checkout.
// @Tags         Cart
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Param        id       path      string                  true  "ID do item no carrinho (UUID)"
// @Param        request  body      dto.IsSelectRequestDTO  true  "Estado de selecao"
// @Success      204      "Selecao atualizada com sucesso"
// @Failure      400      {object}  helpers.SwaggerMessageResponse
// @Failure      401      {object}  helpers.SwaggerMessageResponse
// @Failure      404      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /cart/{id} [put]
func (cartCtrl *CartController) UpdateIsSelectProductMe(c *gin.Context) {
	claims, err := cartCtrl.getClaims(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	userId := claims.Id.String()

	var isSelectDTO dto.IsSelectRequestDTO

	if err = c.ShouldBindJSON(&isSelectDTO); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	cartProductID := c.Param("id")

	if err = cartCtrl.UpdateIsSelectCase.UseCase(isSelectDTO.IsSelect, userId, cartProductID); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, http.StatusNoContent, "update is select", c)

}

// UpdateIsAllSelectProductMe marca ou desmarca todos os itens do carrinho.
// @Summary      Selecionar/desmarcar todos os itens do carrinho
// @Description  Atualiza o estado de selecao de todos os itens no carrinho do usuario logado.
// @Tags         Cart
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Param        request  body      dto.IsSelectRequestDTO  true  "Estado de selecao"
// @Success      204      "Selecao atualizada com sucesso"
// @Failure      400      {object}  helpers.SwaggerMessageResponse
// @Failure      401      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /cart [put]
func (cartCtrl *CartController) UpdateIsAllSelectProductMe(c *gin.Context) {

	claims, err := cartCtrl.getClaims(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	userID := claims.Id.String()

	var isSelectDTO dto.IsSelectRequestDTO

	if err = c.ShouldBindJSON(&isSelectDTO); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	if err = cartCtrl.UpdateIsAllSelectCase.UseCase(isSelectDTO.IsSelect, userID); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, http.StatusNoContent, "updateIsSelect", c)

}
