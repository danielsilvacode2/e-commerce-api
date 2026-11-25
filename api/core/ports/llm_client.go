package ports

import (
	"context"
	"encoding/json"
)

type ToolCallFunction struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type ToolCall struct {
	ID       string           `json:"id,omitempty"`
	Function ToolCallFunction `json:"function"`
}

type ChatMessage struct {
	Role             string          `json:"role"`
	Content          string          `json:"content"`
	ToolCalls        []ToolCall      `json:"tool_calls,omitempty"`
	ToolCallID       string          `json:"tool_call_id,omitempty"`
	ToolName         string          `json:"tool_name,omitempty"`
	ReasoningDetails json.RawMessage `json:"reasoning_details,omitempty"`
}

type ToolProperty struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ToolParameters struct {
	Type       string                  `json:"type"`
	Properties map[string]ToolProperty `json:"properties"`
	Required   []string                `json:"required"`
}

type ToolFunctionDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  ToolParameters `json:"parameters"`
}

type ToolDefinition struct {
	Type     string                 `json:"type"`
	Function ToolFunctionDefinition `json:"function"`
}

type ChatResponse struct {
	Message ChatMessage
}

type LLMClient interface {
	Chat(ctx context.Context, messages []ChatMessage, tools []ToolDefinition) (*ChatResponse, error)
}
