package extractor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
)

type jsonldExtractor struct {
	fallback Extractor
	logger   *log.Logger
}

func NewJSONLDExtractor(fallback Extractor, logger *log.Logger) Extractor {
	return &jsonldExtractor{fallback: fallback, logger: logger}
}

func (e *jsonldExtractor) NormalizeRecipe(ctx context.Context, input Input) (Recipe, error) {
	var lastErr error
	for _, raw := range input.JSONLD {
		recipe, err := tryParseJSONLD(raw, input.Ingredients)
		if err == nil {
			e.logger.Printf("json-ld parse succeeded")
			return recipe, nil
		}
		lastErr = err
	}
	reason := "no JSON-LD blocks"
	if lastErr != nil {
		reason = lastErr.Error()
	}
	if e.fallback != nil {
		e.logger.Printf("json-ld parse failed, falling back to LLM: %s", reason)
		return e.fallback.NormalizeRecipe(ctx, input)
	}
	e.logger.Printf("json-ld parse failed, no LLM configured: %s", reason)
	return Recipe{}, fmt.Errorf("could not extract recipe from structured data: %s", reason)
}

func tryParseJSONLD(raw string, ingredientGroups []IngredientGroup) (Recipe, error) {
	var node map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &node); err != nil {
		return Recipe{}, fmt.Errorf("unmarshal: %w", err)
	}

	// Check for @graph array
	if graphRaw, ok := node["@graph"]; ok {
		var graph []map[string]json.RawMessage
		if err := json.Unmarshal(graphRaw, &graph); err == nil {
			for _, item := range graph {
				if isRecipeType(item) {
					return mapToRecipe(item, ingredientGroups)
				}
			}
		}
	}

	// Check if root node is a Recipe
	if isRecipeType(node) {
		return mapToRecipe(node, ingredientGroups)
	}

	return Recipe{}, fmt.Errorf("no Recipe node found")
}

func isRecipeType(node map[string]json.RawMessage) bool {
	typeRaw, ok := node["@type"]
	if !ok {
		return false
	}
	var typeStr string
	if err := json.Unmarshal(typeRaw, &typeStr); err == nil {
		return typeStr == "Recipe"
	}
	var typeArr []string
	if err := json.Unmarshal(typeRaw, &typeArr); err == nil {
		for _, t := range typeArr {
			if t == "Recipe" {
				return true
			}
		}
	}
	return false
}

func mapToRecipe(node map[string]json.RawMessage, ingredientGroups []IngredientGroup) (Recipe, error) {
	var recipe Recipe

	if v, ok := node["name"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			recipe.Title = s
		}
	}

	if v, ok := node["recipeIngredient"]; ok {
		var items []string
		if err := json.Unmarshal(v, &items); err == nil && len(items) > 0 {
			recipe.Ingredients = []IngredientGroup{{Group: "", Items: items}}
		}
	}

	if v, ok := node["recipeInstructions"]; ok {
		if steps, err := parseInstructions(v); err == nil {
			recipe.Instructions = steps
		}
	}

	if v, ok := node["recipeYield"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err == nil && s != "" {
			recipe.Yield = &s
		} else {
			var n json.Number
			if err := json.Unmarshal(v, &n); err == nil {
				ns := n.String()
				recipe.Yield = &ns
			}
		}
	}

	times := map[string]string{}
	for _, f := range []struct{ key, label string }{
		{"prepTime", "Prep Time"},
		{"cookTime", "Cook Time"},
		{"totalTime", "Total Time"},
	} {
		if v, ok := node[f.key]; ok {
			var s string
			if err := json.Unmarshal(v, &s); err == nil && s != "" {
				if parsed := parseDuration(s); parsed != "" {
					times[f.label] = parsed
				}
			}
		}
	}
	if len(times) > 0 {
		recipe.Times = times
	}

	if v, ok := node["description"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err == nil && s != "" {
			recipe.Notes = &s
		}
	}

	if merged, ok := mergeIngredientGroups(recipe.Ingredients, ingredientGroups); ok {
		recipe.Ingredients = merged
	}

	normalizeRecipe(&recipe)
	if err := validateRecipe(recipe); err != nil {
		return Recipe{}, err
	}
	return recipe, nil
}

