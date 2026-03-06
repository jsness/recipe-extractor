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

type OpenAIConfig struct {
	APIKey         string
	Model          string
	BaseURL        string
	ProjectID      string
	OrganizationID string
	Timeout        time.Duration
}

type OpenAIExtractor struct {
	cfg        OpenAIConfig
	httpClient *http.Client
}

func NewOpenAI(cfg OpenAIConfig) *OpenAIExtractor {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	return &OpenAIExtractor{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (e *OpenAIExtractor) NormalizeRecipe(ctx context.Context, input Input) (Recipe, error) {
	if strings.TrimSpace(e.cfg.APIKey) == "" {
		return Recipe{}, fmt.Errorf("OPENAI_API_KEY is not configured")
	}

	payload := map[string]any{
		"model": e.cfg.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": buildPrompt(input)},
		},
		"response_format": map[string]string{
			"type": "json_object",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return Recipe{}, err
	}

	url := strings.TrimRight(e.cfg.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return Recipe{}, err
	}
	req.Header.Set("Authorization", "Bearer "+e.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	if e.cfg.ProjectID != "" {
		req.Header.Set("OpenAI-Project", e.cfg.ProjectID)
	}
	if e.cfg.OrganizationID != "" {
		req.Header.Set("OpenAI-Organization", e.cfg.OrganizationID)
	}

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
		return Recipe{}, fmt.Errorf("openai request failed: status=%d body=%s", res.StatusCode, string(resBody))
	}

	var completion chatCompletionResponse
	if err := json.Unmarshal(resBody, &completion); err != nil {
		return Recipe{}, err
	}
	if len(completion.Choices) == 0 {
		return Recipe{}, fmt.Errorf("openai returned no choices")
	}

	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	if content == "" {
		return Recipe{}, fmt.Errorf("openai returned empty content")
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

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}
