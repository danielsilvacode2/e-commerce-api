package middleware

import (
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/security/auth"
	"e-commerce/core/apperrors"
	"net/http"

	"github.com/gin-gonic/gin"
)

func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func RateLimiterMiddleware(limiter *IPRateLimiter) gin.HandlerFunc {

	return func(c *gin.Context) {

		clientIp := c.ClientIP()

		clientLimiter := limiter.GetLimiter(clientIp)

		if !clientLimiter.Allow() {
			helpers.HelperErrResponse(nil, c, apperrors.ErrTooManyRequests)
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
