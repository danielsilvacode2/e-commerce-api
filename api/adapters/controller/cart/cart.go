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
	AddProductCase     use_cases.AddProductUseCase
	ReadAllCartCase    use_cases.ReadAllCartProductsUseCase
	RemoveProductCase  use_cases.RemoveProductCartUseCase
	UpdateIsSelectCase use_cases.UpdateIsSelectUseCase
}

func NewCartController(
	AddProductCase use_cases.AddProductUseCase,
	ReadAllCartCase use_cases.ReadAllCartProductsUseCase,
	RemoveProductCase use_cases.RemoveProductCartUseCase,
	UpdateIsSelectCase use_cases.UpdateIsSelectUseCase) *CartController {

	return &CartController{
		AddProductCase:     AddProductCase,
		ReadAllCartCase:    ReadAllCartCase,
		RemoveProductCase:  RemoveProductCase,
		UpdateIsSelectCase: UpdateIsSelectCase,
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
