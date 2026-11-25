package openrouter

import (
	"bytes"
	"context"
	"e-commerce/core/apperrors"
	"e-commerce/core/ports"
	"encoding/json"
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

func newTestClient(apiKey string, model string, fn roundTripFunc) ports.LLMClient {
	client := newOpenRouterClientWithBaseURL("https://openrouter.test/api/v1", apiKey, model).(*openRouterClient)
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

func TestOpenRouterChatSucessoSimples(t *testing.T) {
	var gotAuth, gotTitle, gotURL, gotMethod string
	var gotReq openRouterChatRequest

	client := newTestClient("sk-or-test-key", "", func(r *http.Request) (*http.Response, error) {
		gotMethod = r.Method
		gotURL = r.URL.String()
		gotAuth = r.Header.Get("Authorization")
		gotTitle = r.Header.Get("X-Title")
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("erro ao decodificar request: %v", err)
		}

		return jsonResponse(http.StatusOK, `{
			"choices": [
				{
					"message": {
						"role": "assistant",
						"content": "Olá! Como posso ajudar?"
					}
				}
			]
		}`), nil
	})

	resp, err := client.Chat(context.Background(), []ports.ChatMessage{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "oi"},
	}, nil)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if gotMethod != http.MethodPost || gotURL != "https://openrouter.test/api/v1/chat/completions" {
		t.Fatalf("method/url = %s %s", gotMethod, gotURL)
	}
	if gotAuth != "Bearer sk-or-test-key" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotTitle != appTitle {
		t.Fatalf("X-Title = %q", gotTitle)
	}
	if gotReq.Model != "google/gemini-2.5-flash-lite" {
		t.Fatalf("model = %q", gotReq.Model)
	}
	if gotReq.ToolChoice != "" || len(gotReq.Tools) != 0 {
		t.Fatalf("nao deveria enviar tools: tools=%d tool_choice=%q", len(gotReq.Tools), gotReq.ToolChoice)
	}
	if resp.Message.Role != "assistant" || resp.Message.Content != "Olá! Como posso ajudar?" {
		t.Fatalf("resposta inesperada: %+v", resp.Message)
	}
}

