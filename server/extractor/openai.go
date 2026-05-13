package extractor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

type chatCompletionConfig struct {
	ProviderName          string
	MissingAPIKeyEnv      string
	APIKey                string
	Model                 string
	BaseURL               string
	ProjectID             string
	OrganizationID        string
	RequireAPIKey         bool
	IncludeResponseFormat bool
	TrafficLogger         *log.Logger
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
	return normalizeWithChatCompletion(ctx, e.httpClient, input, chatCompletionConfig{
		ProviderName:          "openai",
		MissingAPIKeyEnv:      "OPENAI_API_KEY",
		APIKey:                e.cfg.APIKey,
		Model:                 e.cfg.Model,
		BaseURL:               e.cfg.BaseURL,
		ProjectID:             e.cfg.ProjectID,
		OrganizationID:        e.cfg.OrganizationID,
		RequireAPIKey:         true,
		IncludeResponseFormat: true,
	})
}

func normalizeWithChatCompletion(ctx context.Context, httpClient *http.Client, input Input, cfg chatCompletionConfig) (Recipe, error) {
	if cfg.RequireAPIKey && strings.TrimSpace(cfg.APIKey) == "" {
		return Recipe{}, fmt.Errorf("%s is not configured", cfg.MissingAPIKeyEnv)
	}

	payload := map[string]any{
		"model": cfg.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": buildPrompt(input)},
		},
	}
	if cfg.IncludeResponseFormat {
		payload["response_format"] = map[string]string{
			"type": "json_object",
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return Recipe{}, err
	}

	url := strings.TrimRight(cfg.BaseURL, "/") + "/chat/completions"
	if cfg.TrafficLogger != nil {
		cfg.TrafficLogger.Printf("%s request url=%s body=%s", cfg.ProviderName, url, string(body))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return Recipe{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(cfg.APIKey) != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	if cfg.ProjectID != "" {
		req.Header.Set("OpenAI-Project", cfg.ProjectID)
	}
	if cfg.OrganizationID != "" {
		req.Header.Set("OpenAI-Organization", cfg.OrganizationID)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return Recipe{}, err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(io.LimitReader(res.Body, 2*1024*1024))
	if err != nil {
		return Recipe{}, err
	}
	if cfg.TrafficLogger != nil {
		cfg.TrafficLogger.Printf("%s response status=%d body=%s", cfg.ProviderName, res.StatusCode, string(resBody))
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Recipe{}, fmt.Errorf("%s request failed: status=%d body=%s", cfg.ProviderName, res.StatusCode, string(resBody))
	}

	var completion chatCompletionResponse
	if err := json.Unmarshal(resBody, &completion); err != nil {
		return Recipe{}, err
	}
	if len(completion.Choices) == 0 {
		return Recipe{}, fmt.Errorf("%s returned no choices", cfg.ProviderName)
	}

	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	if content == "" {
		return Recipe{}, fmt.Errorf("%s returned empty content", cfg.ProviderName)
	}

	recipe, err := parseModelRecipe(content)
	if err != nil {
		return Recipe{}, fmt.Errorf("failed to parse model json: %w", err)
	}

	normalizeRecipe(&recipe)
	reconcileRecipeWithStructuredData(&recipe, input)
	if err := validateRecipe(recipe); err != nil {
		return Recipe{}, err
	}

	return recipe, nil
}

func parseModelRecipe(content string) (Recipe, error) {
	content = strings.TrimSpace(content)

	candidates := []string{content}
	if fenced := extractFencedJSON(content); fenced != "" {
		candidates = append(candidates, fenced)
	}
	if object := extractJSONObject(content); object != "" {
		candidates = append(candidates, object)
	}

	var lastErr error
	for _, candidate := range candidates {
		var recipe Recipe
		if err := json.Unmarshal([]byte(candidate), &recipe); err != nil {
			lastErr = err
			continue
		}
		return recipe, nil
	}
	if lastErr != nil {
		return Recipe{}, lastErr
	}
	return Recipe{}, fmt.Errorf("empty model response")
}

func extractFencedJSON(content string) string {
	start := strings.Index(content, "```")
	if start == -1 {
		return ""
	}
	afterStart := content[start+3:]
	if newline := strings.IndexByte(afterStart, '\n'); newline != -1 {
		firstLine := strings.TrimSpace(afterStart[:newline])
		if firstLine == "" || strings.EqualFold(firstLine, "json") {
			afterStart = afterStart[newline+1:]
		}
	}
	end := strings.Index(afterStart, "```")
	if end == -1 {
		return ""
	}
	return strings.TrimSpace(afterStart[:end])
}

func extractJSONObject(content string) string {
	start := strings.IndexByte(content, '{')
	if start == -1 {
		return ""
	}

	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(content); i++ {
		ch := content[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch ch {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(content[start : i+1])
			}
		}
	}
	return ""
}

func reconcileRecipeWithStructuredData(recipe *Recipe, input Input) {
	structured, ok := bestStructuredRecipe(input)
	if !ok {
		return
	}

	if countIngredientItems(structured.Ingredients) > countIngredientItems(recipe.Ingredients) {
		recipe.Ingredients = structured.Ingredients
	}
	if len(structured.Instructions) >= len(recipe.Instructions) {
		recipe.Instructions = structured.Instructions
	}
	if recipe.Yield == nil && structured.Yield != nil {
		recipe.Yield = structured.Yield
	}
	if len(structured.Times) > 0 {
		if recipe.Times == nil {
			recipe.Times = map[string]string{}
		}
		for key, value := range structured.Times {
			if _, ok := recipe.Times[key]; !ok {
				recipe.Times[key] = value
			}
		}
	}

	normalizeRecipe(recipe)
}

func bestStructuredRecipe(input Input) (Recipe, bool) {
	var best Recipe
	found := false
	for _, raw := range input.JSONLD {
		recipe, err := tryParseJSONLD(raw, input.Ingredients)
		if err != nil {
			continue
		}
		if !found || structuredCompleteness(recipe) > structuredCompleteness(best) {
			best = recipe
			found = true
		}
	}
	return best, found
}

func structuredCompleteness(recipe Recipe) int {
	return countIngredientItems(recipe.Ingredients) + len(recipe.Instructions) + len(recipe.Times)
}

func countIngredientItems(groups []IngredientGroup) int {
	count := 0
	for _, group := range groups {
		count += len(group.Items)
	}
	return count
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}
