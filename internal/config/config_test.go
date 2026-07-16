package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromPropertiesFile(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "secret-from-env")

	dir := t.TempDir()
	path := filepath.Join(dir, "application.properties")
	content := []byte(`
# service
service.name=runtime-a
service.port=9090
service.agentMaxSteps=12
service.modelProvider=openai
service.modelName=gpt-5.4-mini
models.0.provider=openai
models.0.name=gpt-5.4-mini
models.0.used=true
models.0.base_url=https://example.com/v1
models.0.api_key=${OPENAI_API_KEY}
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write properties: %v", err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Service.Name != "runtime-a" {
		t.Fatalf("unexpected service name: %q", cfg.Service.Name)
	}
	if cfg.Service.ListenAddr() != ":9090" {
		t.Fatalf("unexpected http addr: %q", cfg.Service.ListenAddr())
	}
	if cfg.Service.AgentMaxSteps != 12 {
		t.Fatalf("unexpected max steps: %d", cfg.Service.AgentMaxSteps)
	}
	model, err := cfg.ActiveModel()
	if err != nil {
		t.Fatalf("resolve active model: %v", err)
	}
	if model == nil {
		t.Fatal("expected active model")
	}
	if model.Provider != "openai" {
		t.Fatalf("unexpected model provider: %q", model.Provider)
	}
	if model.Name != "gpt-5.4-mini" {
		t.Fatalf("unexpected model name: %q", model.Name)
	}
	if model.BaseURL != "https://example.com/v1" {
		t.Fatalf("unexpected model base url: %q", model.BaseURL)
	}
	if model.APIKey != "secret-from-env" {
		t.Fatalf("unexpected model api key: %q", model.APIKey)
	}
}

func TestLoadFromDotEnvFile(t *testing.T) {
	rootDir := t.TempDir()
	configDir := filepath.Join(rootDir, "etc")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}

	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "")

	if err := os.WriteFile(filepath.Join(rootDir, ".env"), []byte(`
OPENROUTER_API_KEY=openrouter-from-dotenv
DEEPSEEK_API_KEY=deepseek-from-dotenv
`), 0o644); err != nil {
		t.Fatalf("write dotenv: %v", err)
	}

	path := filepath.Join(configDir, "application.properties")
	if err := os.WriteFile(path, []byte(`
service.modelProvider=deepseek
service.modelName=deepseek-chat
models.0.provider=openrouter
models.0.name=openai/gpt-4.1-mini
models.0.used=true
models.0.base_url=https://openrouter.ai/api/v1
models.0.api_key=${OPENROUTER_API_KEY}
models.1.provider=deepseek
models.1.name=deepseek-chat
models.1.used=true
models.1.base_url=https://api.deepseek.com
models.1.api_key=${DEEPSEEK_API_KEY}
`), 0o644); err != nil {
		t.Fatalf("write properties: %v", err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Models[0].APIKey != "openrouter-from-dotenv" {
		t.Fatalf("unexpected openrouter api key: %q", cfg.Models[0].APIKey)
	}

	model, err := cfg.ActiveModel()
	if err != nil {
		t.Fatalf("resolve active model: %v", err)
	}
	if model.APIKey != "deepseek-from-dotenv" {
		t.Fatalf("unexpected deepseek api key: %q", model.APIKey)
	}
}

func TestLoadFromUsesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "application.properties")
	if err := os.WriteFile(path, []byte(`
service.name=
models.0.provider=openai
models.0.name=gpt-5-mini
models.0.used=true
models.0.base_url=https://api.openai.com/v1
`), 0o644); err != nil {
		t.Fatalf("write properties: %v", err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Service.Name != defaultServiceName {
		t.Fatalf("unexpected default service name: %q", cfg.Service.Name)
	}
	model, err := cfg.ActiveModel()
	if err != nil {
		t.Fatalf("resolve default model: %v", err)
	}
	if model == nil {
		t.Fatal("expected default model")
	}
	if model.Provider != defaultModelProvider {
		t.Fatalf("unexpected default provider: %q", model.Provider)
	}
	if model.Name != defaultModelName {
		t.Fatalf("unexpected default model name: %q", model.Name)
	}
}

func TestActiveModelSkipsDisabledEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "application.properties")
	if err := os.WriteFile(path, []byte(`
service.modelProvider=deepseek
service.modelName=deepseek-chat
models.0.provider=deepseek
models.0.name=deepseek-chat
models.0.used=false
models.0.base_url=https://api.deepseek.com
models.1.provider=deepseek
models.1.name=deepseek-chat
models.1.used=true
models.1.base_url=https://api.deepseek.com
models.1.api_key=${DEEPSEEK_API_KEY}
`), 0o644); err != nil {
		t.Fatalf("write properties: %v", err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	model, err := cfg.ActiveModel()
	if err != nil {
		t.Fatalf("resolve active model: %v", err)
	}
	if model == nil {
		t.Fatal("expected active model")
	}
	if model.Used != true {
		t.Fatal("expected enabled model to be selected")
	}
}
