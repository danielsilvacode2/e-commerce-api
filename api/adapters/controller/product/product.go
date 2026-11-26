package controller

import (
	fielderrors "e-commerce/adapters/controller/fieldErrors"
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/controller/product/dto"
	"e-commerce/adapters/controller/product/mapper"
	redisAdapter "e-commerce/adapters/redis"
	"e-commerce/core/apperrors"
	use_cases "e-commerce/core/use_cases/product"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	productListCacheTTL   = 30 * time.Second
	productDetailCacheTTL = 60 * time.Second
)

type ProductController struct {
	createProduct      *use_cases.CreateProductUseCase
	readAllProducts    *use_cases.ReadAllProductsUseCase
	deleteProductById  *use_cases.DeleteProductByIdUseCase
	updateProduct      *use_cases.UpdateProductUseCase
	readProductById    *use_cases.ReadProductByIdUseCase
	searchProducts     *use_cases.SearchProductsUseCase
	uploadProductImage *use_cases.UploadProductImageUseCase
}

func NewProductController(
	createProduct *use_cases.CreateProductUseCase,
	readAllProducts *use_cases.ReadAllProductsUseCase,
	deleteProductById *use_cases.DeleteProductByIdUseCase,
	updateProduct *use_cases.UpdateProductUseCase,
	readProductById *use_cases.ReadProductByIdUseCase,
	searchProducts *use_cases.SearchProductsUseCase,
	uploadProductImage *use_cases.UploadProductImageUseCase) *ProductController {

	return &ProductController{
		createProduct:      createProduct,
		readAllProducts:    readAllProducts,
		deleteProductById:  deleteProductById,
		updateProduct:      updateProduct,
		readProductById:    readProductById,
		searchProducts:     searchProducts,
		uploadProductImage: uploadProductImage,
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
	redisAdapter.InvalidateProductCache(c.Request.Context())

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
	c.Header("Cache-Control", "public, max-age=30, stale-while-revalidate=60")

	var cached dto.ProcuctResponseDTO
	if redisAdapter.GetCachedProductByID(c.Request.Context(), productID, &cached) {
		helpers.HelperResponse(cached, http.StatusOK, "found product", c)
		return
	}

	product, err := productCtrl.readProductById.UseCase(productID)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	dtoResp := mapper.ProductToDto(*product)
	redisAdapter.SetCachedProductByID(c.Request.Context(), productID, dtoResp, productDetailCacheTTL)

	helpers.HelperResponse(dtoResp, http.StatusOK, "found product", c)

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
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	c.Header("Cache-Control", "public, max-age=30, stale-while-revalidate=60")

	var cachedDTO dto.PaginatedProducts
	if redisAdapter.GetCachedProductList(c.Request.Context(), page, limit, &cachedDTO) {
		helpers.HelperResponse(cachedDTO, http.StatusOK, "list products", c)
		return
	}

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

		dtoItem := mapper.ProductToDto(product)

		listProductsDTO = append(listProductsDTO, dtoItem)
	}

	var pagineteDTO dto.PaginatedProducts

	pagineteDTO.Products = listProductsDTO
	pagineteDTO.Limit = paginete.Limit
	pagineteDTO.Page = paginete.Page
	pagineteDTO.TotalPages = paginete.TotalPages
	pagineteDTO.TotalRows = paginete.TotalRows

	redisAdapter.SetCachedProductList(c.Request.Context(), page, limit, pagineteDTO, productListCacheTTL)

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

	redisAdapter.InvalidateProductCache(c.Request.Context())
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

	redisAdapter.InvalidateProductCache(c.Request.Context())
	helpers.HelperResponse(nil, http.StatusNoContent, "product successfully update", c)

}

// UploadProductImage envia uma imagem de produto ao servico de hospedagem (somente admin).
// @Summary      Enviar imagem de produto
// @Description  Valida e realiza upload de uma imagem (JPG, PNG, WEBP ou GIF de ate 5 MB). Requer perfil admin.
// @Tags         Products
// @Accept       mpfd
// @Produce      json
// @Security     CookieAuth
// @Param        image  formData  file  true  "Arquivo de imagem (JPG, PNG, WEBP ou GIF, max 5 MB)"
// @Success      200    {object}  helpers.SwaggerDataResponse[dto.ProductImageUploadResponseDTO]
// @Failure      400    {object}  helpers.SwaggerValidationErrorResponse
// @Failure      401    {object}  helpers.SwaggerMessageResponse
// @Failure      403    {object}  helpers.SwaggerMessageResponse
// @Failure      413    {object}  helpers.SwaggerMessageResponse
// @Failure      503    {object}  helpers.SwaggerMessageResponse
// @Failure      500    {object}  helpers.SwaggerMessageResponse
// @Router       /products/image [post]
func (productCtrl *ProductController) UploadProductImage(c *gin.Context) {
	fileHeader, err := c.FormFile("image")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) || strings.Contains(err.Error(), "http: request body too large") {
			helpers.HelperErrResponse(nil, c, apperrors.ErrPayloadTooLarge)
			return
		}
		helpers.HelperErrResponse([]fielderrors.FieldValidateErrors{
			{
				Field:   "image",
				Message: "the image file is required",
			},
		}, c, apperrors.ErrInvalidInput)
		return
	}

	if fileHeader.Size > use_cases.MaxProductImageSize {
		helpers.HelperErrResponse(nil, c, apperrors.ErrPayloadTooLarge)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, use_cases.MaxProductImageSize+1))
	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	uploadedURL, err := productCtrl.uploadProductImage.UseCase(c.Request.Context(), fileHeader.Filename, data)
	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(dto.ProductImageUploadResponseDTO{URL: uploadedURL}, http.StatusOK, "image uploaded", c)
}
