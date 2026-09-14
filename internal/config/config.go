package config

import (
	"bufio"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	defaultConfigFile    = "etc/application.properties"
	defaultServicePort   = "8080"
	defaultModelProvider = "openai"
	defaultModelBaseURL  = "https://api.openai.com/v1"
	defaultModelName     = "gpt-5-mini"
	defaultAgentMaxSteps = 8
	defaultServiceName   = "web3-service-agent"
	defaultLogPath       = "."
)

type Config struct {
	Service         *Service          `json:"service" yaml:"service, required"`
	Log             *Log              `json:"log" yaml:"log"`
	Models          []*ModelConfig    `json:"models" yaml:"models, required"`
	MarketProviders []*MarketProvider `json:"marketProviders" yaml:"marketProviders, required"`
	Chains          []*Chain          `json:"chains" yaml:"chains"`
}

type Log struct {
	Path string `json:"path" yaml:"path"`
}

type Chain struct {
	Type    string   `json:"type" yaml:"type"`
	Name    string   `json:"name" yaml:"name"`
	ChainID string   `json:"chain_id" yaml:"chain_id"`
	Urls    []string `json:"urls" yaml:"urls"`
	Used    bool     `json:"used" yaml:"used"`
}

type Wallet struct {
}

type Service struct {
	Name          string `json:"name" yaml:"name, required"`
	Port          string `json:"port" yaml:"port, required"`
	AgentMaxSteps int    `json:"agentMaxSteps" yaml:"agentMaxSteps, required"`
	ModelProvider string `json:"modelProvider" yaml:"modelProvider, required"`
	ModelName     string `json:"modelName" yaml:"modelName, required"`
}
type ModelConfig struct {
	Used     bool   `json:"used" yaml:"used, required"`
	Provider string `json:"provider" yaml:"provider, required"`
	Name     string `json:"name" yaml:"name, required"`
	BaseURL  string `json:"base_url" yaml:"base_url, required"`
	APIKey   string `json:"api_key" yaml:"api_key, required"`
}

type MarketProvider struct {
	Used    bool   `json:"used" yaml:"used, required"`
	Name    string `json:"name" yaml:"name, required"`
	BaseURL string `json:"base_url" yaml:"base_url, required"`
	APIKey  string `json:"api_key" yaml:"api_key, required"`
}

func Load() (Config, error) {
	return LoadFrom(defaultConfigFile)
}

func LoadFrom(path string) (Config, error) {
	properties, err := readProperties(path)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Service: &Service{
			Name:          getString(properties, "service.name", defaultServiceName),
			Port:          getString(properties, "service.port", defaultServicePort),
			AgentMaxSteps: getInt(properties, "service.agentMaxSteps", defaultAgentMaxSteps),
			ModelProvider: getString(properties, "service.modelProvider", defaultModelProvider),
			ModelName:     getString(properties, "service.modelName", defaultModelName),
		},
		Log: &Log{
			Path: getString(properties, "log.path", defaultLogPath),
		},
	}

	models, err := loadModels(properties)
	if err != nil {
		return Config{}, err
	}
	cfg.Models = models

	marketProviders, err := loadMarketProviders(properties)
	if err != nil {
		return Config{}, err
	}
	cfg.MarketProviders = marketProviders

	chains, err := loadChains(properties)
	if err != nil {
		return Config{}, err
	}
	cfg.Chains = chains

	if _, err := cfg.ActiveModel(); err != nil {
		return Config{}, err
	}
	if _, err := cfg.ActiveMarketProvider(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) ActiveModel() (*ModelConfig, error) {
	if c.Service == nil {
		return nil, errors.New("service config is required")
	}

	provider := strings.TrimSpace(c.Service.ModelProvider)
	modelName := strings.TrimSpace(c.Service.ModelName)
	if provider == "" || modelName == "" {
		return nil, errors.New("service.modelProvider and service.modelName are required")
	}

	var disabledMatch bool
	for _, model := range c.Models {
		if model == nil {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(model.Provider), provider) || strings.TrimSpace(model.Name) != modelName {
			continue
		}
		if !model.Used {
			disabledMatch = true
			continue
		}
		if model.BaseURL == "" && strings.EqualFold(provider, defaultModelProvider) {
			model.BaseURL = defaultModelBaseURL
		}
		return model, nil
	}

	if disabledMatch {
		return nil, fmt.Errorf("model %q under provider %q is configured but disabled", modelName, provider)
	}
	return nil, fmt.Errorf("enabled model %q under provider %q is not configured", modelName, provider)
}

