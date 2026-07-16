package tool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"web3-service-agent/internal/llm"
)

type Definition struct {
	Name        string
	Description string
	InputSchema json.RawMessage
}

type Call struct {
	SessionID string
	Name      string
	Arguments json.RawMessage
}

type Result struct {
	Content string
}

type Handler func(ctx context.Context, call Call) (Result, error)

type Registry struct {
	mu       sync.RWMutex
	handlers map[string]registeredTool
}

type registeredTool struct {
	definition Definition
	handler    Handler
}

func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]registeredTool),
	}
}

func (r *Registry) Register(definition Definition, handler Handler) error {
	if definition.Name == "" {
		return errors.New("tool name is required")
	}
	if handler == nil {
		return fmt.Errorf("tool %q handler is required", definition.Name)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.handlers[definition.Name]; exists {
		return fmt.Errorf("tool %q already registered", definition.Name)
	}

	r.handlers[definition.Name] = registeredTool{
		definition: definition,
		handler:    handler,
	}
	return nil
}

func (r *Registry) Definitions() []Definition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		names = append(names, name)
	}
	sort.Strings(names)

	definitions := make([]Definition, 0, len(names))
	for _, name := range names {
		definitions = append(definitions, r.handlers[name].definition)
	}
	return definitions
}

func (r *Registry) LLMDefinitions() []llm.ToolDefinition {
	definitions := r.Definitions()
	items := make([]llm.ToolDefinition, 0, len(definitions))
	for _, definition := range definitions {
		items = append(items, llm.ToolDefinition{
			Name:        definition.Name,
			Description: definition.Description,
			InputSchema: definition.InputSchema,
		})
	}
	return items
}

func (r *Registry) Execute(ctx context.Context, call Call) (Result, error) {
	r.mu.RLock()
	registered, exists := r.handlers[call.Name]
	r.mu.RUnlock()
	if !exists {
		return Result{}, fmt.Errorf("tool %q is not registered", call.Name)
	}

	return registered.handler(ctx, call)
}
