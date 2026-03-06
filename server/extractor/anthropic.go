package extractor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type AnthropicConfig struct {
	APIKey  string
	Model   string
	Timeout time.Duration
}

type AnthropicExtractor struct {
	cfg        AnthropicConfig
	httpClient *http.Client
}

func NewAnthropic(cfg AnthropicConfig) *AnthropicExtractor {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	return &AnthropicExtractor{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (e *AnthropicExtractor) NormalizeRecipe(ctx context.Context, input Input) (Recipe, error) {
	if strings.TrimSpace(e.cfg.APIKey) == "" {
		return Recipe{}, fmt.Errorf("ANTHROPIC_API_KEY is not configured")
	}

	payload := map[string]any{
		"model":      e.cfg.Model,
		"max_tokens": 2048,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": buildPrompt(input),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return Recipe{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return Recipe{}, err
	}
	req.Header.Set("x-api-key", e.cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	res, err := e.httpClient.Do(req)
	if err != nil {
		return Recipe{}, err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(io.LimitReader(res.Body, 2*1024*1024))
	if err != nil {
		return Recipe{}, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Recipe{}, fmt.Errorf("anthropic request failed: status=%d body=%s", res.StatusCode, string(resBody))
	}

	var msg anthropicMessageResponse
	if err := json.Unmarshal(resBody, &msg); err != nil {
		return Recipe{}, err
	}
	if len(msg.Content) == 0 {
		return Recipe{}, fmt.Errorf("anthropic returned no content")
	}

	content := strings.TrimSpace(msg.Content[0].Text)
	if content == "" {
		return Recipe{}, fmt.Errorf("anthropic returned empty content")
	}

	// Strip markdown code fences if present
	if strings.HasPrefix(content, "```") {
		lines := strings.SplitN(content, "\n", 2)
		if len(lines) == 2 {
			content = strings.TrimSuffix(strings.TrimSpace(lines[1]), "```")
		}
	}

	var recipe Recipe
	if err := json.Unmarshal([]byte(content), &recipe); err != nil {
		return Recipe{}, fmt.Errorf("failed to parse model json: %w", err)
	}

	normalizeRecipe(&recipe)
	if err := validateRecipe(recipe); err != nil {
		return Recipe{}, err
	}

	return recipe, nil
}

type anthropicMessageResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}
