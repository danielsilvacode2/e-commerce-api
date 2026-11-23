package middleware

import (
	"bytes"
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/security/auth"
	"e-commerce/core/apperrors"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)


func CorsMiddleware() gin.HandlerFunc {

	allowedOrigins := map[string]bool{
		"http://localhost:3000": true,
	}

	return func(c *gin.Context) {

		origin := c.GetHeader("Origin")

		if allowedOrigins[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Vary", "Origin")
		}

		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, Idempotency-Key")
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

func EmailRateLimiterMiddleware(limiter *EmailRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Next()
			return
		}

		// Controllers still need to bind the original JSON body after this middleware.
		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		var payload struct {
			Email string `json:"email"`
		}
		if err := json.Unmarshal(body, &payload); err != nil || payload.Email == "" {
			c.Next()
			return
		}

		if !limiter.GetLimiter(payload.Email).Allow() {
			helpers.HelperErrResponse(nil, c, apperrors.ErrTooManyRequests)
			c.Abort()
			return
		}

		c.Next()
	}
}

func EmailRateLimiterLoginMiddleware(limiter *EmailRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Next()
			return
		}

		// Controllers still need to bind the original JSON body after this middleware.
		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		var payload struct {
			Email string `json:"email"`
		}
		if err := json.Unmarshal(body, &payload); err != nil || payload.Email == "" {
			c.Next()
			return
		}

		l := limiter.GetLimiter(payload.Email)

		if l.Tokens() < 1 {
			helpers.HelperErrResponse(nil, c, apperrors.ErrTooManyRequests)
			c.Abort()
			return
		}

		c.Next()

		if c.Writer.Status() >= 400 {
			l.Allow()
		}
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
