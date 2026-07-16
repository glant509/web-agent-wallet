package llm

import (
	"context"
	"encoding/json"
	"strings"
)

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

type Message struct {
	Role          string          `json:"role"`
	Content       string          `json:"content"`
	Name          string          `json:"name,omitempty"`
	ToolCallID    string          `json:"tool_call_id,omitempty"`
	ToolArguments json.RawMessage `json:"tool_arguments,omitempty"`
}

type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type Request struct {
	SystemPrompt string
	Model        string
	Messages     []Message
	Tools        []ToolDefinition
}

type Response struct {
	Message   string
	ToolCalls []ToolCall
	Raw       json.RawMessage
}

type StreamEvent struct {
	Type  string `json:"type"`
	Delta string `json:"delta,omitempty"`
	Done  bool   `json:"done,omitempty"`
	Err   error  `json:"-"`
}

type Client interface {
	Chat(ctx context.Context, request Request) (Response, error)
	Stream(ctx context.Context, request Request) (<-chan StreamEvent, error)
	ToolCall(ctx context.Context, request Request) (Response, error)
}

func Transcript(messages []Message) string {
	lines := make([]string, 0, len(messages))
	for _, message := range messages {
		header := message.Role
		if message.Name != "" {
			header += "[" + message.Name + "]"
		}
		if message.ToolCallID != "" {
			header += "#" + message.ToolCallID
		}
		lines = append(lines, header+": "+message.Content)
	}
	return strings.Join(lines, "\n")
}
