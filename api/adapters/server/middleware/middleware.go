package middleware

import (
	"bytes"
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/security/auth"
	"e-commerce/config"
	"e-commerce/core/apperrors"
	"encoding/json"
	"io"
	"log"
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

func buildAllowedOrigins() map[string]bool {
	allowedOrigins := make(map[string]bool)
	if len(config.AllowedOrigins) == 0 {
		allowedOrigins["http://localhost:3000"] = true
		allowedOrigins["https://nero-pi-teal.vercel.app"] = true
		return allowedOrigins
	}

	for _, o := range config.AllowedOrigins {
		if trimmed := strings.TrimRight(strings.TrimSpace(o), "/"); trimmed != "" {
			allowedOrigins[trimmed] = true
		}
	}
	return allowedOrigins
}

// CSRFOriginMiddleware valida Fetch Metadata (Sec-Fetch-Site) e Origin/Referer em requisicoes mutaveis
// para mitigar CSRF em autenticacao baseada em cookie. Ignora webhooks externos autenticados via HMAC.
func CSRFOriginMiddleware() gin.HandlerFunc {
	allowedOrigins := buildAllowedOrigins()

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

		origin := strings.TrimRight(strings.TrimSpace(c.GetHeader("Origin")), "/")
		secFetchSite := strings.ToLower(strings.TrimSpace(c.GetHeader("Sec-Fetch-Site")))

		if secFetchSite == "cross-site" {
			if origin == "" || !allowedOrigins[origin] {
				helpers.HelperErrResponse(nil, c, apperrors.ErrForbidden)
				c.Abort()
				return
			}
			c.Next()
			return
		}

		if origin != "" {
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

	allowedOrigins := buildAllowedOrigins()

	return func(c *gin.Context) {

		origin := strings.TrimRight(strings.TrimSpace(c.GetHeader("Origin")), "/")

		if allowedOrigins[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Vary", "Origin")
		}

		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, Idempotency-Key")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

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

		allowed, err := limiter.Allow(c.Request.Context(), clientIp)
		if err != nil {
			log.Printf("[WARN] falha ao verificar rate limit por IP no Redis: %v", err)
			c.Next()
			return
		}

		if !allowed {
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

		allowed, err := limiter.Allow(c.Request.Context(), payload.Email)
		if err != nil {
			log.Printf("[WARN] falha ao verificar rate limit por email no Redis: %v", err)
			c.Next()
			return
		}

		if !allowed {
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

		// Consumo atomico no Redis antes de c.Next() evita bypass por race condition em rajadas concorrentes.
		allowed, err := limiter.Allow(c.Request.Context(), payload.Email)
		if err != nil {
			log.Printf("[WARN] falha ao verificar rate limit de login no Redis: %v", err)
			c.Next()
			return
		}

		if !allowed {
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
