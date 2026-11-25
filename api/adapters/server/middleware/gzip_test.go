package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGzipMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(GzipMiddleware())
	r.GET("/products", func(c *gin.Context) {
		c.String(http.StatusOK, strings.Repeat("nero-product-", 50))
	})
	r.DELETE("/products/1", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	t.Run("Compresses response when Accept-Encoding includes gzip", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/products", nil)
		req.Header.Set("Accept-Encoding", "gzip, deflate")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if got := w.Header().Get("Content-Encoding"); got != "gzip" {
			t.Fatalf("expected Content-Encoding gzip, got %q", got)
		}

		gr, err := gzip.NewReader(w.Body)
		if err != nil {
			t.Fatalf("failed to create gzip reader: %v", err)
		}
		defer gr.Close()

		decoded, err := io.ReadAll(gr)
		if err != nil {
			t.Fatalf("failed to read gzip body: %v", err)
		}
		expected := strings.Repeat("nero-product-", 50)
		if string(decoded) != expected {
			t.Fatalf("decoded body mismatch: got len %d, want %d", len(decoded), len(expected))
		}
	})

	t.Run("Skips compression when client does not accept gzip", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/products", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if got := w.Header().Get("Content-Encoding"); got != "" {
			t.Fatalf("expected empty Content-Encoding, got %q", got)
		}
	})

	t.Run("Does not set gzip header on 204 No Content", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/products/1", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", w.Code)
		}
		if got := w.Header().Get("Content-Encoding"); got != "" {
			t.Fatalf("expected empty Content-Encoding for 204, got %q", got)
		}
	})
}
