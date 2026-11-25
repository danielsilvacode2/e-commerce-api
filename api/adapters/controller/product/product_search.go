package controller

import (
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/controller/product/dto"
	"e-commerce/adapters/controller/product/mapper"
	"e-commerce/core/apperrors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// SearchProducts trata GET /products/search?q=<termo>&page=&limit=
// @Summary      Pesquisar produtos
// @Description  Busca produtos por termo textual com suporte a paginacao.
// @Tags         Products
// @Produce      json
// @Param        q      query     string  true   "Termo de busca"
// @Param        page   query     int     false  "Numero da pagina"  default(1)
// @Param        limit  query     int     false  "Itens por pagina"  default(10)
// @Success      200    {object}  helpers.SwaggerDataResponse[dto.PaginatedProducts]
// @Failure      400    {object}  helpers.SwaggerValidationErrorResponse
// @Failure      500    {object}  helpers.SwaggerMessageResponse
// @Router       /products/search [get]
func (productCtrl *ProductController) SearchProducts(c *gin.Context) {

	dtoRequest := dto.ProductSearchRequestDTO{Query: c.Query("q")}

	if listFieldErr := dtoRequest.Validator(); listFieldErr != nil {
		helpers.HelperErrResponse(listFieldErr, c, apperrors.ErrInvalidInput)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	paginete, err := productCtrl.searchProducts.UseCase(c.Request.Context(), dtoRequest.Query, page, limit)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	listProductsDTO := []dto.ProcuctResponseDTO{}

	for _, product := range paginete.Data {
		listProductsDTO = append(listProductsDTO, mapper.ProductToDto(product))
	}

	var pagineteDTO dto.PaginatedProducts

	pagineteDTO.Products = listProductsDTO
	pagineteDTO.Limit = paginete.Limit
	pagineteDTO.Page = paginete.Page
	pagineteDTO.TotalPages = paginete.TotalPages
	pagineteDTO.TotalRows = paginete.TotalRows

	helpers.HelperResponse(pagineteDTO, http.StatusOK, "search products", c)
}
