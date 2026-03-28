package store

import "time"

type Profile struct {
	ID        string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type RecipeExtraction struct {
	ID             string
	ProfileID      string
	SourceURL      string
	Status         string
	RecipeID       *string
	ErrorMessage   *string
	ParentRecipeID *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type IngredientGroup struct {
	Group string   `json:"group"`
	Items []string `json:"items"`
}

type Recipe struct {
	ID               string
	Title            string
	Ingredients      []IngredientGroup
	Instructions     []string
	Yield            *string
	Times            map[string]string
	Notes            *string
	SourceURL        string
	LinkedRecipeURLs []string
	CreatedAt        time.Time
}

type RecipeInput struct {
	Title            string
	Ingredients      []IngredientGroup
	Instructions     []string
	Yield            *string
	Times            map[string]string
	Notes            *string
	SourceURL        string
	LinkedRecipeURLs []string
}

type RecipeSummary struct {
	ID    string
	Title string
}

type RelatedRecipe struct {
	ID           string
	Title        string
	Relationship string // "component" or "used_in"
}
