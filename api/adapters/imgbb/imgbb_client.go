package imgbb

import (
	"bytes"
	"context"
	"e-commerce/core/apperrors"
	"e-commerce/core/ports"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://api.imgbb.com"
	maxRespBytes   = 1 << 20
)

type imgBBClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewImgBBClient(apiKey string) ports.ImageUploader {
	return newImgBBClientWithBaseURL(defaultBaseURL, apiKey)
}

func newImgBBClientWithBaseURL(baseURL string, apiKey string) ports.ImageUploader {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &imgBBClient{
		baseURL: baseURL,
		apiKey:  strings.TrimSpace(apiKey),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type imgBBError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type imgBBData struct {
	URL string `json:"url"`
}

type imgBBResponse struct {
	Data    imgBBData   `json:"data"`
	Success bool        `json:"success"`
	Status  int         `json:"status"`
	Error   *imgBBError `json:"error,omitempty"`
}

func (c *imgBBClient) UploadImage(ctx context.Context, filename string, data []byte) (string, error) {
	if c.apiKey == "" {
		log.Println("[ERROR] IMGBB_API_KEY nao configurada")
		return "", fmt.Errorf("%w: chave do ImgBB nao configurada", apperrors.ErrImageUploadUnavailable)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Envia a chave no corpo multipart em vez da query string para evitar vazamento em URLs ou logs.
	if err := writer.WriteField("key", c.apiKey); err != nil {
		return "", fmt.Errorf("%w: falha ao preparar chave", apperrors.ErrImageUploadUnavailable)
	}

	part, err := writer.CreateFormFile("image", filename)
	if err != nil {
		return "", fmt.Errorf("%w: falha ao preparar arquivo", apperrors.ErrImageUploadUnavailable)
	}
	if _, err := part.Write(data); err != nil {
		return "", fmt.Errorf("%w: falha ao gravar imagem", apperrors.ErrImageUploadUnavailable)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("%w: falha ao finalizar multipart", apperrors.ErrImageUploadUnavailable)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/1/upload", &body)
	if err != nil {
		return "", fmt.Errorf("%w: falha ao criar requisicao", apperrors.ErrImageUploadUnavailable)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[ERROR] falha ao enviar imagem para ImgBB: %v", err)
		return "", fmt.Errorf("%w: falha de comunicacao com ImgBB", apperrors.ErrImageUploadUnavailable)
	}
	defer res.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(res.Body, maxRespBytes))
	if err != nil {
		log.Printf("[ERROR] falha ao ler resposta do ImgBB: %v", err)
		return "", fmt.Errorf("%w: falha ao ler resposta", apperrors.ErrImageUploadUnavailable)
	}

	var imgRes imgBBResponse
	decodeErr := json.Unmarshal(respBytes, &imgRes)

	if res.StatusCode < 200 || res.StatusCode > 299 || !imgRes.Success {
		errMsg := ""
		if imgRes.Error != nil {
			errMsg = imgRes.Error.Message
		}
		if errMsg == "" {
			errMsg = strings.TrimSpace(string(respBytes))
			if len(errMsg) > 300 {
				errMsg = errMsg[:300]
			}
		}
		log.Printf("[ERROR] ImgBB retornou erro status=%d: %s", res.StatusCode, errMsg)
		return "", fmt.Errorf("%w: status %d", apperrors.ErrImageUploadUnavailable, res.StatusCode)
	}

	if decodeErr != nil {
		log.Printf("[ERROR] falha ao decodificar resposta do ImgBB: %v", decodeErr)
		return "", fmt.Errorf("%w: resposta invalida", apperrors.ErrImageUploadUnavailable)
	}

	uploadedURL := strings.TrimSpace(imgRes.Data.URL)
	if uploadedURL == "" {
		log.Println("[ERROR] ImgBB retornou sucesso sem URL da imagem")
		return "", fmt.Errorf("%w: resposta sem url", apperrors.ErrImageUploadUnavailable)
	}

	return uploadedURL, nil
}
