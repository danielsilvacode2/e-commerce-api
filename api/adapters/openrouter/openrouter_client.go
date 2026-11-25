package openrouter

import (
	"bytes"
	"context"
	"e-commerce/core/apperrors"
	"e-commerce/core/ports"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://openrouter.ai/api/v1"
	defaultModel   = "google/gemini-2.5-flash-lite"
	appTitle       = "NERO E-Commerce"
)

type openRouterClient struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewOpenRouterClient(apiKey string, model string) ports.LLMClient {
	return newOpenRouterClientWithBaseURL(defaultBaseURL, apiKey, model)
}

func newOpenRouterClientWithBaseURL(baseURL string, apiKey string, model string) ports.LLMClient {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultModel
	}

	return &openRouterClient{
		baseURL: baseURL,
		apiKey:  strings.TrimSpace(apiKey),
		model:   model,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

type openRouterToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openRouterToolCall struct {
	ID       string                     `json:"id,omitempty"`
	Type     string                     `json:"type"`
	Function openRouterToolCallFunction `json:"function"`
}

type openRouterMessage struct {
	Role             string               `json:"role"`
	Content          *string              `json:"content"`
	ToolCalls        []openRouterToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string               `json:"tool_call_id,omitempty"`
	Name             string               `json:"name,omitempty"`
	ReasoningDetails json.RawMessage      `json:"reasoning_details,omitempty"`
}

type openRouterChatRequest struct {
	Model      string                 `json:"model"`
	Messages   []openRouterMessage    `json:"messages"`
	Tools      []ports.ToolDefinition `json:"tools,omitempty"`
	ToolChoice string                 `json:"tool_choice,omitempty"`
	Stream     bool                   `json:"stream"`
}

type openRouterError struct {
	Message string `json:"message"`
	Code    any    `json:"code"`
}

type openRouterChoice struct {
	Message openRouterMessage `json:"message"`
}

type openRouterChatResponse struct {
	Choices []openRouterChoice `json:"choices"`
	Error   *openRouterError   `json:"error,omitempty"`
}

func (c *openRouterClient) Chat(ctx context.Context, messages []ports.ChatMessage, tools []ports.ToolDefinition) (*ports.ChatResponse, error) {
	if c.apiKey == "" {
		log.Println("[ERROR] KEY_OPEN_ROUTER nao configurada")
		return nil, fmt.Errorf("%w: chave do OpenRouter nao configurada", apperrors.ErrServiceUnavailable)
	}

	reqPayload := openRouterChatRequest{
		Model:    c.model,
		Messages: buildWireMessages(messages, len(tools) > 0),
		Stream:   false,
	}
	if len(tools) > 0 {
		reqPayload.Tools = tools
		reqPayload.ToolChoice = "auto"
	}

	body, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Title", appTitle)

	res, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[ERROR] falha ao chamar OpenRouter: %v", err)
		return nil, fmt.Errorf("%w: %v", apperrors.ErrServiceUnavailable, err)
	}
	defer res.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		log.Printf("[ERROR] falha ao ler resposta do OpenRouter: %v", err)
		return nil, fmt.Errorf("%w: %v", apperrors.ErrServiceUnavailable, err)
	}

	var orRes openRouterChatResponse
	decodeErr := json.Unmarshal(respBytes, &orRes)

	if res.StatusCode < 200 || res.StatusCode > 299 || orRes.Error != nil {
		errMsg := ""
		if orRes.Error != nil {
			errMsg = orRes.Error.Message
		}
		if errMsg == "" {
			errMsg = strings.TrimSpace(string(respBytes))
			if len(errMsg) > 500 {
				errMsg = errMsg[:500]
			}
		}
		log.Printf("[ERROR] OpenRouter retornou erro status=%d: %s", res.StatusCode, errMsg)
		return nil, fmt.Errorf("%w: status %d %s", apperrors.ErrServiceUnavailable, res.StatusCode, errMsg)
	}

	if decodeErr != nil {
		log.Printf("[ERROR] falha ao decodificar resposta do OpenRouter: %v", decodeErr)
		return nil, fmt.Errorf("%w: %v", apperrors.ErrServiceUnavailable, decodeErr)
	}

	if len(orRes.Choices) == 0 {
		log.Println("[ERROR] OpenRouter retornou resposta sem choices")
		return nil, fmt.Errorf("%w: resposta sem choices", apperrors.ErrServiceUnavailable)
	}

	return &ports.ChatResponse{
		Message: parseWireMessage(orRes.Choices[0].Message),
	}, nil
}

