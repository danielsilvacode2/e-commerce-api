package imgbb

import (
	"bytes"
	"context"
	"e-commerce/core/apperrors"
	"e-commerce/core/ports"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestClient(apiKey string, fn roundTripFunc) ports.ImageUploader {
	client := newImgBBClientWithBaseURL("https://imgbb.test", apiKey).(*imgBBClient)
	client.httpClient = &http.Client{Transport: fn}
	return client
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

func TestImgBBUploadSucesso(t *testing.T) {
	var gotMethod, gotURL, gotQueryKey, gotFormKey, gotFilename string
	var gotFileBytes []byte

	expectedBytes := []byte("\x89PNG\r\n\x1a\nfake-png-content")
	client := newTestClient("secret-imgbb-key", func(r *http.Request) (*http.Response, error) {
		gotMethod = r.Method
		gotURL = r.URL.String()
		gotQueryKey = r.URL.Query().Get("key")

		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("erro ao parsear multipart: %v", err)
		}
		gotFormKey = r.FormValue("key")

		file, header, err := r.FormFile("image")
		if err != nil {
			t.Fatalf("campo image ausente: %v", err)
		}
		defer file.Close()
		gotFilename = header.Filename
		gotFileBytes, err = io.ReadAll(file)
		if err != nil {
			t.Fatalf("erro ao ler bytes da imagem: %v", err)
		}

		return jsonResponse(http.StatusOK, `{"data":{"url":"https://i.ibb.co/abc123/product.png"},"success":true,"status":200}`), nil
	})

	url, err := client.UploadImage(context.Background(), "product.png", expectedBytes)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if gotMethod != http.MethodPost || gotURL != "https://imgbb.test/1/upload" {
		t.Fatalf("method/url = %s %s", gotMethod, gotURL)
	}
	if gotQueryKey != "" {
		t.Fatalf("key nao deveria ir na query string, got %q", gotQueryKey)
	}
	if gotFormKey != "secret-imgbb-key" {
		t.Fatalf("form key = %q", gotFormKey)
	}
	if gotFilename != "product.png" {
		t.Fatalf("filename = %q", gotFilename)
	}
	if !bytes.Equal(gotFileBytes, expectedBytes) {
		t.Fatalf("bytes diferentes do esperado")
	}
	if url != "https://i.ibb.co/abc123/product.png" {
		t.Fatalf("url = %q", url)
	}
}

func TestImgBBUploadErros(t *testing.T) {
	// Sem API key nao dispara requisicao HTTP
	called := false
	clientNoKey := newTestClient("", func(r *http.Request) (*http.Response, error) {
		called = true
		return jsonResponse(http.StatusOK, `{}`), nil
	})
	if _, err := clientNoKey.UploadImage(context.Background(), "img.png", []byte("abc")); !errors.Is(err, apperrors.ErrImageUploadUnavailable) {
		t.Fatalf("sem key: got %v, want ErrImageUploadUnavailable", err)
	}
	if called {
		t.Fatal("nao deveria chamar HTTP sem API key")
	}

	// HTTP 400 do ImgBB
	clientBadReq := newTestClient("super-secret-key", func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, `{"status_code":400,"error":{"message":"Invalid API v1 key.","code":100}}`), nil
	})
	if _, err := clientBadReq.UploadImage(context.Background(), "img.png", []byte("abc")); !errors.Is(err, apperrors.ErrImageUploadUnavailable) {
		t.Fatalf("erro 400: got %v", err)
	} else if strings.Contains(err.Error(), "super-secret-key") {
		t.Fatalf("erro vazou a chave de API: %v", err)
	}

	// HTTP 200 com success=false
	clientSuccessFalse := newTestClient("super-secret-key", func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"success":false}`), nil
	})
	if _, err := clientSuccessFalse.UploadImage(context.Background(), "img.png", []byte("abc")); !errors.Is(err, apperrors.ErrImageUploadUnavailable) {
		t.Fatalf("success=false: got %v", err)
	}

	// HTTP 200 com URL vazia
	clientEmptyURL := newTestClient("super-secret-key", func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"data":{"url":""},"success":true,"status":200}`), nil
	})
	if _, err := clientEmptyURL.UploadImage(context.Background(), "img.png", []byte("abc")); !errors.Is(err, apperrors.ErrImageUploadUnavailable) {
		t.Fatalf("url vazia: got %v", err)
	}

	// Erro de rede
	clientNetErr := newTestClient("super-secret-key", func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("connection reset")
	})
	if _, err := clientNetErr.UploadImage(context.Background(), "img.png", []byte("abc")); !errors.Is(err, apperrors.ErrImageUploadUnavailable) {
		t.Fatalf("erro de rede: got %v", err)
	} else if strings.Contains(err.Error(), "super-secret-key") {
		t.Fatalf("erro de rede vazou a chave de API: %v", err)
	}
}