func (c Config) ActiveMarketProvider() (*MarketProvider, error) {
	for _, provider := range c.MarketProviders {
		if provider == nil || !provider.Used {
			continue
		}
		if strings.TrimSpace(provider.Name) == "" {
			return nil, errors.New("market provider name is required")
		}
		if strings.TrimSpace(provider.BaseURL) == "" {
			return nil, fmt.Errorf("market provider %q base_url is required", provider.Name)
		}
		return provider, nil
	}
	return nil, errors.New("no enabled market provider configured")
}

func (c Config) ActiveChain(chainID string) (*Chain, error) {
	target := normalizeChainKey(chainID)
	if target == "" {
		return nil, errors.New("chain_id is required")
	}
	for _, chain := range c.Chains {
		if chain == nil || !chain.Used {
			continue
		}
		if normalizeChainKey(chain.ChainID) != target && normalizeChainKey(chain.Name) != target {
			continue
		}
		if len(chain.Urls) == 0 {
			return nil, fmt.Errorf("chain %q has no configured urls", chain.ChainID)
		}
		return chain, nil
	}
	return nil, fmt.Errorf("no enabled chain configured for %q", chainID)
}

func (c Config) RandomChainURL(chainID string) (string, error) {
	chain, err := c.ActiveChain(chainID)
	if err != nil {
		return "", err
	}
	return chain.RandomURL()
}

func (c Chain) RandomURL() (string, error) {
	urls := make([]string, 0, len(c.Urls))
	for _, value := range c.Urls {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			urls = append(urls, trimmed)
		}
	}
	if len(urls) == 0 {
		return "", fmt.Errorf("chain %q has no configured urls", c.ChainID)
	}
	if len(urls) == 1 {
		return urls[0], nil
	}
	index, err := rand.Int(rand.Reader, big.NewInt(int64(len(urls))))
	if err != nil {
		return "", fmt.Errorf("choose random url for chain %q: %w", c.ChainID, err)
	}
	return urls[index.Int64()], nil
}

func (s Service) ListenAddr() string {
	port := strings.TrimSpace(s.Port)
	if port == "" {
		port = defaultServicePort
	}
	if strings.HasPrefix(port, ":") {
		return port
	}
	return ":" + port
}

func readProperties(path string) (map[string]string, error) {
	dotenv, err := loadDotEnv(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("open config file %q: %w", path, err)
	}
	defer file.Close()

	properties := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}

		key, value, ok := splitProperty(line)
		if !ok {
			return nil, fmt.Errorf("invalid config line %q", line)
		}

		properties[key] = resolveValue(value, dotenv)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read config file %q: %w", path, err)
	}

	return properties, nil
}

func loadDotEnv(configPath string) (map[string]string, error) {
	dotenvPath, ok := findDotEnv(configPath)
	if !ok {
		return map[string]string{}, nil
	}

	file, err := os.Open(filepath.Clean(dotenvPath))
	if err != nil {
		return nil, fmt.Errorf("open dotenv file %q: %w", dotenvPath, err)
	}
	defer file.Close()

	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := splitEnvLine(line)
		if !ok {
			return nil, fmt.Errorf("invalid dotenv line %q", line)
		}
		values[key] = normalizeEnvValue(value)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read dotenv file %q: %w", dotenvPath, err)
	}

	return values, nil
}

func findDotEnv(configPath string) (string, bool) {
	dir := filepath.Dir(filepath.Clean(configPath))
	for {
		candidate := filepath.Join(dir, ".env")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func splitProperty(line string) (string, string, bool) {
	index := strings.IndexAny(line, "=:")
	if index <= 0 {
		return "", "", false
	}

	key := strings.TrimSpace(line[:index])
	value := strings.TrimSpace(line[index+1:])
	if key == "" {
		return "", "", false
	}
	return key, value, true
}

func splitEnvLine(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(line, "export "))
	index := strings.Index(trimmed, "=")
	if index <= 0 {
		return "", "", false
	}

	key := strings.TrimSpace(trimmed[:index])
	value := strings.TrimSpace(trimmed[index+1:])
	if key == "" {
		return "", "", false
	}
	return key, value, true
}

func normalizeEnvValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) >= 2 {
		if (trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"') || (trimmed[0] == '\'' && trimmed[len(trimmed)-1] == '\'') {
			return trimmed[1 : len(trimmed)-1]
		}
	}
	return trimmed
}

func resolveValue(value string, dotenv map[string]string) string {
	return os.Expand(value, func(name string) string {
		if current, ok := os.LookupEnv(name); ok && current != "" {
			return current
		}
		return dotenv[name]
	})
}

