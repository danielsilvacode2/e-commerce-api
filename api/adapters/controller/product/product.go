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
	searchProducts    *use_cases.SearchProductsUseCase
}

func NewProductController(
	createProduct *use_cases.CreateProductUseCase,
	readAllProducts *use_cases.ReadAllProductsUseCase,
	deleteProductById *use_cases.DeleteProductByIdUseCase,
	updateProduct *use_cases.UpdateProductUseCase,
	readProductById *use_cases.ReadProductByIdUseCase,
	searchProducts *use_cases.SearchProductsUseCase) *ProductController {

	return &ProductController{
		createProduct:     createProduct,
		readAllProducts:   readAllProducts,
		deleteProductById: deleteProductById,
		updateProduct:     updateProduct,
		readProductById:   readProductById,
		searchProducts:    searchProducts,
	}
}

// CreateProduct cria um novo produto (somente admin).
// @Summary      Criar produto
// @Description  Cadastra um novo produto no catalogo. Requer perfil admin.
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Param        request  body      dto.ProcuctCreateRequestDTO  true  "Dados do produto"
// @Success      201      {object}  helpers.SwaggerDataResponse[dto.ProcuctResponseDTO]
// @Failure      400      {object}  helpers.SwaggerValidationErrorResponse
// @Failure      401      {object}  helpers.SwaggerMessageResponse
// @Failure      403      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /products [post]
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

// ReadProductById retorna um produto pelo seu ID.
// @Summary      Buscar produto por ID
// @Description  Retorna os detalhes de um produto especifico pelo ID.
// @Tags         Products
// @Produce      json
// @Security     CookieAuth
// @Param        id   path      string  true  "ID do produto (UUID)"
// @Success      200  {object}  helpers.SwaggerDataResponse[dto.ProcuctResponseDTO]
// @Failure      401  {object}  helpers.SwaggerMessageResponse
// @Failure      404  {object}  helpers.SwaggerMessageResponse
// @Failure      500  {object}  helpers.SwaggerMessageResponse
// @Router       /products/{id} [get]
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

// ReadAllProducts lista os produtos de forma paginada.
// @Summary      Listar produtos
// @Description  Retorna uma lista paginada de produtos disponiveis no catalogo.
// @Tags         Products
// @Produce      json
// @Param        page   query     int  false  "Numero da pagina"      default(1)
// @Param        limit  query     int  false  "Itens por pagina"      default(10)
// @Success      200    {object}  helpers.SwaggerDataResponse[dto.PaginatedProducts]
// @Failure      500    {object}  helpers.SwaggerMessageResponse
// @Router       /products [get]
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

// DeleteProductById remove um produto pelo ID (somente admin).
// @Summary      Remover produto
// @Description  Remove um produto do catalogo pelo ID. Requer perfil admin.
// @Tags         Products
// @Produce      json
// @Security     CookieAuth
// @Param        id   path  string  true  "ID do produto (UUID)"
// @Success      204  "Produto removido com sucesso"
// @Failure      401  {object}  helpers.SwaggerMessageResponse
// @Failure      403  {object}  helpers.SwaggerMessageResponse
// @Failure      404  {object}  helpers.SwaggerMessageResponse
// @Failure      500  {object}  helpers.SwaggerMessageResponse
// @Router       /products/{id} [delete]
func (productCtrl *ProductController) DeleteProductById(c *gin.Context) {

	Id := c.Param("id")

	if err := productCtrl.deleteProductById.UseCase(Id); err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(nil, http.StatusNoContent, "product successfully deleted", c)

}

// UpdateProduct atualiza os dados de um produto (somente admin).
// @Summary      Atualizar produto
// @Description  Atualiza os dados de um produto existente. Requer perfil admin.
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Param        id       path      string                       true  "ID do produto (UUID)"
// @Param        request  body      dto.ProcuctUpdateRequestDTO  true  "Dados atualizados do produto"
// @Success      204      "Produto atualizado com sucesso"
// @Failure      400      {object}  helpers.SwaggerValidationErrorResponse
// @Failure      401      {object}  helpers.SwaggerMessageResponse
// @Failure      403      {object}  helpers.SwaggerMessageResponse
// @Failure      404      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /products/{id} [put]
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