func TestOpenRouterChatToolCallEMultiTurn(t *testing.T) {
	calls := 0
	var secondReq openRouterChatRequest

	client := newTestClient("sk-or-test-key", "google/gemini-2.5-flash-lite", func(r *http.Request) (*http.Response, error) {
		calls++
		var req openRouterChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode req: %v", err)
		}

		if calls == 1 {
			if req.ToolChoice != "auto" || len(req.Tools) != 1 {
				t.Fatalf("esperava tool_choice=auto e 1 tool, got choice=%q tools=%d", req.ToolChoice, len(req.Tools))
			}
			return jsonResponse(http.StatusOK, `{
				"choices": [
					{
						"message": {
							"role": "assistant",
							"content": null,
							"reasoning_details": [{"type":"reasoning.encrypted","data":"sig123"}],
							"tool_calls": [
								{
									"id": "call_order_1",
									"type": "function",
									"function": {
										"name": "consultar_pedido",
										"arguments": "{\"numero_pedido\":\"1234\"}"
									}
								}
							]
						}
					}
				]
			}`), nil
		}

		secondReq = req
		return jsonResponse(http.StatusOK, `{
			"choices": [
				{
					"message": {
						"role": "assistant",
						"content": "O status do seu pedido é Enviado."
					}
				}
			]
		}`), nil
	})

	tools := []ports.ToolDefinition{
		{
			Type: "function",
			Function: ports.ToolFunctionDefinition{
				Name:        "consultar_pedido",
				Description: "Consulta status do pedido",
				Parameters: ports.ToolParameters{
					Type: "object",
					Properties: map[string]ports.ToolProperty{
						"numero_pedido": {Type: "string", Description: "ID do pedido"},
					},
					Required: []string{},
				},
			},
		},
	}

	firstResp, err := client.Chat(context.Background(), []ports.ChatMessage{
		{Role: "user", Content: "status do pedido 1234"},
	}, tools)
	if err != nil {
		t.Fatalf("erro na 1a chamada: %v", err)
	}
	if len(firstResp.Message.ToolCalls) != 1 {
		t.Fatalf("esperava 1 tool call, got %d", len(firstResp.Message.ToolCalls))
	}
	tc := firstResp.Message.ToolCalls[0]
	if tc.ID != "call_order_1" || tc.Function.Name != "consultar_pedido" || tc.Function.Arguments["numero_pedido"] != "1234" {
		t.Fatalf("tool call inesperada: %+v", tc)
	}
	if !strings.Contains(string(firstResp.Message.ReasoningDetails), "sig123") {
		t.Fatalf("reasoning_details nao preservado: %s", string(firstResp.Message.ReasoningDetails))
	}

	secondResp, err := client.Chat(context.Background(), []ports.ChatMessage{
		{Role: "user", Content: "status do pedido 1234"},
		firstResp.Message,
		{Role: "tool", ToolCallID: tc.ID, ToolName: tc.Function.Name, Content: "Enviado"},
	}, tools)
	if err != nil {
		t.Fatalf("erro na 2a chamada: %v", err)
	}
	if secondResp.Message.Content != "O status do seu pedido é Enviado." {
		t.Fatalf("conteudo final inesperado: %q", secondResp.Message.Content)
	}

	if len(secondReq.Messages) != 3 {
		t.Fatalf("esperava 3 mensagens na 2a chamada, got %d", len(secondReq.Messages))
	}
	assistantWire := secondReq.Messages[1]
	if assistantWire.Content != nil {
		t.Fatalf("assistant tool_call content deveria ser null, got %q", *assistantWire.Content)
	}
	if len(assistantWire.ToolCalls) != 1 || assistantWire.ToolCalls[0].Type != "function" || assistantWire.ToolCalls[0].ID != "call_order_1" {
		t.Fatalf("assistant tool_calls wire invalido: %+v", assistantWire.ToolCalls)
	}
	if !strings.Contains(assistantWire.ToolCalls[0].Function.Arguments, `"numero_pedido":"1234"`) {
		t.Fatalf("arguments deveria ser JSON string, got %q", assistantWire.ToolCalls[0].Function.Arguments)
	}
	if !strings.Contains(string(assistantWire.ReasoningDetails), "sig123") {
		t.Fatalf("reasoning_details nao reenviado: %s", string(assistantWire.ReasoningDetails))
	}
	toolWire := secondReq.Messages[2]
	if toolWire.Role != "tool" || toolWire.ToolCallID != "call_order_1" || toolWire.Name != "consultar_pedido" {
		t.Fatalf("tool wire invalido: %+v", toolWire)
	}
}