func loadModels(properties map[string]string) ([]*ModelConfig, error) {
	indexedModels := make(map[int]*ModelConfig)

	for key, value := range properties {
		if !strings.HasPrefix(key, "models.") {
			continue
		}

		index, field, err := parseModelPropertyKey(key)
		if err != nil {
			return nil, err
		}

		model := indexedModels[index]
		if model == nil {
			model = &ModelConfig{}
		}

		switch field {
		case "used":
			model.Used = strings.EqualFold(value, "true")
		case "provider":
			model.Provider = value
		case "name":
			model.Name = value
		case "base_url":
			model.BaseURL = value
		case "api_key":
			model.APIKey = value
		default:
			return nil, fmt.Errorf("unsupported model config field %q", key)
		}

		indexedModels[index] = model
	}

	indexes := make([]int, 0, len(indexedModels))
	for index := range indexedModels {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	models := make([]*ModelConfig, 0, len(indexes))
	for _, index := range indexes {
		model := indexedModels[index]
		if model == nil {
			continue
		}
		models = append(models, model)
	}

	return models, nil
}

func loadMarketProviders(properties map[string]string) ([]*MarketProvider, error) {
	indexedProviders := make(map[int]*MarketProvider)

	for key, value := range properties {
		if !strings.HasPrefix(key, "marketProviders.") {
			continue
		}

		index, field, err := parseIndexedPropertyKey(key, "marketProviders.")
		if err != nil {
			return nil, err
		}

		provider := indexedProviders[index]
		if provider == nil {
			provider = &MarketProvider{}
		}

		switch field {
		case "used":
			provider.Used = strings.EqualFold(value, "true")
		case "name":
			provider.Name = value
		case "base_url":
			provider.BaseURL = value
		case "api_key":
			provider.APIKey = value
		default:
			return nil, fmt.Errorf("unsupported market provider config field %q", key)
		}

		indexedProviders[index] = provider
	}

	indexes := make([]int, 0, len(indexedProviders))
	for index := range indexedProviders {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	providers := make([]*MarketProvider, 0, len(indexes))
	for _, index := range indexes {
		provider := indexedProviders[index]
		if provider == nil {
			continue
		}
		providers = append(providers, provider)
	}

	return providers, nil
}

func loadChains(properties map[string]string) ([]*Chain, error) {
	indexedChains := make(map[int]*Chain)

	for key, value := range properties {
		if !strings.HasPrefix(key, "chains.") {
			continue
		}

		index, field, err := parseIndexedPropertyKey(key, "chains.")
		if err != nil {
			return nil, err
		}

		chain := indexedChains[index]
		if chain == nil {
			chain = &Chain{}
		}

		switch {
		case field == "used":
			chain.Used = strings.EqualFold(value, "true")
		case field == "type":
			chain.Type = value
		case field == "name":
			chain.Name = value
		case field == "chain_id":
			chain.ChainID = value
		case strings.HasPrefix(field, "urls."):
			urlIndex, err := strconv.Atoi(strings.TrimPrefix(field, "urls."))
			if err != nil || urlIndex < 0 {
				return nil, fmt.Errorf("invalid chain url index in key %q", key)
			}
			if len(chain.Urls) <= urlIndex {
				expanded := make([]string, urlIndex+1)
				copy(expanded, chain.Urls)
				chain.Urls = expanded
			}
			chain.Urls[urlIndex] = value
		default:
			return nil, fmt.Errorf("unsupported chain config field %q", key)
		}

		indexedChains[index] = chain
	}

	indexes := make([]int, 0, len(indexedChains))
	for index := range indexedChains {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	chains := make([]*Chain, 0, len(indexes))
	for _, index := range indexes {
		chain := indexedChains[index]
		if chain == nil {
			continue
		}
		chain.Urls = compactStrings(chain.Urls)
		chains = append(chains, chain)
	}
	return chains, nil
}

func parseModelPropertyKey(key string) (int, string, error) {
	return parseIndexedPropertyKey(key, "models.")
}

func compactStrings(values []string) []string {
	compacted := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			compacted = append(compacted, trimmed)
		}
	}
	return compacted
}

func normalizeChainKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func parseIndexedPropertyKey(key, prefix string) (int, string, error) {
	remainder := strings.TrimPrefix(key, prefix)
	parts := strings.SplitN(remainder, ".", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid config key %q", key)
	}

	index, err := strconv.Atoi(strings.Trim(parts[0], "[]"))
	if err != nil || index < 0 {
		return 0, "", fmt.Errorf("invalid config index in key %q", key)
	}

	field := strings.TrimSpace(parts[1])
	if field == "" {
		return 0, "", fmt.Errorf("invalid config key %q", key)
	}

	return index, field, nil
}

func getString(properties map[string]string, key, fallback string) string {
	if value, ok := properties[key]; ok && value != "" {
		return value
	}
	return fallback
}

func getInt(properties map[string]string, key string, fallback int) int {
	raw, ok := properties[key]
	if !ok || raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
