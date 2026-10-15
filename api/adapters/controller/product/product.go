package controller

import (
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/controller/product/dto"
	"e-commerce/adapters/controller/product/mapper"
	"e-commerce/core/apperrors"
	use_cases "e-commerce/core/use_cases/product"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ProductController struct {
	createProduct     *use_cases.CreateProductUseCase
	readAllProducts   *use_cases.ReadAllProductsUseCase
	deleteProductById *use_cases.DeleteProductByIdUseCase
	updateProduct     *use_cases.UpdateProductUseCase
	readProductById   *use_cases.ReadProductByIdUseCase
}

func NewProductController(
	createProduct *use_cases.CreateProductUseCase, 
	readAllProducts *use_cases.ReadAllProductsUseCase, 
	deleteProductById *use_cases.DeleteProductByIdUseCase, 
	updateProduct *use_cases.UpdateProductUseCase, 
	readProductById *use_cases.ReadProductByIdUseCase, ) *ProductController {

	return &ProductController{
		createProduct:     createProduct,
		readAllProducts:   readAllProducts,
		deleteProductById: deleteProductById,
		updateProduct:     updateProduct,
		readProductById:   readProductById,
	}
}

func (productCtrl *ProductController) CreateProduct(c *gin.Context) {

	var dtoRequest dto.ProcuctCreateRequestDTO

	if err := c.ShouldBindJSON(&dtoRequest); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	if listErrors := dtoRequest.Validator(); listErrors != nil {
		helpers.HelperErrResponse(listErrors, c, apperrors.ErrInvalidInput)
		return
	}

	potentialProduct := mapper.DtoCreateToProduct(dtoRequest)
	product, err :=productCtrl.createProduct.UseCase(potentialProduct)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	dtoResponse := mapper.ProductToDto(*product)

	helpers.HelperResponse(dtoResponse, http.StatusCreated, "create product", c)

}

func (productCtrl *ProductController) ReadProductById(c *gin.Context) {

	productID := c.Param("id")

	product, err := productCtrl.readProductById.UseCase(productID)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	dto := mapper.ProductToDto(*product)

	helpers.HelperResponse(dto, http.StatusOK, "found product", c)

}

func (productCtrl *ProductController) ReadAllProducts(c *gin.Context) {

	listProducts, err :=productCtrl.readAllProducts.UseCase()

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

	if err := productCtrl.deleteProductById.UseCase(Id); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, http.StatusNoContent, "product successfully deleted", c)

}

func (productCtrl *ProductController) UpdateProduct(c *gin.Context) {

	var dtoRequest dto.ProcuctUpdateRequestDTO

	if err := c.ShouldBindJSON(&dtoRequest); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	if listFieldErr := dtoRequest.Validator(); listFieldErr != nil {
		helpers.HelperErrResponse(listFieldErr, c, apperrors.ErrInvalidInput)
		return
	}

	product := mapper.DtoUpdateToProduct(dtoRequest)

	id := c.Param("id")

	if err := productCtrl.updateProduct.UseCase(product, id); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, http.StatusNoContent, "product successfully update", c)

}
