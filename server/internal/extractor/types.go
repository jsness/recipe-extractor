package extractor

import "context"

type Input struct {
	SourceURL string
	JSONLD    []string
	Text      string
}

type IngredientGroup struct {
	Group string   `json:"group"`
	Items []string `json:"items"`
}

type Recipe struct {
	Title        string            `json:"title"`
	Ingredients  []IngredientGroup `json:"ingredients"`
	Instructions []string          `json:"instructions"`
	Yield        *string           `json:"yield,omitempty"`
	Times        map[string]string `json:"times,omitempty"`
	Notes        *string           `json:"notes,omitempty"`
}

type Extractor interface {
	NormalizeRecipe(ctx context.Context, input Input) (Recipe, error)
}
