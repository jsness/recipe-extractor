package config

import "testing"

func TestLoadFromEnv(t *testing.T) {
	envKeys := []string{
		"HTTP_ADDR",
		"FRONTEND_DEV_PROXY_URL",
		"DATABASE_URL",
		"EXTRACTOR",
		"LLM_ONLY_EXTRACTION",
		"OPENAI_API_KEY",
		"OPENAI_MODEL",
		"OPENAI_BASE_URL",
		"OPENAI_PROJECT_ID",
		"OPENAI_ORGANIZATION_ID",
		"OPENAI_TIMEOUT_SECONDS",
		"ANTHROPIC_API_KEY",
		"ANTHROPIC_MODEL",
		"ANTHROPIC_TIMEOUT_SECONDS",
	}

	tests := []struct {
		name string
		env  map[string]string
		want Config
	}{
		{
			name: "defaults",
			want: Config{
				HTTPAddr:                ":8080",
				DatabaseURL:             "postgres://postgres:postgres@localhost:5433/recipes?sslmode=disable",
				Extractor:               "openai",
				OpenAIModel:             "gpt-5-mini",
				OpenAIBaseURL:           "https://api.openai.com/v1",
				OpenAITimeoutSeconds:    45,
				AnthropicModel:          "claude-sonnet-4-6",
				AnthropicTimeoutSeconds: 45,
			},
		},
		{
			name: "overrides",
			env: map[string]string{
				"HTTP_ADDR":                 ":9090",
				"FRONTEND_DEV_PROXY_URL":    "http://localhost:5173",
				"DATABASE_URL":              "postgres://example",
				"EXTRACTOR":                 "anthropic",
				"LLM_ONLY_EXTRACTION":       "true",
				"OPENAI_API_KEY":            "openai-key",
				"OPENAI_MODEL":              "gpt-test",
				"OPENAI_BASE_URL":           "https://api.example.com/v1",
				"OPENAI_PROJECT_ID":         "project",
				"OPENAI_ORGANIZATION_ID":    "org",
				"OPENAI_TIMEOUT_SECONDS":    "12",
				"ANTHROPIC_API_KEY":         "anthropic-key",
				"ANTHROPIC_MODEL":           "claude-test",
				"ANTHROPIC_TIMEOUT_SECONDS": "34",
			},
			want: Config{
				HTTPAddr:                ":9090",
				FrontendDevProxyURL:     "http://localhost:5173",
				DatabaseURL:             "postgres://example",
				Extractor:               "anthropic",
				LLMOnlyExtraction:       true,
				OpenAIAPIKey:            "openai-key",
				OpenAIModel:             "gpt-test",
				OpenAIBaseURL:           "https://api.example.com/v1",
				OpenAIProjectID:         "project",
				OpenAIOrganizationID:    "org",
				OpenAITimeoutSeconds:    12,
				AnthropicAPIKey:         "anthropic-key",
				AnthropicModel:          "claude-test",
				AnthropicTimeoutSeconds: 34,
			},
		},
		{
			name: "invalid parsed values fall back",
			env: map[string]string{
				"LLM_ONLY_EXTRACTION":       "not-bool",
				"OPENAI_TIMEOUT_SECONDS":    "not-int",
				"ANTHROPIC_TIMEOUT_SECONDS": "also-not-int",
			},
			want: Config{
				HTTPAddr:                ":8080",
				DatabaseURL:             "postgres://postgres:postgres@localhost:5433/recipes?sslmode=disable",
				Extractor:               "openai",
				OpenAIModel:             "gpt-5-mini",
				OpenAIBaseURL:           "https://api.openai.com/v1",
				OpenAITimeoutSeconds:    45,
				AnthropicModel:          "claude-sonnet-4-6",
				AnthropicTimeoutSeconds: 45,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, key := range envKeys {
				t.Setenv(key, "")
			}
			for key, value := range tc.env {
				t.Setenv(key, value)
			}

			if got := LoadFromEnv(); got != tc.want {
				t.Fatalf("LoadFromEnv() = %#v, want %#v", got, tc.want)
			}
		})
	}
}
