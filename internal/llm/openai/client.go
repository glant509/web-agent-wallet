package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"web3-service-agent/internal/llm"
)

const chatCompletionsPath = "/chat/completions"

type Client struct {
	apiKey       string
	baseURL      string
	defaultModel string
	httpClient   *http.Client
}

type chatCompletionsRequest struct {
	Model      string               `json:"model"`
	Messages   []chatMessage        `json:"messages"`
	Tools      []chatToolDefinition `json:"tools,omitempty"`
	ToolChoice any                  `json:"tool_choice,omitempty"`
	Stream     bool                 `json:"stream,omitempty"`
}

type chatMessage struct {
	Role       string         `json:"role"`
	Content    any            `json:"content,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolCalls  []chatToolCall `json:"tool_calls,omitempty"`
}

type chatToolDefinition struct {
	Type     string       `json:"type"`
	Function chatFunction `json:"function"`
}

type chatToolCall struct {
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type"`
	Function chatFunction `json:"function"`
}

type chatFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Arguments   string          `json:"arguments,omitempty"`
}

type chatCompletionsResponse struct {
	Choices []chatChoice `json:"choices"`
}

type chatChoice struct {
	Message      chatMessage `json:"message"`
	Delta        chatMessage `json:"delta"`
	FinishReason string      `json:"finish_reason"`
}

