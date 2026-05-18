package api

import (
	"reflect"
	"testing"
	"time"

	"github.com/jsness/recipe-extractor/server/scraper"
	"github.com/jsness/recipe-extractor/server/store"
)

func TestNewRecipeExtractionResponse(t *testing.T) {
	blockedMessage := (&scraper.FetchError{Kind: scraper.FetchErrorKindBlockedAccess, StatusCode: 403}).Error()
	recipeID := "recipe-1"

	tests := []struct {
		name       string
		extraction store.RecipeExtraction
		want       getRecipeExtractionResponse
	}{
		{
			name: "failed blocked source can try archive",
			extraction: store.RecipeExtraction{
				ID:           "extract-1",
				SourceURL:    "https://example.com/recipe",
				Status:       "failed",
				RecipeID:     &recipeID,
				ErrorMessage: &blockedMessage,
			},
			want: getRecipeExtractionResponse{
				ID:                   "extract-1",
				SourceURL:            "https://example.com/recipe",
				Status:               "failed",
				RecipeID:             &recipeID,
				ErrorMessage:         &blockedMessage,
				CanTryArchivedSource: true,
			},
		},
		{
			name: "archive url cannot try archive again",
			extraction: store.RecipeExtraction{
				ID:           "extract-2",
				SourceURL:    "https://web.archive.org/web/20200101/https://example.com/recipe",
				Status:       "failed",
				ErrorMessage: &blockedMessage,
			},
			want: getRecipeExtractionResponse{
				ID:           "extract-2",
				SourceURL:    "https://web.archive.org/web/20200101/https://example.com/recipe",
				Status:       "failed",
				ErrorMessage: &blockedMessage,
			},
		},
		{
			name: "non failed status cannot try archive",
			extraction: store.RecipeExtraction{
				ID:           "extract-3",
				SourceURL:    "https://example.com/recipe",
				Status:       "queued",
				ErrorMessage: &blockedMessage,
			},
			want: getRecipeExtractionResponse{
				ID:           "extract-3",
				SourceURL:    "https://example.com/recipe",
				Status:       "queued",
				ErrorMessage: &blockedMessage,
			},
		},
		{
			name: "unrelated failure cannot try archive",
			extraction: store.RecipeExtraction{
				ID:           "extract-4",
				SourceURL:    "https://example.com/recipe",
				Status:       "failed",
				ErrorMessage: stringPtr("unexpected status code: 500"),
			},
			want: getRecipeExtractionResponse{
				ID:           "extract-4",
				SourceURL:    "https://example.com/recipe",
				Status:       "failed",
				ErrorMessage: stringPtr("unexpected status code: 500"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := newRecipeExtractionResponse(tc.extraction)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("newRecipeExtractionResponse() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestNewRecipeResponse(t *testing.T) {
	createdAt := time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC)
	yield := "2 servings"
	notes := "Warm"

	tests := []struct {
		name    string
		recipe  store.Recipe
		related []store.RelatedRecipe
		want    recipeResponse
	}{
		{
			name: "recipe without related recipes",
			recipe: store.Recipe{
				ID:           "recipe-1",
				Title:        "Soup",
				Ingredients:  []store.IngredientGroup{{Items: []string{"salt"}}},
				Instructions: []string{"Stir."},
				Yield:        &yield,
				Times:        map[string]string{"Cook Time": "20 minutes"},
				Notes:        &notes,
				SourceURL:    "https://example.com/soup",
				CreatedAt:    createdAt,
			},
			want: recipeResponse{
				ID:           "recipe-1",
				Title:        "Soup",
				Ingredients:  []store.IngredientGroup{{Items: []string{"salt"}}},
				Instructions: []string{"Stir."},
				Yield:        &yield,
				Times:        map[string]string{"Cook Time": "20 minutes"},
				Notes:        &notes,
				SourceURL:    "https://example.com/soup",
				CreatedAt:    createdAt,
			},
		},
		{
			name: "recipe with related recipes",
			recipe: store.Recipe{
				ID:           "recipe-1",
				Title:        "Soup",
				Ingredients:  []store.IngredientGroup{{Items: []string{"salt"}}},
				Instructions: []string{"Stir."},
				SourceURL:    "https://example.com/soup",
				CreatedAt:    createdAt,
			},
			related: []store.RelatedRecipe{
				{ID: "recipe-2", Title: "Stock", Relationship: "component"},
			},
			want: recipeResponse{
				ID:           "recipe-1",
				Title:        "Soup",
				Ingredients:  []store.IngredientGroup{{Items: []string{"salt"}}},
				Instructions: []string{"Stir."},
				SourceURL:    "https://example.com/soup",
				CreatedAt:    createdAt,
				RelatedRecipes: []relatedRecipeResponse{
					{ID: "recipe-2", Title: "Stock", Relationship: "component"},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := newRecipeResponse(tc.recipe, tc.related)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("newRecipeResponse() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
