package prompt

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"web3-service-agent/internal/tool"
)

const defaultTemplatePath = "internal/prompt/system_prompt.tmpl"

//go:embed system_prompt.tmpl
var defaultTemplate string

type Builder struct {
	serviceName string
	tmpl        *template.Template
}

type systemPromptData struct {
	ServiceName string
	ToolList    string
}

func NewBuilder(serviceName string) (*Builder, error) {
	return newBuilder(serviceName, defaultTemplatePath, defaultTemplate)
}

func NewBuilderFromPath(serviceName, path string) (*Builder, error) {
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("read prompt template %q: %w", path, err)
	}
	return newBuilder(serviceName, path, string(content))
}

func newBuilder(serviceName, name, content string) (*Builder, error) {
	tmpl, err := template.New(filepath.Base(name)).Parse(content)
	if err != nil {
		return nil, fmt.Errorf("parse prompt template %q: %w", name, err)
	}

	return &Builder{
		serviceName: serviceName,
		tmpl:        tmpl,
	}, nil
}

func (b *Builder) SystemPrompt(tools []tool.Definition) string {
	var toolList []string
	for _, definition := range tools {
		toolList = append(toolList, fmt.Sprintf("- %s: %s", definition.Name, definition.Description))
	}

	if len(toolList) == 0 {
		toolList = append(toolList, "- no tools registered yet")
	}

	var out bytes.Buffer
	if err := b.tmpl.Execute(&out, systemPromptData{
		ServiceName: b.serviceName,
		ToolList:    strings.Join(toolList, "\n"),
	}); err != nil {
		panic(fmt.Sprintf("render system prompt: %v", err))
	}

	return strings.TrimSpace(out.String())
}
