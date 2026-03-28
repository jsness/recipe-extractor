package api

import (
	"time"

	"recipe-extractor/server/store"
)

type createRecipeRequest struct {
	URL string `json:"url"`
}

type createProfileRequest struct {
	Name string `json:"name"`
}

type createRecipeResponse struct {
	ExtractionID string `json:"extraction_id"`
	Status       string `json:"status"`
}

type profileResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type recipeSummaryResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type relatedRecipeResponse struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Relationship string `json:"relationship"` // "component" | "used_in"
}

type recipeResponse struct {
	ID             string                  `json:"id"`
	Title          string                  `json:"title"`
	Ingredients    []store.IngredientGroup `json:"ingredients"`
	Instructions   []string                `json:"instructions"`
	Yield          *string                 `json:"yield,omitempty"`
	Times          map[string]string       `json:"times,omitempty"`
	Notes          *string                 `json:"notes,omitempty"`
	SourceURL      string                  `json:"source_url"`
	CreatedAt      time.Time               `json:"created_at"`
	RelatedRecipes []relatedRecipeResponse `json:"related_recipes,omitempty"`
}

type getRecipeExtractionResponse struct {
	ID           string  `json:"id"`
	SourceURL    string  `json:"source_url"`
	Status       string  `json:"status"`
	RecipeID     *string `json:"recipe_id,omitempty"`
	ErrorMessage *string `json:"error_message,omitempty"`
}

func newRecipeResponse(recipe store.Recipe, related []store.RelatedRecipe) recipeResponse {
	resp := recipeResponse{
		ID:           recipe.ID,
		Title:        recipe.Title,
		Ingredients:  recipe.Ingredients,
		Instructions: recipe.Instructions,
		Yield:        recipe.Yield,
		Times:        recipe.Times,
		Notes:        recipe.Notes,
		SourceURL:    recipe.SourceURL,
		CreatedAt:    recipe.CreatedAt,
	}
	for _, rel := range related {
		resp.RelatedRecipes = append(resp.RelatedRecipes, relatedRecipeResponse{
			ID:           rel.ID,
			Title:        rel.Title,
			Relationship: rel.Relationship,
		})
	}
	return resp
}

func newRecipeExtractionResponse(extraction store.RecipeExtraction) getRecipeExtractionResponse {
	return getRecipeExtractionResponse{
		ID:           extraction.ID,
		SourceURL:    extraction.SourceURL,
		Status:       extraction.Status,
		RecipeID:     extraction.RecipeID,
		ErrorMessage: extraction.ErrorMessage,
	}
}

func newProfileResponse(profile store.Profile) profileResponse {
	return profileResponse{
		ID:        profile.ID,
		Name:      profile.Name,
		CreatedAt: profile.CreatedAt,
	}
}
