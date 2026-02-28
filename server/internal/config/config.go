package config

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr             string
	DatabaseURL          string
	Extractor            string
	OpenAIAPIKey         string
	OpenAIModel          string
	OpenAIBaseURL        string
	OpenAIProjectID      string
	OpenAIOrganizationID string
	OpenAITimeoutSeconds int
	AnthropicAPIKey      string
	AnthropicModel       string
	AnthropicTimeoutSeconds int
}

func LoadFromEnv() Config {
	addr := getenv("HTTP_ADDR", ":8080")
	db := getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/recipes?sslmode=disable")
	return Config{
		HTTPAddr:                addr,
		DatabaseURL:             db,
		Extractor:               getenv("EXTRACTOR", "openai"),
		OpenAIAPIKey:            getenv("OPENAI_API_KEY", ""),
		OpenAIModel:             getenv("OPENAI_MODEL", "gpt-5-mini"),
		OpenAIBaseURL:           getenv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		OpenAIProjectID:         getenv("OPENAI_PROJECT_ID", ""),
		OpenAIOrganizationID:    getenv("OPENAI_ORGANIZATION_ID", ""),
		OpenAITimeoutSeconds:    getenvInt("OPENAI_TIMEOUT_SECONDS", 45),
		AnthropicAPIKey:         getenv("ANTHROPIC_API_KEY", ""),
		AnthropicModel:          getenv("ANTHROPIC_MODEL", "claude-sonnet-4-6"),
		AnthropicTimeoutSeconds: getenvInt("ANTHROPIC_TIMEOUT_SECONDS", 45),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return parsed
}
