package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"web3-service-agent/internal/tool"
)

func TestBuilderRendersTemplateFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "system_prompt.tmpl")
	if err := os.WriteFile(path, []byte(`Service={{.ServiceName}}
Tools:
{{.ToolList}}`), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	builder, err := NewBuilderFromPath("web3-service-agent", path)
	if err != nil {
		t.Fatalf("new builder: %v", err)
	}

	prompt := builder.SystemPrompt([]tool.Definition{
		{Name: "market_token_overview", Description: "Get market data"},
	})

	if !strings.Contains(prompt, "Service=web3-service-agent") {
		t.Fatalf("unexpected prompt: %q", prompt)
	}
	if !strings.Contains(prompt, "- market_token_overview: Get market data") {
		t.Fatalf("missing tool list in prompt: %q", prompt)
	}
}
