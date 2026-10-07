package extractor

import (
	"context"
	"io"
	"log"
	"testing"
)

func TestJSONLDCollectionsSkipIncompleteRecipes(t *testing.T) {
	valid := `{"@type":"Recipe","name":"Soup","recipeIngredient":["water","salt"],"recipeInstructions":["Mix."]}`
	for name, raw := range map[string]string{
		"top level array":        `[{"@type":"BreadcrumbList"},{"@type":"Recipe","name":"Incomplete"},` + valid + `]`,
		"graph":                  `{"@graph":[{"@type":"Recipe","name":"Incomplete"},` + valid + `]}`,
		"array containing graph": `[null,{"@graph":[` + valid + `]}]`,
	} {
		t.Run(name, func(t *testing.T) {
			ext := NewJSONLDExtractor(nil, log.New(io.Discard, "", 0))
			recipe, err := ext.NormalizeRecipe(context.Background(), Input{JSONLD: []string{raw}})
			if err != nil {
				t.Fatal(err)
			}
			if recipe.Title != "Soup" || len(recipe.Ingredients[0].Items) != 2 || len(recipe.Instructions) != 1 {
				t.Fatalf("incomplete recipe: %#v", recipe)
			}
		})
	}
}
