package controller

import (
	"bytes"
	"context"
	"e-commerce/adapters/server/middleware"
	"e-commerce/core/apperrors"
	use_cases "e-commerce/core/use_cases/product"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type stubImageUploader struct {
	url string
	err error
}

func (s *stubImageUploader) UploadImage(ctx context.Context, filename string, data []byte) (string, error) {
	return s.url, s.err
}

func newMultipartImageRequest(t *testing.T, fieldName string, filename string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if fieldName != "" {
		part, err := writer.CreateFormFile(fieldName, filename)
		if err != nil {
			t.Fatalf("erro ao criar form file: %v", err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatalf("erro ao escrever bytes: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("erro ao fechar multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/products/image", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestUploadProductImageController(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validPNG := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D}

	t.Run("retorna 200 com URL ao enviar imagem valida", func(t *testing.T) {
		ctrl := NewProductController(nil, nil, nil, nil, nil, nil, use_cases.NewUploadProductImageUseCase(&stubImageUploader{
			url: "https://i.ibb.co/xyz/produto.png",
		}))

		r := gin.New()
		r.Use(middleware.MaxBodySizeMiddlewareWithRouteLimits(1<<20, map[string]int64{
			"POST /api/v1/products/image": 6 << 20,
		}))
		r.POST("/api/v1/products/image", ctrl.UploadProductImage)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, newMultipartImageRequest(t, "image", "produto.png", validPNG))

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, esperado 200 (body=%s)", w.Code, w.Body.String())
		}

		var payload struct {
			Body struct {
				URL string `json:"url"`
			} `json:"Body:"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
			t.Fatalf("erro ao decodificar JSON: %v", err)
		}
		if payload.Body.URL != "https://i.ibb.co/xyz/produto.png" {
			t.Fatalf("url = %q", payload.Body.URL)
		}
	})

	t.Run("retorna 400 quando campo image esta ausente", func(t *testing.T) {
		ctrl := NewProductController(nil, nil, nil, nil, nil, nil, use_cases.NewUploadProductImageUseCase(&stubImageUploader{}))
		r := gin.New()
		r.POST("/api/v1/products/image", ctrl.UploadProductImage)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, newMultipartImageRequest(t, "", "", nil))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, esperado 400", w.Code)
		}
	})

	t.Run("retorna 400 para arquivo que nao e imagem", func(t *testing.T) {
		ctrl := NewProductController(nil, nil, nil, nil, nil, nil, use_cases.NewUploadProductImageUseCase(&stubImageUploader{}))
		r := gin.New()
		r.POST("/api/v1/products/image", ctrl.UploadProductImage)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, newMultipartImageRequest(t, "image", "fake.png", []byte("not an image")))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, esperado 400", w.Code)
		}
	})

	t.Run("retorna 413 quando imagem excede limite de 5 MB", func(t *testing.T) {
		ctrl := NewProductController(nil, nil, nil, nil, nil, nil, use_cases.NewUploadProductImageUseCase(&stubImageUploader{}))
		r := gin.New()
		r.Use(middleware.MaxBodySizeMiddlewareWithRouteLimits(1<<20, map[string]int64{
			"POST /api/v1/products/image": 6 << 20,
		}))
		r.POST("/api/v1/products/image", ctrl.UploadProductImage)

		oversized := append(append([]byte(nil), validPNG...), make([]byte, use_cases.MaxProductImageSize)...)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newMultipartImageRequest(t, "image", "grande.png", oversized))

		if w.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, esperado 413 (body=%s)", w.Code, w.Body.String())
		}
	})

	t.Run("retorna 503 quando servico de upload esta indisponivel", func(t *testing.T) {
		ctrl := NewProductController(nil, nil, nil, nil, nil, nil, use_cases.NewUploadProductImageUseCase(&stubImageUploader{
			err: apperrors.ErrImageUploadUnavailable,
		}))
		r := gin.New()
		r.POST("/api/v1/products/image", ctrl.UploadProductImage)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, newMultipartImageRequest(t, "image", "produto.png", validPNG))

		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, esperado 503 (body=%s)", w.Code, w.Body.String())
		}
	})
}
