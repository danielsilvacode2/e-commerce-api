package middleware

import (
	"e-commerce/core/apperrors"
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/security/auth"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(role string) gin.HandlerFunc {

	return func(c *gin.Context) {

		claims, err := auth.ValidateToken(c)

		if err != nil {
			helpers.HelperErrResponse(nil, c, err)
			c.Abort()
			return
		}

		if claims.Role != role && claims.Role != "admin" {

			helpers.HelperErrResponse(nil, c, apperrors.ErrForbidden)
			c.Abort()
			return
		}

		c.Set("claims", claims)

		c.Next()
	}

}
