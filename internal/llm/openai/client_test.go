package openai

import (
	"encoding/json"
	"testing"
	"web3-service-agent/internal/llm"
)

func TestBuildMessagesPreservesToolContext(t *testing.T) {
	messages := buildMessages("system prompt", []llm.Message{
		{Role: llm.RoleUser, Content: "check balance"},
		{
			Role:          llm.RoleAssistant,
			Name:          "evm_balance",
			ToolCallID:    "call_1",
			ToolArguments: json.RawMessage(`{"address":"0xabc"}`),
			Content:       "calling tool",
		},
		{Role: llm.RoleTool, ToolCallID: "call_1", Content: "balance=1 ETH"},
	})

	if len(messages) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(messages))
	}
	if messages[0].Role != llm.RoleSystem {
		t.Fatalf("expected system role, got %q", messages[0].Role)
	}
	if len(messages[2].ToolCalls) != 1 {
		t.Fatalf("expected assistant tool call message")
	}
	if messages[2].ToolCalls[0].Function.Name != "evm_balance" {
		t.Fatalf("unexpected tool call name: %q", messages[2].ToolCalls[0].Function.Name)
	}
	if messages[3].ToolCallID != "call_1" {
		t.Fatalf("unexpected tool call id: %q", messages[3].ToolCallID)
	}
}

func TestToLLMResponseParsesToolCalls(t *testing.T) {
	response := chatCompletionsResponse{
		Choices: []chatChoice{
			{
				Message: chatMessage{
					Content: "done",
					ToolCalls: []chatToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: chatFunction{
								Name:      "evm_balance",
								Arguments: `{"address":"0xabc"}`,
							},
						},
					},
				},
			},
		},
	}

	got := toLLMResponse(response, json.RawMessage(`{}`))
	if got.Message != "done" {
		t.Fatalf("unexpected message: %q", got.Message)
	}
	if len(got.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(got.ToolCalls))
	}
	if got.ToolCalls[0].Name != "evm_balance" {
		t.Fatalf("unexpected tool call name: %q", got.ToolCalls[0].Name)
	}
}
