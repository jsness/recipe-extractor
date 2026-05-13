package extractor

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "minutes", in: "PT45M", want: "45 minutes"},
		{name: "singular minute", in: "PT1M", want: "1 minute"},
		{name: "hours and minutes", in: "PT1H30M", want: "1 hour 30 minutes"},
		{name: "days hours minutes", in: "P2DT3H4M", want: "2 days 3 hours 4 minutes"},
		{name: "zero duration passes through", in: "PT", want: "PT"},
		{name: "non iso passes through", in: "about 20 minutes", want: "about 20 minutes"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseDuration(tc.in); got != tc.want {
				t.Fatalf("parseDuration(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseInstructions(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    []string
		wantErr string
	}{
		{
			name: "string array",
			raw:  `["Mix.","Bake."]`,
			want: []string{"Mix.", "Bake."},
		},
		{
			name: "how to steps",
			raw:  `[{"@type":"HowToStep","text":"Mix."},{"@type":"HowToStep","text":"Bake."}]`,
			want: []string{"Mix.", "Bake."},
		},
		{
			name: "how to section",
			raw:  `[{"@type":"HowToSection","itemListElement":[{"text":"Mix."},{"text":"Bake."}]}]`,
			want: []string{"Mix.", "Bake."},
		},
		{
			name:    "objects without steps",
			raw:     `[{"@type":"Thing","text":"Skip."}]`,
			wantErr: "no steps found",
		},
		{
			name:    "invalid shape",
			raw:     `{"text":"Mix."}`,
			wantErr: "parse instructions",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseInstructions(json.RawMessage(tc.raw))
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("parseInstructions() error = %v, want nil", err)
				}
				if !reflect.DeepEqual(got, tc.want) {
					t.Fatalf("parseInstructions() = %#v, want %#v", got, tc.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("parseInstructions() error = nil, want %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("parseInstructions() error = %q, want containing %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestTryParseJSONLD(t *testing.T) {
	tests := []struct {
		name             string
		raw              string
		ingredientGroups []IngredientGroup
		want             Recipe
		wantErr          string
	}{
		{
			name: "root recipe",
			raw: `{
				"@type": "Recipe",
				"name": " Soup ",
				"recipeIngredient": [" salt ", "water"],
				"recipeInstructions": [" Mix. ", "Serve."],
				"recipeYield": " 2 bowls ",
				"prepTime": "PT10M",
				"cookTime": "PT1H",
				"totalTime": "about 70 minutes",
				"description": " Warm. "
			}`,
			want: Recipe{
				Title:        "Soup",
				Ingredients:  []IngredientGroup{{Items: []string{"salt", "water"}}},
				Instructions: []string{"Mix.", "Serve."},
				Yield:        stringPtr("2 bowls"),
				Times: map[string]string{
					"Prep Time":  "10 minutes",
					"Cook Time":  "1 hour",
					"Total Time": "about 70 minutes",
				},
				Notes:            stringPtr("Warm."),
				LinkedRecipeURLs: []string{},
			},
		},
		{
			name: "graph recipe with numeric yield",
			raw: `{
				"@graph": [
					{"@type": "WebPage", "name": "Page"},
					{
						"@type": ["Thing", "Recipe"],
						"name": "Bread",
						"recipeIngredient": ["flour"],
						"recipeInstructions": [{"@type":"HowToStep","text":"Knead."}],
						"recipeYield": 1
					}
				]
			}`,
			want: Recipe{
				Title:            "Bread",
				Ingredients:      []IngredientGroup{{Items: []string{"flour"}}},
				Instructions:     []string{"Knead."},
				Yield:            stringPtr("1 serving"),
				Times:            map[string]string{},
				LinkedRecipeURLs: []string{},
			},
		},
		{
			name: "array recipe yield",
			raw: `{
					"@type": "Recipe",
					"name": "Toast",
					"recipeIngredient": ["bread"],
					"recipeInstructions": ["Toast."],
					"recipeYield": ["8"]
				}`,
			want: Recipe{
				Title:            "Toast",
				Ingredients:      []IngredientGroup{{Items: []string{"bread"}}},
				Instructions:     []string{"Toast."},
				Yield:            stringPtr("8 servings"),
				Times:            map[string]string{},
				LinkedRecipeURLs: []string{},
			},
		},
		{
			name: "merges matching html ingredient groups",
			raw: `{
				"@type": "Recipe",
				"name": "Layer Cake",
				"recipeIngredient": ["1 cup flour", "2 eggs", "1 cup sugar"],
				"recipeInstructions": ["Mix."]
			}`,
			ingredientGroups: []IngredientGroup{
				{Group: "Cake", Items: []string{"1 cup flour", "2 eggs"}},
				{Group: "Frosting", Items: []string{"1 cup sugar"}},
			},
			want: Recipe{
				Title: "Layer Cake",
				Ingredients: []IngredientGroup{
					{Group: "Cake", Items: []string{"1 cup flour", "2 eggs"}},
					{Group: "Frosting", Items: []string{"1 cup sugar"}},
				},
				Instructions:     []string{"Mix."},
				Times:            map[string]string{},
				LinkedRecipeURLs: []string{},
			},
		},
		{
			name:    "invalid json",
			raw:     `{`,
			wantErr: "unmarshal",
		},
		{
			name:    "no recipe node",
			raw:     `{"@type":"WebPage","name":"Page"}`,
			wantErr: "no Recipe node",
		},
		{
			name:    "validation failure",
			raw:     `{"@type":"Recipe","name":"Untitled"}`,
			wantErr: "missing ingredients",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tryParseJSONLD(tc.raw, tc.ingredientGroups)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("tryParseJSONLD() error = %v, want nil", err)
				}
				if !reflect.DeepEqual(got, tc.want) {
					t.Fatalf("tryParseJSONLD() = %#v, want %#v", got, tc.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("tryParseJSONLD() error = nil, want %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("tryParseJSONLD() error = %q, want containing %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestMergeIngredientGroups(t *testing.T) {
	tests := []struct {
		name   string
		jsonLD []IngredientGroup
		html   []IngredientGroup
		want   []IngredientGroup
		wantOK bool
	}{
		{
			name:   "matching groups",
			jsonLD: []IngredientGroup{{Items: []string{"1 cup Flour", "2 eggs", "pinch salt"}}},
			html: []IngredientGroup{
				{Group: "Cake", Items: []string{"1 cup flour", "2 eggs"}},
				{Group: "Finish", Items: []string{"pinch salt"}},
			},
			want: []IngredientGroup{
				{Group: "Cake", Items: []string{"1 cup Flour", "2 eggs"}},
				{Group: "Finish", Items: []string{"pinch salt"}},
			},
			wantOK: true,
		},
		{
			name:   "normalizes punctuation and spaces before matching",
			jsonLD: []IngredientGroup{{Items: []string{"1 cup flour, sifted"}}},
			html:   []IngredientGroup{{Group: "Cake", Items: []string{" 1 cup flour sifted "}}},
			want:   []IngredientGroup{{Group: "Cake", Items: []string{"1 cup flour, sifted"}}},
			wantOK: true,
		},
		{
			name:   "mismatched counts",
			jsonLD: []IngredientGroup{{Items: []string{"flour"}}},
			html:   []IngredientGroup{{Group: "Cake", Items: []string{"flour", "eggs"}}},
		},
		{
			name:   "mismatched ingredients",
			jsonLD: []IngredientGroup{{Items: []string{"flour"}}},
			html:   []IngredientGroup{{Group: "Cake", Items: []string{"sugar"}}},
		},
		{
			name:   "jsonld already grouped",
			jsonLD: []IngredientGroup{{Group: "Cake", Items: []string{"flour"}}},
			html:   []IngredientGroup{{Group: "Cake", Items: []string{"flour"}}},
		},
		{
			name:   "no html groups",
			jsonLD: []IngredientGroup{{Items: []string{"flour"}}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := mergeIngredientGroups(tc.jsonLD, tc.html)
			if ok != tc.wantOK {
				t.Fatalf("mergeIngredientGroups() ok = %v, want %v", ok, tc.wantOK)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("mergeIngredientGroups() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
