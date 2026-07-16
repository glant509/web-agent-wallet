package provider

import (
	"fmt"
	"strings"
	"web3-service-agent/constants"
	"web3-service-agent/internal/config"
	"web3-service-agent/internal/llm"
	"web3-service-agent/internal/llm/openai"
)

func NewClient(cfg *config.Config) (llm.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	modelProvider := strings.TrimSpace(cfg.Service.ModelProvider)
	if strings.EqualFold(modelProvider, constants.MODEL_PROVIDER_OPENAI) ||
		strings.EqualFold(modelProvider, constants.MODEL_PROVIDER_OPENROUTER) ||
		strings.EqualFold(modelProvider, constants.MODEL_PROVIDER_DEEPSEEK) {
		model, err := cfg.ActiveModel()
		if err != nil {
			return nil, err
		}
		return openai.New(model.APIKey, model.BaseURL, model.Name), nil
	}

	return nil, fmt.Errorf("unsupported model provider: %s", modelProvider)
}
