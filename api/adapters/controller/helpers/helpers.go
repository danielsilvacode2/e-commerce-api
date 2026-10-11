package helpers

import (
	"e-commerce/adapters/controller/apperrors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HelperResponseDto struct {
	Status   int    `json:"status"`
	Mensagem string `json:"mensagem"`
}

func HelperResponse(body interface{}, status int, msg string, c *gin.Context) {

	dtoResponse := HelperResponseDto{
		Status:   status,
		Mensagem: msg,
	}

	if body != nil {

		c.JSON(status, gin.H{
			"Response":       dtoResponse,
			"Response-Body:": body,
		})

		return
	}

	c.JSON(status, gin.H{
		"Response": dtoResponse,
	})

}

func HelperErrResponse(body interface{},  c *gin.Context, err error) {

	switch err {
		
	case apperrors.RegisterNotFound:
		HelperResponse(body, http.StatusNotFound, err.Error(), c)
		return

	default:
		HelperResponse(body, http.StatusInternalServerError, "Error internal", c)	
		return
	
	}

}