func New(apiKey, baseURL, defaultModel string) *Client {
	return &Client{
		apiKey:       apiKey,
		baseURL:      strings.TrimRight(baseURL, "/"),
		defaultModel: defaultModel,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

func (c *Client) Chat(ctx context.Context, request llm.Request) (llm.Response, error) {
	return c.do(ctx, request, false)
}

func (c *Client) ToolCall(ctx context.Context, request llm.Request) (llm.Response, error) {
	return c.do(ctx, request, false)
}

func (c *Client) Stream(ctx context.Context, request llm.Request) (<-chan llm.StreamEvent, error) {
	if c.apiKey == "" {
		return nil, errors.New("api key is not configured")
	}

	payload := c.buildRequest(request, true)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal chat completions stream request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+chatCompletionsPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build chat completions stream request: %w", err)
	}
	c.setHeaders(httpRequest)

	resp, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("call chat completions stream api: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
		return nil, fmt.Errorf("chat completions stream api returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	events := make(chan llm.StreamEvent)
	go c.readStream(resp.Body, events)
	return events, nil
}

func (c *Client) do(ctx context.Context, request llm.Request, stream bool) (llm.Response, error) {
	if c.apiKey == "" {
		return llm.Response{}, errors.New("api key is not configured")
	}

	payload := c.buildRequest(request, stream)
	body, err := json.Marshal(payload)
	if err != nil {
		return llm.Response{}, fmt.Errorf("marshal chat completions request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+chatCompletionsPath, bytes.NewReader(body))
	if err != nil {
		return llm.Response{}, fmt.Errorf("build chat completions request: %w", err)
	}
	c.setHeaders(httpRequest)

	resp, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return llm.Response{}, fmt.Errorf("call chat completions api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
		return llm.Response{}, fmt.Errorf("chat completions api returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return llm.Response{}, fmt.Errorf("read chat completions response: %w", err)
	}

	var payloadResponse chatCompletionsResponse
	if err := json.Unmarshal(rawBody, &payloadResponse); err != nil {
		return llm.Response{}, fmt.Errorf("decode chat completions response: %w", err)
	}

	return toLLMResponse(payloadResponse, rawBody), nil
}

func (c *Client) buildRequest(request llm.Request, stream bool) chatCompletionsRequest {
	tools := make([]chatToolDefinition, 0, len(request.Tools))
	for _, definition := range request.Tools {
		tools = append(tools, chatToolDefinition{
			Type: "function",
			Function: chatFunction{
				Name:        definition.Name,
				Description: definition.Description,
				Parameters:  definition.InputSchema,
			},
		})
	}

	model := request.Model
	if model == "" {
		model = c.defaultModel
	}

	payload := chatCompletionsRequest{
		Model:    model,
		Messages: buildMessages(request.SystemPrompt, request.Messages),
		Tools:    tools,
		Stream:   stream,
	}
	if len(tools) > 0 {
		payload.ToolChoice = "auto"
	}
	return payload
}

func buildMessages(systemPrompt string, messages []llm.Message) []chatMessage {
	out := make([]chatMessage, 0, len(messages)+1)
	if strings.TrimSpace(systemPrompt) != "" {
		out = append(out, chatMessage{
			Role:    llm.RoleSystem,
			Content: systemPrompt,
		})
	}

	for _, message := range messages {
		out = append(out, toChatMessage(message))
	}

	return out
}

func toChatMessage(message llm.Message) chatMessage {
	switch message.Role {
	case llm.RoleAssistant:
		if message.Name != "" && message.ToolCallID != "" && len(message.ToolArguments) > 0 {
			return chatMessage{
				Role: llm.RoleAssistant,
				ToolCalls: []chatToolCall{
					{
						ID:   message.ToolCallID,
						Type: "function",
						Function: chatFunction{
							Name:      message.Name,
							Arguments: string(message.ToolArguments),
						},
					},
				},
			}
		}
		return chatMessage{
			Role:    message.Role,
			Content: message.Content,
		}
	case llm.RoleTool:
		return chatMessage{
			Role:       llm.RoleTool,
			Content:    message.Content,
			ToolCallID: message.ToolCallID,
		}
	default:
		return chatMessage{
			Role:    message.Role,
			Content: message.Content,
		}
	}
}

func toLLMResponse(payload chatCompletionsResponse, raw json.RawMessage) llm.Response {
	if len(payload.Choices) == 0 {
		return llm.Response{Raw: raw}
	}

	choice := payload.Choices[0]
	message := extractContent(choice.Message.Content)
	toolCalls := make([]llm.ToolCall, 0, len(choice.Message.ToolCalls))
	for _, toolCall := range choice.Message.ToolCalls {
		toolCalls = append(toolCalls, llm.ToolCall{
			ID:        toolCall.ID,
			Name:      toolCall.Function.Name,
			Arguments: json.RawMessage(toolCall.Function.Arguments),
		})
	}

	return llm.Response{
		Message:   message,
		ToolCalls: toolCalls,
		Raw:       raw,
	}
}

func (c *Client) setHeaders(request *http.Request) {
	request.Header.Set("Authorization", "Bearer "+c.apiKey)
	request.Header.Set("Content-Type", "application/json")
	if strings.Contains(request.URL.Host, "openrouter.ai") {
		request.Header.Set("HTTP-Referer", "https://github.com")
		request.Header.Set("X-Title", "web3-service-agent")
	}
}

func (c *Client) readStream(body io.ReadCloser, events chan<- llm.StreamEvent) {
	defer close(events)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var payload strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if payload.Len() == 0 {
				continue
			}

			event, done, emit, err := parseEvent(payload.String())
			payload.Reset()
			if err != nil {
				events <- llm.StreamEvent{Type: "error", Err: err}
				return
			}
			if emit {
				events <- event
			}
			if done {
				return
			}
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			payload.WriteString(strings.TrimPrefix(line, "data: "))
		}
	}

	if err := scanner.Err(); err != nil {
		events <- llm.StreamEvent{Type: "error", Err: err}
	}
}

func parseEvent(payload string) (llm.StreamEvent, bool, bool, error) {
	if payload == "[DONE]" {
		return llm.StreamEvent{Type: "done", Done: true}, true, true, nil
	}

	var raw chatCompletionsResponse
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return llm.StreamEvent{}, false, false, fmt.Errorf("decode stream event: %w", err)
	}

	if len(raw.Choices) == 0 {
		return llm.StreamEvent{}, false, false, nil
	}

	choice := raw.Choices[0]
	delta := extractContent(choice.Delta.Content)
	if delta != "" {
		return llm.StreamEvent{Type: "content.delta", Delta: delta}, false, true, nil
	}

	if choice.FinishReason != "" {
		return llm.StreamEvent{Type: "done", Done: true}, true, true, nil
	}

	return llm.StreamEvent{}, false, false, nil
}

func extractContent(content any) string {
	switch value := content.(type) {
	case string:
		return value
	case []any:
		var builder strings.Builder
		for _, item := range value {
			if part, ok := item.(map[string]any); ok {
				if text, ok := part["text"].(string); ok {
					builder.WriteString(text)
				}
			}
		}
		return builder.String()
	default:
		return ""
	}
}
