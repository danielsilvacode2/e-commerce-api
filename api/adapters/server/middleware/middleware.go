package middleware

import (
	"bytes"
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/security/auth"
	"e-commerce/config"
	"e-commerce/core/apperrors"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

const maxRateLimitBodyBytes = 1 << 20 // 1 MB

func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("Referrer-Policy", "no-referrer")
		c.Writer.Header().Set("X-XSS-Protection", "0")
		if config.CookieSecure {
			c.Writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}

// MaxBodySizeMiddleware limita globalmente o tamanho do corpo HTTP para prevenir DoS por consumo de memoria.
func MaxBodySizeMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}

// CSRFOriginMiddleware valida Fetch Metadata (Sec-Fetch-Site) e Origin/Referer em requisicoes mutaveis
// para mitigar CSRF em autenticacao baseada em cookie. Ignora webhooks externos autenticados via HMAC.
func CSRFOriginMiddleware() gin.HandlerFunc {
	allowedOrigins := make(map[string]bool)
	if len(config.AllowedOrigins) == 0 {
		allowedOrigins["http://localhost:3000"] = true
	} else {
		for _, o := range config.AllowedOrigins {
			allowedOrigins[strings.TrimRight(o, "/")] = true
		}
	}

	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}

		if c.Request.URL.Path == "/api/v1/payments/webhook" {
			c.Next()
			return
		}

		if secFetchSite := strings.ToLower(strings.TrimSpace(c.GetHeader("Sec-Fetch-Site"))); secFetchSite == "cross-site" {
			helpers.HelperErrResponse(nil, c, apperrors.ErrForbidden)
			c.Abort()
			return
		}

		if origin := strings.TrimRight(strings.TrimSpace(c.GetHeader("Origin")), "/"); origin != "" {
			if !allowedOrigins[origin] {
				helpers.HelperErrResponse(nil, c, apperrors.ErrForbidden)
				c.Abort()
				return
			}
			c.Next()
			return
		}

		if referer := strings.TrimSpace(c.GetHeader("Referer")); referer != "" {
			parsed, err := url.Parse(referer)
			if err != nil || parsed.Scheme == "" || parsed.Host == "" {
				helpers.HelperErrResponse(nil, c, apperrors.ErrForbidden)
				c.Abort()
				return
			}
			refOrigin := parsed.Scheme + "://" + parsed.Host
			if !allowedOrigins[refOrigin] {
				helpers.HelperErrResponse(nil, c, apperrors.ErrForbidden)
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

func CorsMiddleware() gin.HandlerFunc {

	allowedOrigins := make(map[string]bool)
	if len(config.AllowedOrigins) == 0 {
		allowedOrigins["http://localhost:3000"] = true
	} else {
		for _, o := range config.AllowedOrigins {
			allowedOrigins[o] = true
		}
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

func readLimitedBody(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRateLimitBodyBytes)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, false
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	return body, true
}

func EmailRateLimiterMiddleware(limiter *EmailRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, ok := readLimitedBody(c)
		if !ok {
			helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
			c.Abort()
			return
		}

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
		body, ok := readLimitedBody(c)
		if !ok {
			helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
			c.Abort()
			return
		}

		var payload struct {
			Email string `json:"email"`
		}
		if err := json.Unmarshal(body, &payload); err != nil || payload.Email == "" {
			c.Next()
			return
		}

		// Consumo atomico antes de c.Next() evita bypass por race condition em rajadas concorrentes.
		if !limiter.GetLimiter(payload.Email).Allow() {
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
