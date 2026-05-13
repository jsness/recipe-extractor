package extractor

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	defaultLocalLLMBaseURL = "http://localhost:11434/v1"
	defaultLocalLLMModel   = "llama3.1"
)

type LocalConfig struct {
	APIKey  string
	Model   string
	BaseURL string
	Timeout time.Duration
	Logger  *log.Logger
}

type LocalExtractor struct {
	cfg        LocalConfig
	httpClient *http.Client
}

func NewLocal(cfg LocalConfig) *LocalExtractor {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = defaultLocalLLMBaseURL
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = defaultLocalLLMModel
	}
	return &LocalExtractor{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (e *LocalExtractor) NormalizeRecipe(ctx context.Context, input Input) (Recipe, error) {
	return normalizeWithChatCompletion(ctx, e.httpClient, input, chatCompletionConfig{
		ProviderName:          "local llm",
		APIKey:                e.cfg.APIKey,
		Model:                 e.cfg.Model,
		BaseURL:               e.cfg.BaseURL,
		IncludeResponseFormat: true,
		TrafficLogger:         e.cfg.Logger,
	})
}
