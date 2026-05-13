package extractor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLocalExtractorNormalizeRecipe(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotPayload map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "{\"title\":\" Tomato Soup \",\"ingredients\":[{\"group\":\" Soup \",\"items\":[\" tomatoes \"]}],\"instructions\":[\" Simmer. \"],\"times\":{},\"linked_recipe_urls\":[]}"
				}
			}]
		}`))
	}))
	defer srv.Close()

	ext := NewLocal(LocalConfig{
		Model:   "local-model",
		BaseURL: srv.URL + "/v1",
	})

	recipe, err := ext.NormalizeRecipe(context.Background(), Input{
		SourceURL: "https://example.com/recipe",
		Text:      "Tomato soup recipe",
	})
	if err != nil {
		t.Fatalf("NormalizeRecipe() error = %v", err)
	}

	if gotPath != "/v1/chat/completions" {
		t.Fatalf("request path = %q, want /v1/chat/completions", gotPath)
	}
	if gotAuth != "" {
		t.Fatalf("Authorization header = %q, want empty", gotAuth)
	}
	if gotPayload["model"] != "local-model" {
		t.Fatalf("model = %q, want local-model", gotPayload["model"])
	}
	responseFormat, ok := gotPayload["response_format"].(map[string]any)
	if !ok {
		t.Fatalf("response_format = %#v, want object", gotPayload["response_format"])
	}
	if responseFormat["type"] != "json_object" {
		t.Fatalf("response_format.type = %q, want json_object", responseFormat["type"])
	}
	if recipe.Title != "Tomato Soup" {
		t.Fatalf("Title = %q, want Tomato Soup", recipe.Title)
	}
	if len(recipe.Ingredients) != 1 || len(recipe.Ingredients[0].Items) != 1 || recipe.Ingredients[0].Items[0] != "tomatoes" {
		t.Fatalf("Ingredients = %#v, want normalized tomato ingredient", recipe.Ingredients)
	}
	if len(recipe.Instructions) != 1 || recipe.Instructions[0] != "Simmer." {
		t.Fatalf("Instructions = %#v, want normalized simmer instruction", recipe.Instructions)
	}
}

func TestLocalExtractorParsesFencedJSON(t *testing.T) {
	responseContent := "Here is the normalized recipe JSON:\n\n```json\n{\"title\":\"Soup\",\"ingredients\":[{\"group\":\"\",\"items\":[\"salt\"]}],\"instructions\":[\"Stir.\"],\"times\":{\"prep\":\"PT5M\"},\"linked_recipe_urls\":[]}\n```"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]string{
						"content": responseContent,
					},
				},
			},
		})
	}))
	defer srv.Close()

	ext := NewLocal(LocalConfig{
		Model:   "local-model",
		BaseURL: srv.URL,
	})

	recipe, err := ext.NormalizeRecipe(context.Background(), Input{})
	if err != nil {
		t.Fatalf("NormalizeRecipe() error = %v", err)
	}
	if recipe.Title != "Soup" {
		t.Fatalf("Title = %q, want Soup", recipe.Title)
	}
	if recipe.Times["Prep Time"] != "5 minutes" {
		t.Fatalf("Times = %#v, want Prep Time 5 minutes", recipe.Times)
	}
}

