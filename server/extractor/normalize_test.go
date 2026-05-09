package extractor

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestNormalizeRecipe(t *testing.T) {
	yield := "  4 servings  "
	notes := "  Make ahead.  "

	recipe := Recipe{
		Title: "  Tomato Soup  ",
		Ingredients: []IngredientGroup{
			{Group: "  Soup  ", Items: []string{"  1 onion  ", "", "  2 tomatoes"}},
			{Group: "Empty", Items: []string{" ", "\t"}},
		},
		Instructions:     []string{"  Chop onion.  ", "", " Simmer. "},
		Yield:            &yield,
		Notes:            &notes,
		LinkedRecipeURLs: []string{" https://example.com/a ", "", "https://example.com/a", "https://example.com/b"},
	}

	normalizeRecipe(&recipe)

	wantYield := "4 servings"
	wantNotes := "Make ahead."
	want := Recipe{
		Title: "Tomato Soup",
		Ingredients: []IngredientGroup{
			{Group: "Soup", Items: []string{"1 onion", "2 tomatoes"}},
		},
		Instructions:     []string{"Chop onion.", "Simmer."},
		Yield:            &wantYield,
		Times:            map[string]string{},
		Notes:            &wantNotes,
		LinkedRecipeURLs: []string{"https://example.com/a", "https://example.com/b"},
	}

	if !reflect.DeepEqual(recipe, want) {
		t.Fatalf("normalizeRecipe() = %#v, want %#v", recipe, want)
	}
}

func TestNormalizeRecipeClearsBlankOptionalStrings(t *testing.T) {
	yield := " "
	notes := "\t"
	recipe := Recipe{Yield: &yield, Notes: &notes}

	normalizeRecipe(&recipe)

	if recipe.Yield != nil {
		t.Fatalf("Yield = %q, want nil", *recipe.Yield)
	}
	if recipe.Notes != nil {
		t.Fatalf("Notes = %q, want nil", *recipe.Notes)
	}
}

func TestValidateRecipe(t *testing.T) {
	valid := Recipe{
		Title:        "Soup",
		Ingredients:  []IngredientGroup{{Items: []string{"salt"}}},
		Instructions: []string{"Stir."},
	}

	tests := []struct {
		name    string
		recipe  Recipe
		wantErr string
	}{
		{
			name:   "valid recipe",
			recipe: valid,
		},
		{
			name: "missing title",
			recipe: Recipe{
				Ingredients:  valid.Ingredients,
				Instructions: valid.Instructions,
			},
			wantErr: "missing title",
		},
		{
			name: "missing ingredients",
			recipe: Recipe{
				Title:        valid.Title,
				Instructions: valid.Instructions,
			},
			wantErr: "missing ingredients",
		},
		{
			name: "empty ingredient groups",
			recipe: Recipe{
				Title:        valid.Title,
				Ingredients:  []IngredientGroup{{Group: "Sauce"}},
				Instructions: valid.Instructions,
			},
			wantErr: "missing ingredients",
		},
		{
			name: "missing instructions",
			recipe: Recipe{
				Title:       valid.Title,
				Ingredients: valid.Ingredients,
			},
			wantErr: "missing instructions",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRecipe(tc.recipe)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("validateRecipe() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validateRecipe() error = nil, want %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("validateRecipe() error = %q, want containing %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestBuildPromptCapsInputs(t *testing.T) {
	input := Input{
		SourceURL: "https://example.com/recipe",
		JSONLD:    []string{strings.Repeat("j", 50001)},
		Text:      strings.Repeat("t", 30001),
		Links:     make([]string, 101),
	}
	for i := range input.Links {
		input.Links[i] = "recipe-link-" + strconv.Itoa(i)
	}

	prompt := buildPrompt(input)

	if strings.Contains(prompt, strings.Repeat("j", 50001)) {
		t.Fatal("buildPrompt() did not cap JSON-LD content")
	}
	if strings.Contains(prompt, strings.Repeat("t", 30001)) {
		t.Fatal("buildPrompt() did not cap page text")
	}
	if !strings.Contains(prompt, "recipe-link-99") {
		t.Fatal("buildPrompt() did not include the 100th link")
	}
	if strings.Contains(prompt, "recipe-link-100") {
		t.Fatal("buildPrompt() included a link beyond the first 100")
	}
}
