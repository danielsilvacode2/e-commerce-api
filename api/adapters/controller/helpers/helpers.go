package helpers

import (
	fielderrors "e-commerce/adapters/controller/fieldErrors"
	"e-commerce/core/apperrors"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HelperResponseDto struct {
	Status   int    `json:"status"`
	Mensagem string `json:"mensagem"`
}

// SwaggerMessageResponse representa a resposta padrao da API sem payload em Body.
type SwaggerMessageResponse struct {
	Response HelperResponseDto `json:"Response"`
}

// SwaggerDataResponse representa a resposta padrao da API contendo payload em Body.
type SwaggerDataResponse[T any] struct {
	Response HelperResponseDto `json:"Response"`
	Body     T                 `json:"Body:"`
}

// SwaggerValidationErrorResponse representa a resposta de erro com lista de campos invalidos.
type SwaggerValidationErrorResponse struct {
	Response HelperResponseDto                 `json:"Response"`
	Body     []fielderrors.FieldValidateErrors `json:"Body:"`
}

func HelperResponse(body interface{}, status int, msg string, c *gin.Context) {

	dtoResponse := HelperResponseDto{
		Status:   status,
		Mensagem: msg,
	}

	if body != nil {

		c.JSON(status, gin.H{
			"Response": dtoResponse,
			"Body:":    body,
		})

		return
	}

	c.JSON(status, gin.H{
		"Response": dtoResponse,
	})

}

func HelperErrResponse(body interface{}, c *gin.Context, err error) {

	switch {

	case errors.Is(err, apperrors.ErrNotFound):
		HelperResponse(body, http.StatusNotFound, apperrors.ErrNotFound.Error(), c)
		return
	case errors.Is(err, apperrors.ErrInvalidInput):
		HelperResponse(body, http.StatusBadRequest, apperrors.ErrInvalidInput.Error(), c)
		return
	case errors.Is(err, apperrors.ErrConflict):
		HelperResponse(body, http.StatusConflict, apperrors.ErrConflict.Error(), c)
		return
	case errors.Is(err, apperrors.ErrForbidden):
		HelperResponse(body, http.StatusForbidden, apperrors.ErrForbidden.Error(), c)
		return
	case errors.Is(err, apperrors.ErrUnauthorized):
		HelperResponse(body, http.StatusUnauthorized, apperrors.ErrUnauthorized.Error(), c)
		return
	case errors.Is(err, apperrors.ErrTooManyRequests):
		HelperResponse(body, http.StatusTooManyRequests, apperrors.ErrTooManyRequests.Error(), c)
		return
	case errors.Is(err, apperrors.ErrUnprocessableEntity):
		HelperResponse(body, http.StatusUnprocessableEntity, apperrors.ErrUnprocessableEntity.Error(), c)
		return
	case errors.Is(err, apperrors.ErrPayloadTooLarge):
		HelperResponse(body, http.StatusRequestEntityTooLarge, apperrors.ErrPayloadTooLarge.Error(), c)
		return
	case errors.Is(err, apperrors.ErrServiceUnavailable):
		HelperResponse(body, http.StatusServiceUnavailable, apperrors.ErrServiceUnavailable.Error(), c)
		return
	case errors.Is(err, apperrors.ErrImageUploadUnavailable):
		HelperResponse(body, http.StatusServiceUnavailable, apperrors.ErrImageUploadUnavailable.Error(), c)
		return
	default:
		log.Printf("[ERROR] internal server error [%s %s]: %v", c.Request.Method, c.FullPath(), err)
		HelperResponse(body, http.StatusInternalServerError, "internal server error", c)
		return

	}

}
