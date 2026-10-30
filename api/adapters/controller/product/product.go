package controller

import (
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/controller/product/dto"
	"e-commerce/adapters/controller/product/mapper"
	"e-commerce/core/apperrors"
	use_cases "e-commerce/core/use_cases/product"
	"net/http"
	"strconv"

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
	readProductById *use_cases.ReadProductByIdUseCase) *ProductController {

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
	product, err := productCtrl.createProduct.UseCase(potentialProduct)

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

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	paginete, err := productCtrl.readAllProducts.UseCase(page, limit)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	var listProductsDTO []dto.ProcuctResponseDTO

	if paginete == nil {
		helpers.HelperResponse(listProductsDTO, http.StatusOK, "list is empty", c)
		return
	}

	for _, product := range paginete.Data {

		dto := mapper.ProductToDto(product)

		listProductsDTO = append(listProductsDTO, dto)
	}

	var pagineteDTO dto.PaginatedProducts

	pagineteDTO.Products = listProductsDTO
	pagineteDTO.Limit = paginete.Limit
	pagineteDTO.Page = paginete.Page
	pagineteDTO.TotalPages = paginete.TotalPages
	pagineteDTO.TotalRows = paginete.TotalRows

	helpers.HelperResponse(pagineteDTO, http.StatusOK, "list products", c)
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
