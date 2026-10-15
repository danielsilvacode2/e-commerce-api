package controller

import (
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/controller/product/dto"
	"e-commerce/adapters/controller/product/mapper"
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/product"
	use_cases "e-commerce/core/use_cases/product"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ProductController struct {
	repository entities.ProductRepository
}

func NewProductController(repository entities.ProductRepository) *ProductController {
	return &ProductController{
		repository: repository,
	}
}

func (productCtrl *ProductController) CreateProduct(c *gin.Context) {

	var dtoRequest dto.ProcuctRequestDTO

	if err := c.ShouldBindJSON(&dtoRequest); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	if listErrors := dtoRequest.Validator(); listErrors != nil {
		helpers.HelperErrResponse(listErrors, c, apperrors.ErrInvalidInput)
		return
	}

	potentialProduct := mapper.DtoToProduct(dtoRequest)
	product, err := use_cases.CreateProduct(potentialProduct, productCtrl.repository)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	dtoResponse := mapper.ProductToDto(*product)

	helpers.HelperResponse(dtoResponse, http.StatusOK, "create product", c)

}

func (productCtrl *ProductController) ReadProductById(c *gin.Context) {

	productID := c.Param("id")

	product, err := use_cases.ReadProductById(productID, productCtrl.repository)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	dto := mapper.ProductToDto(*product)

	helpers.HelperResponse(dto, http.StatusOK, "found product", c)

}

func (productCtrl *ProductController) ReadAllProducts(c *gin.Context) {

	listProducts, err := use_cases.ReadAllProducts(productCtrl.repository)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	var listProductsDTO []dto.ProcuctResponseDTO

	for _, product := range listProducts {

		dto := mapper.ProductToDto(product)

		listProductsDTO = append(listProductsDTO, dto)
	}

	helpers.HelperResponse(listProductsDTO, http.StatusOK, "list products", c)
}

func (productCtrl *ProductController) DeleteProductById(c *gin.Context) {

	Id := c.Param("id")

	if err := use_cases.DeleteProductById(Id, productCtrl.repository); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, http.StatusOK, "product successfully deleted", c)

}
