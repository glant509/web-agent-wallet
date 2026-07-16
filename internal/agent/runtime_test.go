package agent

import (
	"context"
	"encoding/json"
	"testing"
	"web3-service-agent/internal/llm"
	"web3-service-agent/internal/prompt"

	"web3-service-agent/internal/session"
	"web3-service-agent/internal/tool"
)

func TestRuntimeRunWithToolCall(t *testing.T) {
	registry := tool.NewRegistry()
	if err := registry.Register(tool.Definition{
		Name:        "evm_balance",
		Description: "return evm balance",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}, func(_ context.Context, call tool.Call) (tool.Result, error) {
		return tool.Result{Content: "balance=1 ETH"}, nil
	}); err != nil {
		t.Fatalf("register tool: %v", err)
	}

	client := &fakeLLM{
		responses: []llm.Response{
			{
				ToolCalls: []llm.ToolCall{
					{
						ID:        "call_1",
						Name:      "evm_balance",
						Arguments: json.RawMessage(`{"address":"0xabc"}`),
					},
				},
			},
			{
				Message: "wallet balance fetched",
			},
		},
	}

	promptBuilder, err := prompt.NewBuilder("web3-service-agent")
	if err != nil {
		t.Fatalf("new prompt builder: %v", err)
	}

	runtime := New(Dependencies{
		Sessions:      session.NewManager(),
		PromptBuilder: promptBuilder,
		LLM:           client,
		Tools:         registry,
		MaxSteps:      4,
	})

	response, err := runtime.Run(context.Background(), RunRequest{
		Input: "check my wallet balance",
	})
	if err != nil {
		t.Fatalf("run runtime: %v", err)
	}

	if response.Message != "wallet balance fetched" {
		t.Fatalf("unexpected final message: %q", response.Message)
	}

	if len(response.ToolExecutions) != 1 {
		t.Fatalf("expected 1 tool execution, got %d", len(response.ToolExecutions))
	}

	if response.ToolExecutions[0].Observation != "balance=1 ETH" {
		t.Fatalf("unexpected tool observation: %q", response.ToolExecutions[0].Observation)
	}
}

type fakeLLM struct {
	responses []llm.Response
	requests  []llm.Request
}

func (f *fakeLLM) Chat(_ context.Context, request llm.Request) (llm.Response, error) {
	f.requests = append(f.requests, cloneRequest(request))
	response := f.responses[0]
	f.responses = f.responses[1:]
	return response, nil
}

func (f *fakeLLM) Stream(_ context.Context, _ llm.Request) (<-chan llm.StreamEvent, error) {
	ch := make(chan llm.StreamEvent)
	close(ch)
	return ch, nil
}

func (f *fakeLLM) ToolCall(ctx context.Context, request llm.Request) (llm.Response, error) {
	return f.Chat(ctx, request)
}

func TestRuntimeRunPassesSessionHistoryToSubsequentCalls(t *testing.T) {
	client := &fakeLLM{
		responses: []llm.Response{
			{Message: "first answer"},
			{Message: "your previous question was: check btc price"},
		},
	}

	promptBuilder, err := prompt.NewBuilder("web3-service-agent")
	if err != nil {
		t.Fatalf("new prompt builder: %v", err)
	}

	runtime := New(Dependencies{
		Sessions:      session.NewManager(),
		PromptBuilder: promptBuilder,
		LLM:           client,
		Tools:         tool.NewRegistry(),
		MaxSteps:      4,
	})

	sessionID := "session-1"
	if _, err := runtime.Run(context.Background(), RunRequest{
		SessionID: sessionID,
		Input:     "check btc price",
	}); err != nil {
		t.Fatalf("first run failed: %v", err)
	}

	if _, err := runtime.Run(context.Background(), RunRequest{
		SessionID: sessionID,
		Input:     "what was my last question?",
	}); err != nil {
		t.Fatalf("second run failed: %v", err)
	}

	if len(client.requests) != 2 {
		t.Fatalf("expected 2 llm requests, got %d", len(client.requests))
	}

	second := client.requests[1]
	if len(second.Messages) != 3 {
		t.Fatalf("expected 3 messages in second request, got %d", len(second.Messages))
	}
	if second.Messages[0].Role != llm.RoleUser || second.Messages[0].Content != "check btc price" {
		t.Fatalf("unexpected first history message: %#v", second.Messages[0])
	}
	if second.Messages[1].Role != llm.RoleAssistant || second.Messages[1].Content != "first answer" {
		t.Fatalf("unexpected assistant history message: %#v", second.Messages[1])
	}
	if second.Messages[2].Role != llm.RoleUser || second.Messages[2].Content != "what was my last question?" {
		t.Fatalf("unexpected current user message: %#v", second.Messages[2])
	}
}

func cloneRequest(request llm.Request) llm.Request {
	messages := make([]llm.Message, len(request.Messages))
	copy(messages, request.Messages)
	tools := make([]llm.ToolDefinition, len(request.Tools))
	copy(tools, request.Tools)
	return llm.Request{
		SystemPrompt: request.SystemPrompt,
		Model:        request.Model,
		Messages:     messages,
		Tools:        tools,
	}
}
