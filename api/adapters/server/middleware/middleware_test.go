package middleware

import (
	"bytes"
	"e-commerce/config"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCorsMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origAllowed := config.AllowedOrigins
	t.Cleanup(func() {
		config.AllowedOrigins = origAllowed
	})

	config.AllowedOrigins = nil

	r := gin.New()
	r.Use(CorsMiddleware())
	r.GET("/api/v1/products", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	t.Run("Allows Vercel origin on GET and sets credentials headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
		req.Header.Set("Origin", "https://nero-pi-teal.vercel.app")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://nero-pi-teal.vercel.app" {
			t.Fatalf("expected Access-Control-Allow-Origin https://nero-pi-teal.vercel.app, got %q", got)
		}
		if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
			t.Fatalf("expected Access-Control-Allow-Credentials true, got %q", got)
		}
	})

	t.Run("Allows preflight OPTIONS for Vercel origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/login", nil)
		req.Header.Set("Origin", "https://nero-pi-teal.vercel.app")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://nero-pi-teal.vercel.app" {
			t.Fatalf("expected Access-Control-Allow-Origin https://nero-pi-teal.vercel.app, got %q", got)
		}
	})

	t.Run("Does not set Access-Control-Allow-Origin for untrusted origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
		req.Header.Set("Origin", "https://evil.example.com")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("expected empty Access-Control-Allow-Origin, got %q", got)
		}
	})
}

func TestCSRFOriginMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origAllowed := config.AllowedOrigins
	t.Cleanup(func() {
		config.AllowedOrigins = origAllowed
	})

	config.AllowedOrigins = nil

	r := gin.New()
	r.Use(CSRFOriginMiddleware())
	r.POST("/api/v1/login", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	r.POST("/api/v1/payments/webhook", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	r.GET("/api/v1/products", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	cases := []struct {
		name         string
		method       string
		path         string
		origin       string
		referer      string
		secFetchSite string
		wantCode     int
	}{
		{
			name:         "Allows cross-site POST from allowed Vercel origin",
			method:       http.MethodPost,
			path:         "/api/v1/login",
			origin:       "https://nero-pi-teal.vercel.app",
			secFetchSite: "cross-site",
			wantCode:     http.StatusOK,
		},
		{
			name:         "Blocks cross-site POST from untrusted origin",
			method:       http.MethodPost,
			path:         "/api/v1/login",
			origin:       "https://evil.example.com",
			secFetchSite: "cross-site",
			wantCode:     http.StatusForbidden,
		},
		{
			name:         "Blocks cross-site POST with null origin",
			method:       http.MethodPost,
			path:         "/api/v1/login",
			origin:       "null",
			secFetchSite: "cross-site",
			wantCode:     http.StatusForbidden,
		},
		{
			name:         "Blocks cross-site POST without origin even with allowed referer",
			method:       http.MethodPost,
			path:         "/api/v1/login",
			referer:      "https://nero-pi-teal.vercel.app/login",
			secFetchSite: "cross-site",
			wantCode:     http.StatusForbidden,
		},
		{
			name:         "Allows same-site POST from localhost",
			method:       http.MethodPost,
			path:         "/api/v1/login",
			origin:       "http://localhost:3000",
			secFetchSite: "same-site",
			wantCode:     http.StatusOK,
		},
		{
			name:         "Allows cross-site GET",
			method:       http.MethodGet,
			path:         "/api/v1/products",
			origin:       "https://evil.example.com",
			secFetchSite: "cross-site",
			wantCode:     http.StatusOK,
		},
		{
			name:         "Allows Mercado Pago webhook without origin",
			method:       http.MethodPost,
			path:         "/api/v1/payments/webhook",
			secFetchSite: "cross-site",
			wantCode:     http.StatusOK,
		},
		{
			name:     "Blocks POST with untrusted referer when origin is absent",
			method:   http.MethodPost,
			path:     "/api/v1/login",
			referer:  "https://evil.example.com/attack",
			wantCode: http.StatusForbidden,
		},
		{
			name:     "Allows POST with allowed referer when origin and Sec-Fetch-Site are absent",
			method:   http.MethodPost,
			path:     "/api/v1/login",
			referer:  "https://nero-pi-teal.vercel.app/login",
			wantCode: http.StatusOK,
		},
		{
			name:     "Allows non-browser POST without origin, referer, or Sec-Fetch-Site",
			method:   http.MethodPost,
			path:     "/api/v1/login",
			wantCode: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.referer != "" {
				req.Header.Set("Referer", tc.referer)
			}
			if tc.secFetchSite != "" {
				req.Header.Set("Sec-Fetch-Site", tc.secFetchSite)
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.wantCode {
				t.Fatalf("expected %d, got %d (body=%s)", tc.wantCode, w.Code, w.Body.String())
			}
		})
	}
}

func TestMaxBodySizeMiddlewareWithRouteLimits(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(MaxBodySizeMiddlewareWithRouteLimits(16, map[string]int64{
		"POST /api/v1/products/image": 64,
	}))

	readAllHandler := func(c *gin.Context) {
		if _, err := io.ReadAll(c.Request.Body); err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusOK)
	}
	r.POST("/api/v1/products", readAllHandler)
	r.POST("/api/v1/products/image", readAllHandler)

	// Rota padrao rejeita corpo maior que 16 bytes
	reqDefault := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(make([]byte, 32)))
	wDefault := httptest.NewRecorder()
	r.ServeHTTP(wDefault, reqDefault)
	if wDefault.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("rota padrao: status = %d, esperado 413", wDefault.Code)
	}

	// Rota de upload aceita 32 bytes dentro do limite de 64 bytes
	reqUpload := httptest.NewRequest(http.MethodPost, "/api/v1/products/image", bytes.NewReader(make([]byte, 32)))
	wUpload := httptest.NewRecorder()
	r.ServeHTTP(wUpload, reqUpload)
	if wUpload.Code != http.StatusOK {
		t.Fatalf("rota upload dentro do limite: status = %d, esperado 200", wUpload.Code)
	}

	// Rota de upload rejeita corpo acima de 64 bytes
	reqUploadOver := httptest.NewRequest(http.MethodPost, "/api/v1/products/image", bytes.NewReader(make([]byte, 80)))
	wUploadOver := httptest.NewRecorder()
	r.ServeHTTP(wUploadOver, reqUploadOver)
	if wUploadOver.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("rota upload acima do limite: status = %d, esperado 413", wUploadOver.Code)
	}
}
