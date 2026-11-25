package ollama

import (
	"bytes"
	"context"
	"e-commerce/core/apperrors"
	"e-commerce/core/ports"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type ollamaClient struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewOllamaClient(host string, port string, model string) ports.LLMClient {
	host = strings.TrimSpace(host)
	baseURL := fmt.Sprintf("http://%s:%s", host, port)
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		baseURL = strings.TrimRight(host, "/")
	}

	return &ollamaClient{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

type ollamaChatRequest struct {
	Model    string                 `json:"model"`
	Messages []ports.ChatMessage    `json:"messages"`
	Tools    []ports.ToolDefinition `json:"tools,omitempty"`
	Stream   bool                   `json:"stream"`
}

type ollamaChatResponse struct {
	Message ports.ChatMessage `json:"message"`
	Error   string            `json:"error"`
}

func (c *ollamaClient) Chat(ctx context.Context, messages []ports.ChatMessage, tools []ports.ToolDefinition) (*ports.ChatResponse, error) {

	body, err := json.Marshal(ollamaChatRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    tools,
		Stream:   false,
	})

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(body))

	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)

	if err != nil {
		log.Printf("[ERROR] falha ao chamar Ollama: %v", err)
		return nil, fmt.Errorf("%w: %v", apperrors.ErrServiceUnavailable, err)
	}

	defer res.Body.Close()

	var ollamaRes ollamaChatResponse

	if err = json.NewDecoder(res.Body).Decode(&ollamaRes); err != nil && res.StatusCode < 300 {
		log.Printf("[ERROR] falha ao decodificar resposta do Ollama: %v", err)
		return nil, fmt.Errorf("%w: %v", apperrors.ErrServiceUnavailable, err)
	}

	if res.StatusCode < 200 || res.StatusCode > 299 || ollamaRes.Error != "" {
		log.Printf("[ERROR] Ollama retornou erro status=%d: %s", res.StatusCode, ollamaRes.Error)
		return nil, fmt.Errorf("%w: status %d %s", apperrors.ErrServiceUnavailable, res.StatusCode, ollamaRes.Error)
	}

	return &ports.ChatResponse{Message: ollamaRes.Message}, nil
}
