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
log.path=/var/tmp/web3-agent
models.0.provider=openai
models.0.name=gpt-5.4-mini
models.0.used=true
models.0.base_url=https://example.com/v1
models.0.api_key=${OPENAI_API_KEY}
marketProviders.0.used=true
marketProviders.0.name=coingecko
marketProviders.0.base_url=https://api.coingecko.com/api/v3
marketProviders.0.api_key=${CG-1HoLG61sqiEsjoXaieJUwmkq}
chains.0.used=true
chains.0.type=evm
chains.0.name=Ethereum
chains.0.chain_id=ethereum
chains.0.urls.0=https://rpc-a.example
chains.0.urls.1=https://rpc-b.example
chains.1.used=false
chains.1.type=evm
chains.1.name=Ignored
chains.1.chain_id=ignored
chains.1.urls.0=https://ignored.example
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
	if cfg.Log == nil || cfg.Log.Path != "/var/tmp/web3-agent" {
		t.Fatalf("unexpected log path: %#v", cfg.Log)
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
	provider, err := cfg.ActiveMarketProvider()
	if err != nil {
		t.Fatalf("resolve active market provider: %v", err)
	}
	if provider.APIKey != "" {
		t.Fatalf("unexpected provider api key: %q", provider.APIKey)
	}
	chain, err := cfg.ActiveChain("ethereum")
	if err != nil {
		t.Fatalf("resolve active chain: %v", err)
	}
	if chain.Type != "evm" {
		t.Fatalf("unexpected chain type: %q", chain.Type)
	}
	if len(chain.Urls) != 2 {
		t.Fatalf("unexpected chain url count: %d", len(chain.Urls))
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
CG-1HoLG61sqiEsjoXaieJUwmkq=cg-from-dotenv
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
marketProviders.0.used=true
marketProviders.0.name=coingecko
marketProviders.0.base_url=https://api.coingecko.com/api/v3
marketProviders.0.api_key=${CG-1HoLG61sqiEsjoXaieJUwmkq}
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
	provider, err := cfg.ActiveMarketProvider()
	if err != nil {
		t.Fatalf("resolve active market provider: %v", err)
	}
	if provider.APIKey != "cg-from-dotenv" {
		t.Fatalf("unexpected market provider api key: %q", provider.APIKey)
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
marketProviders.0.used=true
marketProviders.0.name=coingecko
marketProviders.0.base_url=https://api.coingecko.com/api/v3
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
	if cfg.Log == nil || cfg.Log.Path != defaultLogPath {
		t.Fatalf("unexpected default log path: %#v", cfg.Log)
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
marketProviders.0.used=false
marketProviders.0.name=disabled
marketProviders.0.base_url=https://disabled.example
marketProviders.1.used=true
marketProviders.1.name=coingecko
marketProviders.1.base_url=https://api.coingecko.com/api/v3
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
	provider, err := cfg.ActiveMarketProvider()
	if err != nil {
		t.Fatalf("resolve active market provider: %v", err)
	}
	if provider.Name != "coingecko" {
		t.Fatalf("unexpected active provider: %q", provider.Name)
	}
}

func TestRandomChainURLUsesConfiguredList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "application.properties")
	if err := os.WriteFile(path, []byte(`
models.0.provider=openai
models.0.name=gpt-5-mini
models.0.used=true
models.0.base_url=https://api.openai.com/v1
marketProviders.0.used=true
marketProviders.0.name=coingecko
marketProviders.0.base_url=https://api.coingecko.com/api/v3
chains.0.used=true
chains.0.type=evm
chains.0.name=Base
chains.0.chain_id=base
chains.0.urls.0=https://base-a.example
chains.0.urls.1=https://base-b.example
chains.0.urls.2=https://base-c.example
`), 0o644); err != nil {
		t.Fatalf("write properties: %v", err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	allowed := map[string]bool{
		"https://base-a.example": true,
		"https://base-b.example": true,
		"https://base-c.example": true,
	}
	for i := 0; i < 12; i++ {
		url, err := cfg.RandomChainURL("base")
		if err != nil {
			t.Fatalf("random chain url: %v", err)
		}
		if !allowed[url] {
			t.Fatalf("unexpected url selected: %q", url)
		}
	}
}
