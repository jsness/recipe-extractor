package extractor

import (
	"fmt"
	"strings"
)

const systemPrompt = `You normalize recipe content into strict JSON. Return only valid JSON with keys: title, ingredients, instructions, yield, times, notes, linked_recipe_urls. The ingredients value must be an array of objects, each with a "group" key (the sub-recipe or section name, e.g. "Bacon Jam", or empty string when there are no distinct sections) and an "items" key (array of ingredient strings). Also include a "linked_recipe_urls" key: an array of URLs (from [url] annotations in the text) that appear to be ingredient references to other recipes on external pages. Only include URLs that are clearly links to full recipes used as a component of this recipe. Omit navigation links, ads, or unrelated content.`

func buildPrompt(input Input) string {
	jsonLD := "none"
	if len(input.JSONLD) > 0 {
		jsonLD = strings.Join(input.JSONLD, "\n\n")
		if len(jsonLD) > 50000 {
			jsonLD = jsonLD[:50000]
		}
	}

	text := input.Text
	if len(text) > 30000 {
		text = text[:30000]
	}

	links := "none"
	if len(input.Links) > 0 {
		capped := input.Links
		if len(capped) > 100 {
			capped = capped[:100]
		}
		links = strings.Join(capped, "\n")
	}

	return fmt.Sprintf(
		"Extract a single recipe from this webpage and normalize fields. If unknown, use null for yield/notes and empty object for times. Instructions must be an array of strings. Ingredients must be an array of group objects—each with \"group\" (the section or sub-recipe name, or empty string if there is only one group) and \"items\" (array of ingredient strings). Use separate groups when the recipe has distinct sub-recipes or sections (e.g. \"Sauce\", \"Crust\", \"Filling\"). Use the \"Page links\" section below to populate linked_recipe_urls: include only URLs whose link text matches an ingredient that is itself a full recipe used as a component of this recipe.\n\nSource URL: %s\n\nJSON-LD:\n%s\n\nPage text:\n%s\n\nPage links:\n%s",
		input.SourceURL,
		jsonLD,
		text,
		links,
	)
}

func normalizeRecipe(recipe *Recipe) {
	recipe.Title = strings.TrimSpace(recipe.Title)

	groups := make([]IngredientGroup, 0, len(recipe.Ingredients))
	for _, g := range recipe.Ingredients {
		items := compactStrings(g.Items)
		if len(items) > 0 {
			groups = append(groups, IngredientGroup{
				Group: strings.TrimSpace(g.Group),
				Items: items,
			})
		}
	}
	recipe.Ingredients = groups

	recipe.Instructions = compactStrings(recipe.Instructions)
	if recipe.Yield != nil {
		y := strings.TrimSpace(*recipe.Yield)
		if y == "" {
			recipe.Yield = nil
		} else {
			recipe.Yield = &y
		}
	}
	if recipe.Notes != nil {
		n := strings.TrimSpace(*recipe.Notes)
		if n == "" {
			recipe.Notes = nil
		} else {
			recipe.Notes = &n
		}
	}
	if recipe.Times == nil {
		recipe.Times = map[string]string{}
	}

	seen := make(map[string]struct{})
	clean := make([]string, 0, len(recipe.LinkedRecipeURLs))
	for _, u := range recipe.LinkedRecipeURLs {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if _, ok := seen[u]; !ok {
			seen[u] = struct{}{}
			clean = append(clean, u)
		}
	}
	recipe.LinkedRecipeURLs = clean
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func validateRecipe(recipe Recipe) error {
	if recipe.Title == "" {
		return fmt.Errorf("normalized recipe is missing title")
	}
	hasIngredients := false
	for _, g := range recipe.Ingredients {
		if len(g.Items) > 0 {
			hasIngredients = true
			break
		}
	}
	if !hasIngredients {
		return fmt.Errorf("normalized recipe is missing ingredients")
	}
	if len(recipe.Instructions) == 0 {
		return fmt.Errorf("normalized recipe is missing instructions")
	}
	return nil
}
