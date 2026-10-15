package middleware

import (
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/security/auth"
	"e-commerce/core/apperrors"

	"github.com/gin-gonic/gin"
)

func RateLimiterMiddleware(limiter *IPRateLimiter) gin.HandlerFunc {

	return func(c *gin.Context) {

		clientIp := c.ClientIP()

		clientLimiter := limiter.GetLimiter(clientIp)

		if !clientLimiter.Allow(){
			helpers.HelperErrResponse(nil,c , apperrors.ErrTooManyRequests)
			c.Abort()
			return 
		}


		c.Next()
	}
}

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