func TestLocalExtractorReconcilesIncompleteModelOutputWithJSONLD(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "{\"title\":\"French Toast\",\"ingredients\":[{\"group\":\"\",\"items\":[\"4 large eggs\",\"2/3 cup milk\"]}],\"instructions\":[\"Preheat griddle.\"],\"yield\":\"8 servings\",\"times\":{\"Total Time\":\"15 minutes\"},\"linked_recipe_urls\":[]}"
				}
			}]
		}`))
	}))
	defer srv.Close()

	ext := NewLocal(LocalConfig{
		Model:   "local-model",
		BaseURL: srv.URL,
	})

	recipe, err := ext.NormalizeRecipe(context.Background(), Input{
		JSONLD: []string{`{
			"@type": "Recipe",
			"name": "French Toast",
			"recipeYield": ["8"],
			"prepTime": "PT5M",
			"cookTime": "PT10M",
			"totalTime": "PT15M",
			"recipeIngredient": [
				"4 large eggs",
				"2/3 cup milk",
				"8 thick slices bread"
			],
			"recipeInstructions": [
				{"@type":"HowToStep","text":"Preheat griddle."},
				{"@type":"HowToStep","text":"Dip bread."},
				{"@type":"HowToStep","text":"Serve warm."}
			]
		}`},
	})
	if err != nil {
		t.Fatalf("NormalizeRecipe() error = %v", err)
	}

	if countIngredientItems(recipe.Ingredients) != 3 {
		t.Fatalf("ingredient count = %d, want 3; ingredients=%#v", countIngredientItems(recipe.Ingredients), recipe.Ingredients)
	}
	if len(recipe.Instructions) != 3 {
		t.Fatalf("instruction count = %d, want 3; instructions=%#v", len(recipe.Instructions), recipe.Instructions)
	}
	if recipe.Times["Prep Time"] != "5 minutes" || recipe.Times["Cook Time"] != "10 minutes" || recipe.Times["Total Time"] != "15 minutes" {
		t.Fatalf("Times = %#v, want prep/cook/total from JSON-LD", recipe.Times)
	}
}

func TestLocalExtractorPrefersStructuredInstructionsWhenModelSplitsSteps(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "{\"title\":\"French Toast\",\"ingredients\":[{\"group\":\"\",\"items\":[\"4 large eggs\"]}],\"instructions\":[\"Preheat griddle.\",\"In a blender, add the eggs, milk, flour, sugar, salt, cinnamon, and vanilla. Blend until smooth.\",\"If you prefer whisking by hand, start by mixing the flour and eggs together in a shallow dish, then whisk in the rest of the ingredients until combined.\",\"Dip bread.\",\"Cook until golden.\"],\"yield\":\"8 servings\",\"times\":{},\"linked_recipe_urls\":[]}"
				}
			}]
		}`))
	}))
	defer srv.Close()

	ext := NewLocal(LocalConfig{
		Model:   "local-model",
		BaseURL: srv.URL,
	})

	recipe, err := ext.NormalizeRecipe(context.Background(), Input{
		JSONLD: []string{`{
			"@type": "Recipe",
			"name": "French Toast",
			"recipeIngredient": ["4 large eggs"],
			"recipeInstructions": [
				{"@type":"HowToStep","text":"Preheat griddle."},
				{"@type":"HowToStep","text":"In a blender, add the eggs, milk, flour, sugar, salt, cinnamon, and vanilla. Blend until smooth. If you prefer whisking by hand, start by mixing the flour and eggs together in a shallow dish, then whisk in the rest of the ingredients until combined."},
				{"@type":"HowToStep","text":"Dip bread."},
				{"@type":"HowToStep","text":"Cook until golden."},
				{"@type":"HowToStep","text":"Remove to a plate. Serve warm with syrup and a sprinkle of powdered sugar."}
			]
		}`},
	})
	if err != nil {
		t.Fatalf("NormalizeRecipe() error = %v", err)
	}

	wantFinalStep := "Remove to a plate. Serve warm with syrup and a sprinkle of powdered sugar."
	if len(recipe.Instructions) != 5 {
		t.Fatalf("instruction count = %d, want 5; instructions=%#v", len(recipe.Instructions), recipe.Instructions)
	}
	if recipe.Instructions[1] == "In a blender, add the eggs, milk, flour, sugar, salt, cinnamon, and vanilla. Blend until smooth." {
		t.Fatalf("model split instruction was preserved; instructions=%#v", recipe.Instructions)
	}
	if recipe.Instructions[4] != wantFinalStep {
		t.Fatalf("final instruction = %q, want %q", recipe.Instructions[4], wantFinalStep)
	}
}

func TestLocalExtractorSendsOptionalAPIKey(t *testing.T) {
	var gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "{\"title\":\"Soup\",\"ingredients\":[{\"group\":\"\",\"items\":[\"salt\"]}],\"instructions\":[\"Stir.\"],\"times\":{},\"linked_recipe_urls\":[]}"
				}
			}]
		}`))
	}))
	defer srv.Close()

	ext := NewLocal(LocalConfig{
		APIKey:  "local-key",
		Model:   "local-model",
		BaseURL: srv.URL,
	})

	if _, err := ext.NormalizeRecipe(context.Background(), Input{}); err != nil {
		t.Fatalf("NormalizeRecipe() error = %v", err)
	}
	if gotAuth != "Bearer local-key" {
		t.Fatalf("Authorization header = %q, want Bearer local-key", gotAuth)
	}
}

func TestLocalExtractorErrors(t *testing.T) {
	tests := []struct {
		name     string
		response string
		status   int
		wantErr  string
	}{
		{
			name:     "non 2xx",
			response: `nope`,
			status:   http.StatusBadGateway,
			wantErr:  "local llm request failed",
		},
		{
			name:     "malformed model json",
			response: `{"choices":[{"message":{"content":"not json"}}]}`,
			status:   http.StatusOK,
			wantErr:  "failed to parse model json",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.response))
			}))
			defer srv.Close()

			ext := NewLocal(LocalConfig{
				Model:   "local-model",
				BaseURL: srv.URL,
			})

			_, err := ext.NormalizeRecipe(context.Background(), Input{})
			if err == nil {
				t.Fatalf("NormalizeRecipe() error = nil, want containing %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("NormalizeRecipe() error = %q, want containing %q", err.Error(), tc.wantErr)
			}
		})
	}
}
