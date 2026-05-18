package core

import (
	"errors"
	"testing"

	"github.com/jsness/recipe-extractor/server/store"
)

func TestCheckExistingExtraction(t *testing.T) {
	recipeID := "recipe-id"

	tests := []struct {
		name     string
		existing *store.RecipeExtraction
		wantErr  error
	}{
		{
			name: "no existing extraction",
		},
		{
			name: "done extraction with recipe blocks duplicate",
			existing: &store.RecipeExtraction{
				Status:   "done",
				RecipeID: &recipeID,
			},
			wantErr: ErrRecipeAlreadyExtracted,
		},
		{
			name: "done extraction without recipe can be recreated",
			existing: &store.RecipeExtraction{
				Status: "done",
			},
		},
		{
			name: "failed extraction can be retried",
			existing: &store.RecipeExtraction{
				Status: "failed",
			},
		},
		{
			name: "queued extraction blocks duplicate",
			existing: &store.RecipeExtraction{
				Status: "queued",
			},
			wantErr: ErrRecipeExtractionInProgress,
		},
		{
			name: "extracting extraction blocks duplicate",
			existing: &store.RecipeExtraction{
				Status: "extracting",
			},
			wantErr: ErrRecipeExtractionInProgress,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkExistingExtraction(tc.existing)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("checkExistingExtraction() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
