package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"web3-service-agent/internal/llm"
	"web3-service-agent/internal/prompt"

	"web3-service-agent/internal/session"
	"web3-service-agent/internal/tool"
)

type Dependencies struct {
	Sessions      *session.Manager
	PromptBuilder *prompt.Builder
	LLM           llm.Client
	Tools         *tool.Registry
	MaxSteps      int
}

type Runtime struct {
	sessions      *session.Manager
	promptBuilder *prompt.Builder
	llm           llm.Client
	tools         *tool.Registry
	maxSteps      int
}

type RunRequest struct {
	SessionID string `json:"session_id"`
	Input     string `json:"input"`
	Model     string `json:"model,omitempty"`
}

type ToolExecution struct {
	Name        string          `json:"name"`
	Arguments   json.RawMessage `json:"arguments,omitempty"`
	Observation string          `json:"observation"`
}

type RunResponse struct {
	SessionID      string          `json:"session_id"`
	Message        string          `json:"message"`
	Steps          int             `json:"steps"`
	ToolExecutions []ToolExecution `json:"tool_executions,omitempty"`
}

func New(deps Dependencies) *Runtime {
	maxSteps := deps.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 8
	}

	return &Runtime{
		sessions:      deps.Sessions,
		promptBuilder: deps.PromptBuilder,
		llm:           deps.LLM,
		tools:         deps.Tools,
		maxSteps:      maxSteps,
	}
}

func (r *Runtime) Run(ctx context.Context, request RunRequest) (RunResponse, error) {
	if strings.TrimSpace(request.Input) == "" {
		return RunResponse{}, errors.New("input is required")
	}

	snapshot := r.sessions.Ensure(request.SessionID)
	r.sessions.Append(snapshot.ID, llm.Message{
		Role:    llm.RoleUser,
		Content: request.Input,
	})

	executions := make([]ToolExecution, 0)
	for step := 1; step <= r.maxSteps; step++ {
		modelResponse, err := r.callModel(ctx, snapshot.ID, request.Model)
		if err != nil {
			return RunResponse{}, err
		}

		if modelResponse.Message != "" {
			r.sessions.Append(snapshot.ID, llm.Message{
				Role:    llm.RoleAssistant,
				Content: modelResponse.Message,
			})
		}

		if len(modelResponse.ToolCalls) == 0 {
			return RunResponse{
				SessionID:      snapshot.ID,
				Message:        modelResponse.Message,
				Steps:          step,
				ToolExecutions: executions,
			}, nil
		}

		for _, toolCall := range modelResponse.ToolCalls {
			r.sessions.Append(snapshot.ID, llm.Message{
				Role:          llm.RoleAssistant,
				Name:          toolCall.Name,
				ToolCallID:    toolCall.ID,
				ToolArguments: toolCall.Arguments,
				Content:       fmt.Sprintf("calling tool %s with arguments %s", toolCall.Name, compactJSON(toolCall.Arguments)),
			})

			result, err := r.tools.Execute(ctx, tool.Call{
				SessionID: snapshot.ID,
				Name:      toolCall.Name,
				Arguments: toolCall.Arguments,
			})

			observation := result.Content
			if err != nil {
				observation = fmt.Sprintf("tool %s failed: %v", toolCall.Name, err)
			}

			r.sessions.Append(snapshot.ID, llm.Message{
				Role:       llm.RoleTool,
				Name:       toolCall.Name,
				ToolCallID: toolCall.ID,
				Content:    observation,
			})

			executions = append(executions, ToolExecution{
				Name:        toolCall.Name,
				Arguments:   toolCall.Arguments,
				Observation: observation,
			})
		}
	}

	return RunResponse{}, fmt.Errorf("agent exceeded max steps (%d)", r.maxSteps)
}

func (r *Runtime) Stream(ctx context.Context, request RunRequest) (<-chan llm.StreamEvent, string, error) {
	if strings.TrimSpace(request.Input) == "" {
		return nil, "", errors.New("input is required")
	}

	snapshot := r.sessions.Ensure(request.SessionID)
	r.sessions.Append(snapshot.ID, llm.Message{
		Role:    llm.RoleUser,
		Content: request.Input,
	})

	if len(r.tools.Definitions()) > 0 {
		return nil, "", errors.New("streaming with tool-enabled runtime is not available yet")
	}

	stream, err := r.llm.Stream(ctx, llm.Request{
		SystemPrompt: r.promptBuilder.SystemPrompt(nil),
		Model:        request.Model,
		Messages:     r.sessions.GetOrEmpty(snapshot.ID).Messages,
	})
	if err != nil {
		return nil, "", err
	}

	out := make(chan llm.StreamEvent)
	go func() {
		defer close(out)

		var final strings.Builder
		for event := range stream {
			if event.Err != nil {
				out <- event
				return
			}

			if event.Delta != "" {
				final.WriteString(event.Delta)
			}
			out <- event

			if event.Done {
				if final.Len() > 0 {
					r.sessions.Append(snapshot.ID, llm.Message{
						Role:    llm.RoleAssistant,
						Content: final.String(),
					})
				}
				return
			}
		}
	}()

	return out, snapshot.ID, nil
}

func (r *Runtime) callModel(ctx context.Context, sessionID, model string) (llm.Response, error) {
	history := r.sessions.GetOrEmpty(sessionID).Messages
	request := llm.Request{
		SystemPrompt: r.promptBuilder.SystemPrompt(r.tools.Definitions()),
		Model:        model,
		Messages:     history,
		Tools:        r.tools.LLMDefinitions(),
	}

	if len(request.Tools) > 0 {
		return r.llm.ToolCall(ctx, request)
	}
	return r.llm.Chat(ctx, request)
}

func compactJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "{}"
	}

	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		return string(raw)
	}
	return compact.String()
}