func TestOpenRouterSanitizaHistoricoComToolsQuandoSemTools(t *testing.T) {
	var gotReq openRouterChatRequest

	client := newTestClient("sk-or-test-key", "google/gemini-2.5-flash-lite", func(r *http.Request) (*http.Response, error) {
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("decode req: %v", err)
		}
		return jsonResponse(http.StatusOK, `{"choices":[{"message":{"role":"assistant","content":"De nada!"}}]}`), nil
	})

	// Historico anterior teve tool_call + tool + resposta final; nova mensagem nao oferece tools
	_, err := client.Chat(context.Background(), []ports.ChatMessage{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "status do pedido"},
		{Role: "assistant", ToolCalls: []ports.ToolCall{{ID: "call_1", Function: ports.ToolCallFunction{Name: "consultar_pedido"}}}},
		{Role: "tool", ToolCallID: "call_1", ToolName: "consultar_pedido", Content: "Enviado"},
		{Role: "assistant", Content: "O status do seu pedido é Enviado."},
		{Role: "user", Content: "obrigado"},
	}, nil)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if len(gotReq.Messages) != 4 {
		t.Fatalf("esperava 4 mensagens sanitizadas, got %d: %+v", len(gotReq.Messages), gotReq.Messages)
	}
	for _, m := range gotReq.Messages {
		if m.Role == "tool" || len(m.ToolCalls) > 0 {
			t.Fatalf("mensagem com tool nao foi sanitizada: %+v", m)
		}
	}

	// Caso limite de iteracoes (tool pendente sem resposta final e tools=nil): preserva resultado como contexto user
	_, err = client.Chat(context.Background(), []ports.ChatMessage{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "status do pedido"},
		{Role: "assistant", ToolCalls: []ports.ToolCall{{ID: "call_1", Function: ports.ToolCallFunction{Name: "consultar_pedido"}}}},
		{Role: "tool", ToolCallID: "call_1", ToolName: "consultar_pedido", Content: "Enviado"},
	}, nil)
	if err != nil {
		t.Fatalf("erro inesperado no fallback de iteracoes: %v", err)
	}
	last := gotReq.Messages[len(gotReq.Messages)-1]
	if last.Role != "user" || last.Content == nil || !strings.Contains(*last.Content, "Enviado") {
		t.Fatalf("esperava resultado preservado como user, got %+v", last)
	}
}

func TestOpenRouterErrosRetornamServiceUnavailable(t *testing.T) {
	// Sem API key nao faz requisicao HTTP
	called := false
	clientNoKey := newTestClient("", "", func(r *http.Request) (*http.Response, error) {
		called = true
		return jsonResponse(http.StatusOK, `{}`), nil
	})
	if _, err := clientNoKey.Chat(context.Background(), []ports.ChatMessage{{Role: "user", Content: "oi"}}, nil); !errors.Is(err, apperrors.ErrServiceUnavailable) {
		t.Fatalf("sem key: got %v, want ErrServiceUnavailable", err)
	}
	if called {
		t.Fatal("nao deveria chamar HTTP sem API key")
	}

	// Erro HTTP 429
	clientErr := newTestClient("sk-or-test-key", "", func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusTooManyRequests, `{"error":{"code":429,"message":"Rate limit exceeded"}}`), nil
	})
	if _, err := clientErr.Chat(context.Background(), []ports.ChatMessage{{Role: "user", Content: "oi"}}, nil); !errors.Is(err, apperrors.ErrServiceUnavailable) || !strings.Contains(err.Error(), "Rate limit exceeded") {
		t.Fatalf("erro 429: got %v", err)
	}

	// HTTP 200 com objeto error no corpo (comum no OpenRouter quando provider falha)
	clientBodyErr := newTestClient("sk-or-test-key", "", func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"error":{"code":502,"message":"Provider returned error"}}`), nil
	})
	if _, err := clientBodyErr.Chat(context.Background(), []ports.ChatMessage{{Role: "user", Content: "oi"}}, nil); !errors.Is(err, apperrors.ErrServiceUnavailable) || !strings.Contains(err.Error(), "Provider returned error") {
		t.Fatalf("erro 200+error: got %v", err)
	}

	// HTTP 200 sem choices
	clientEmpty := newTestClient("sk-or-test-key", "", func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"choices":[]}`), nil
	})
	if _, err := clientEmpty.Chat(context.Background(), []ports.ChatMessage{{Role: "user", Content: "oi"}}, nil); !errors.Is(err, apperrors.ErrServiceUnavailable) || !strings.Contains(err.Error(), "sem choices") {
		t.Fatalf("sem choices: got %v", err)
	}

	// Erro de transporte/rede
	clientNetErr := newTestClient("sk-or-test-key", "", func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("connection reset")
	})
	if _, err := clientNetErr.Chat(context.Background(), []ports.ChatMessage{{Role: "user", Content: "oi"}}, nil); !errors.Is(err, apperrors.ErrServiceUnavailable) {
		t.Fatalf("erro de rede: got %v", err)
	}
}