func parseInstructions(raw json.RawMessage) ([]string, error) {
	var strSlice []string
	if err := json.Unmarshal(raw, &strSlice); err == nil {
		return strSlice, nil
	}

	var items []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("parse instructions: %w", err)
	}

	var steps []string
	for _, item := range items {
		typeRaw, ok := item["@type"]
		if !ok {
			continue
		}
		var typeStr string
		json.Unmarshal(typeRaw, &typeStr) //nolint:errcheck

		switch typeStr {
		case "HowToStep":
			if textRaw, ok := item["text"]; ok {
				var text string
				if err := json.Unmarshal(textRaw, &text); err == nil && text != "" {
					steps = append(steps, text)
				}
			}
		case "HowToSection":
			if listRaw, ok := item["itemListElement"]; ok {
				var subItems []map[string]json.RawMessage
				if err := json.Unmarshal(listRaw, &subItems); err == nil {
					for _, sub := range subItems {
						if textRaw, ok := sub["text"]; ok {
							var text string
							if err := json.Unmarshal(textRaw, &text); err == nil && text != "" {
								steps = append(steps, text)
							}
						}
					}
				}
			}
		}
	}

	if len(steps) == 0 {
		return nil, fmt.Errorf("no steps found in instruction objects")
	}
	return steps, nil
}

var isoDurationRe = regexp.MustCompile(`^P(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?)?$`)

func parseDuration(s string) string {
	m := isoDurationRe.FindStringSubmatch(s)
	if m == nil {
		return s
	}
	days, _ := strconv.Atoi(m[1])
	hours, _ := strconv.Atoi(m[2])
	minutes, _ := strconv.Atoi(m[3])

	var parts []string
	switch days {
	case 0:
	case 1:
		parts = append(parts, "1 day")
	default:
		parts = append(parts, fmt.Sprintf("%d days", days))
	}
	switch hours {
	case 0:
	case 1:
		parts = append(parts, "1 hour")
	default:
		parts = append(parts, fmt.Sprintf("%d hours", hours))
	}
	switch minutes {
	case 0:
	case 1:
		parts = append(parts, "1 minute")
	default:
		parts = append(parts, fmt.Sprintf("%d minutes", minutes))
	}

	if len(parts) == 0 {
		return s
	}
	return strings.Join(parts, " ")
}

func mergeIngredientGroups(jsonLDGroups []IngredientGroup, htmlGroups []IngredientGroup) ([]IngredientGroup, bool) {
	if len(jsonLDGroups) != 1 || len(htmlGroups) == 0 {
		return nil, false
	}

	jsonLDItems := jsonLDGroups[0].Items
	if len(jsonLDItems) == 0 || len(jsonLDGroups[0].Group) != 0 {
		return nil, false
	}

	htmlItems := flattenIngredientItems(htmlGroups)
	if len(htmlItems) != len(jsonLDItems) {
		return nil, false
	}

	for i := range jsonLDItems {
		if normalizeIngredientForMatch(jsonLDItems[i]) != normalizeIngredientForMatch(htmlItems[i]) {
			return nil, false
		}
	}

	merged := make([]IngredientGroup, 0, len(htmlGroups))
	offset := 0
	for _, group := range htmlGroups {
		size := len(group.Items)
		if size == 0 {
			continue
		}
		merged = append(merged, IngredientGroup{
			Group: strings.TrimSpace(group.Group),
			Items: append([]string(nil), jsonLDItems[offset:offset+size]...),
		})
		offset += size
	}

	if offset != len(jsonLDItems) || len(merged) == 0 {
		return nil, false
	}
	return merged, true
}

func flattenIngredientItems(groups []IngredientGroup) []string {
	total := 0
	for _, group := range groups {
		total += len(group.Items)
	}

	items := make([]string, 0, total)
	for _, group := range groups {
		items = append(items, group.Items...)
	}
	return items
}

var ingredientNormalizer = strings.NewReplacer(
	"\u00a0", " ",
	"(", " ",
	")", " ",
	",", " ",
	":", " ",
	";", " ",
)

func normalizeIngredientForMatch(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = ingredientNormalizer.Replace(s)
	s = strings.Join(strings.Fields(s), " ")
	return s
}
