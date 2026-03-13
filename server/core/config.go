package core

import internalconfig "recipe-extractor/server/internal/config"

type Config struct {
	DatabaseURL             string
	Extractor               string
	LLMOnlyExtraction       bool
	OpenAIAPIKey            string
	OpenAIModel             string
	OpenAIBaseURL           string
	OpenAIProjectID         string
	OpenAIOrganizationID    string
	OpenAITimeoutSeconds    int
	AnthropicAPIKey         string
	AnthropicModel          string
	AnthropicTimeoutSeconds int
}

func (c Config) workerConfig() internalconfig.Config {
	return internalconfig.Config{
		DatabaseURL:             c.DatabaseURL,
		Extractor:               c.Extractor,
		LLMOnlyExtraction:       c.LLMOnlyExtraction,
		OpenAIAPIKey:            c.OpenAIAPIKey,
		OpenAIModel:             c.OpenAIModel,
		OpenAIBaseURL:           c.OpenAIBaseURL,
		OpenAIProjectID:         c.OpenAIProjectID,
		OpenAIOrganizationID:    c.OpenAIOrganizationID,
		OpenAITimeoutSeconds:    c.OpenAITimeoutSeconds,
		AnthropicAPIKey:         c.AnthropicAPIKey,
		AnthropicModel:          c.AnthropicModel,
		AnthropicTimeoutSeconds: c.AnthropicTimeoutSeconds,
	}
}
