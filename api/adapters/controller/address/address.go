package controller

import (
	"e-commerce/adapters/controller/address/dto"
	"e-commerce/adapters/controller/address/mapper"
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/security/auth"
	"e-commerce/core/apperrors"
	useCasesAddress "e-commerce/core/use_cases/address"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AddressController struct {
	lookupAddressByCep *useCasesAddress.LookupAddressByCepUseCase
	readMyAddresses    *useCasesAddress.ReadMyAddressesUseCase
	createAddress      *useCasesAddress.CreateAddressUseCase
}

func NewAddressController(
	lookupAddressByCep *useCasesAddress.LookupAddressByCepUseCase,
	readMyAddresses *useCasesAddress.ReadMyAddressesUseCase,
	createAddress *useCasesAddress.CreateAddressUseCase,
) *AddressController {
	return &AddressController{
		lookupAddressByCep: lookupAddressByCep,
		readMyAddresses:    readMyAddresses,
		createAddress:      createAddress,
	}
}

func (ctrl *AddressController) getClaims(c *gin.Context) (*auth.Claims, error) {
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

// CreateAddress cadastra um novo endereco salvo para o usuario autenticado.
// @Summary      Cadastrar novo endereco salvo
// @Description  Valida o endereco via BrasilAPI/IBGE e cadastra uma nova instancia vinculada a conta do usuario autenticado.
// @Tags         Addresses
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Param        request  body      dto.CreateAddressRequestDTO  true  "Dados do novo endereco"
// @Success      201      {object}  helpers.SwaggerDataResponse[dto.SavedAddressResponseDTO]
// @Failure      400      {object}  helpers.SwaggerValidationErrorResponse
// @Failure      401      {object}  helpers.SwaggerMessageResponse
// @Failure      422      {object}  helpers.SwaggerValidationErrorResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /addresses [post]
func (ctrl *AddressController) CreateAddress(c *gin.Context) {
	claims, err := ctrl.getClaims(c)
	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	var dtoRequest dto.CreateAddressRequestDTO
	if err = c.ShouldBindJSON(&dtoRequest); err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	if listErrField := dtoRequest.Validator(); listErrField != nil {
		helpers.HelperErrResponse(listErrField, c, apperrors.ErrInvalidInput)
		return
	}

	result, err := ctrl.createAddress.UseCase(
		c.Request.Context(),
		claims.Id.String(),
		mapper.DtoCreateRequestToAddress(dtoRequest),
	)
	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(
		mapper.SavedAddressToDtoWithWarning(*result.Address, result.Warning),
		http.StatusCreated,
		"address created",
		c,
	)
}

// ReadMyAddresses lista todos os enderecos salvos do usuario autenticado.
// @Summary      Listar meus enderecos salvos
// @Description  Retorna as instancias completas de endereco ja cadastradas pelo usuario autenticado para reutilizacao no checkout.
// @Tags         Addresses
// @Produce      json
// @Security     CookieAuth
// @Success      200  {object}  helpers.SwaggerDataResponse[[]dto.SavedAddressResponseDTO]
// @Failure      401  {object}  helpers.SwaggerMessageResponse
// @Failure      500  {object}  helpers.SwaggerMessageResponse
// @Router       /addresses/me [get]
func (ctrl *AddressController) ReadMyAddresses(c *gin.Context) {
	claims, err := ctrl.getClaims(c)
	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	addresses, err := ctrl.readMyAddresses.UseCase(claims.Id.String())
	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(mapper.SavedAddressesToDto(addresses), http.StatusOK, "list addresses", c)
}

// LookupByCep consulta um CEP brasileiro via BrasilAPI para preenchimento automatico de endereco.
// @Summary      Consultar endereco por CEP
// @Description  Valida o formato do CEP (8 digitos, aceitando mascara 00000-000) e retorna rua, bairro, cidade e estado via BrasilAPI.
// @Tags         Addresses
// @Produce      json
// @Security     CookieAuth
// @Param        cep  path      string  true  "CEP (00000000 ou 00000-000)"
// @Success      200  {object}  helpers.SwaggerDataResponse[dto.AddressLookupResponseDTO]
// @Failure      400  {object}  helpers.SwaggerValidationErrorResponse
// @Failure      401  {object}  helpers.SwaggerMessageResponse
// @Failure      404  {object}  helpers.SwaggerMessageResponse
// @Failure      503  {object}  helpers.SwaggerMessageResponse
// @Router       /addresses/lookup/{cep} [get]
func (ctrl *AddressController) LookupByCep(c *gin.Context) {
	result, err := ctrl.lookupAddressByCep.UseCase(c.Request.Context(), c.Param("cep"))
	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(mapper.CepLookupResultToDto(result), http.StatusOK, "address found", c)
}
