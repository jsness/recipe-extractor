package worker

import (
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"recipe-extractor/server/extractor"
	"recipe-extractor/server/internal/config"
)

func TestBuildExtractorSupportsLocalWithoutAPIKey(t *testing.T) {
	ext, timeout, err := buildExtractor(config.Config{
		Extractor:              "local",
		LLMOnlyExtraction:      true,
		LocalLLMModel:          "local-model",
		LocalLLMBaseURL:        "http://localhost:1234/v1",
		LocalLLMTimeoutSeconds: 56,
	}, testLogger())
	if err != nil {
		t.Fatalf("buildExtractor() error = %v", err)
	}
	if _, ok := ext.(*extractor.LocalExtractor); !ok {
		t.Fatalf("extractor = %T, want *extractor.LocalExtractor", ext)
	}
	if timeout != 56*time.Second {
		t.Fatalf("timeout = %s, want 56s", timeout)
	}
}

func TestBuildExtractorRequiresCloudAPIKeyForLLMOnly(t *testing.T) {
	_, _, err := buildExtractor(config.Config{
		Extractor:         "openai",
		LLMOnlyExtraction: true,
	}, testLogger())
	if err == nil {
		t.Fatal("buildExtractor() error = nil, want missing configured extractor error")
	}
	if !strings.Contains(err.Error(), "LLM_ONLY_EXTRACTION requires a configured openai extractor") {
		t.Fatalf("buildExtractor() error = %q, want configured openai extractor error", err.Error())
	}
}

func TestBuildExtractorKeepsOpenAIPath(t *testing.T) {
	ext, timeout, err := buildExtractor(config.Config{
		Extractor:            "openai",
		LLMOnlyExtraction:    true,
		OpenAIAPIKey:         "openai-key",
		OpenAIModel:          "gpt-test",
		OpenAIBaseURL:        "https://api.example.com/v1",
		OpenAITimeoutSeconds: 12,
	}, testLogger())
	if err != nil {
		t.Fatalf("buildExtractor() error = %v", err)
	}
	if _, ok := ext.(*extractor.OpenAIExtractor); !ok {
		t.Fatalf("extractor = %T, want *extractor.OpenAIExtractor", ext)
	}
	if timeout != 12*time.Second {
		t.Fatalf("timeout = %s, want 12s", timeout)
	}
}

func TestBuildExtractorKeepsAnthropicPath(t *testing.T) {
	ext, timeout, err := buildExtractor(config.Config{
		Extractor:               "anthropic",
		LLMOnlyExtraction:       true,
		AnthropicAPIKey:         "anthropic-key",
		AnthropicModel:          "claude-test",
		AnthropicTimeoutSeconds: 34,
	}, testLogger())
	if err != nil {
		t.Fatalf("buildExtractor() error = %v", err)
	}
	if _, ok := ext.(*extractor.AnthropicExtractor); !ok {
		t.Fatalf("extractor = %T, want *extractor.AnthropicExtractor", ext)
	}
	if timeout != 34*time.Second {
		t.Fatalf("timeout = %s, want 34s", timeout)
	}
}

func testLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}