func buildWireMessages(messages []ports.ChatMessage, hasTools bool) []openRouterMessage {
	wire := make([]openRouterMessage, 0, len(messages))

	for i, msg := range messages {
		if !hasTools {
			// Gemini/OpenRouter rejeitam historico com tool_calls/role=tool quando "tools" nao e enviado.
			if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
				if strings.TrimSpace(msg.Content) != "" {
					content := msg.Content
					wire = append(wire, openRouterMessage{
						Role:    "assistant",
						Content: &content,
					})
				}
				continue
			}
			if msg.Role == "tool" {
				if hasAssistantReplyAfter(messages, i) {
					continue
				}
				label := msg.ToolName
				if label == "" {
					label = "consulta"
				}
				content := fmt.Sprintf("[Resultado da ferramenta %s]: %s", label, msg.Content)
				wire = append(wire, openRouterMessage{
					Role:    "user",
					Content: &content,
				})
				continue
			}
		}

		wm := openRouterMessage{
			Role:             msg.Role,
			ToolCallID:       msg.ToolCallID,
			ReasoningDetails: msg.ReasoningDetails,
		}

		if msg.Role == "tool" {
			wm.Name = msg.ToolName
		}

		if len(msg.ToolCalls) > 0 {
			wm.ToolCalls = make([]openRouterToolCall, 0, len(msg.ToolCalls))
			for idx, tc := range msg.ToolCalls {
				argsJSON := "{}"
				if len(tc.Function.Arguments) > 0 {
					if raw, err := json.Marshal(tc.Function.Arguments); err == nil {
						argsJSON = string(raw)
					}
				}
				id := strings.TrimSpace(tc.ID)
				if id == "" {
					id = fmt.Sprintf("call_%d_%d", i, idx+1)
				}
				wm.ToolCalls = append(wm.ToolCalls, openRouterToolCall{
					ID:   id,
					Type: "function",
					Function: openRouterToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: argsJSON,
					},
				})
			}
			if msg.Content != "" {
				content := msg.Content
				wm.Content = &content
			}
		} else {
			content := msg.Content
			wm.Content = &content
		}

		wire = append(wire, wm)
	}

	return wire
}

func hasAssistantReplyAfter(messages []ports.ChatMessage, idx int) bool {
	for j := idx + 1; j < len(messages); j++ {
		if messages[j].Role == "assistant" && len(messages[j].ToolCalls) == 0 && strings.TrimSpace(messages[j].Content) != "" {
			return true
		}
	}
	return false
}

func parseWireMessage(wm openRouterMessage) ports.ChatMessage {
	role := wm.Role
	if role == "" {
		role = "assistant"
	}

	content := ""
	if wm.Content != nil {
		content = *wm.Content
	}

	msg := ports.ChatMessage{
		Role:             role,
		Content:          content,
		ReasoningDetails: wm.ReasoningDetails,
	}

	if len(wm.ToolCalls) > 0 {
		msg.ToolCalls = make([]ports.ToolCall, 0, len(wm.ToolCalls))
		for idx, tc := range wm.ToolCalls {
			args := make(map[string]any)
			rawArgs := strings.TrimSpace(tc.Function.Arguments)
			if rawArgs != "" {
				if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
					log.Printf("[WARN] falha ao decodificar arguments da tool %q: %v", tc.Function.Name, err)
				}
				if args == nil {
					args = make(map[string]any)
				}
			}

			id := strings.TrimSpace(tc.ID)
			if id == "" {
				id = fmt.Sprintf("call_%d", idx+1)
			}

			msg.ToolCalls = append(msg.ToolCalls, ports.ToolCall{
				ID: id,
				Function: ports.ToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: args,
				},
			})
		}
	}

	return msg
}
