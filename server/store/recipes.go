package store

import (
	"context"
	"database/sql"
)

func (s *Store) GetRecipeByID(ctx context.Context, id string) (Recipe, error) {
	const q = `
		SELECT id::text, title, ingredients, instructions, yield, times, notes, source_url, linked_recipe_urls, created_at
		FROM recipes
		WHERE id = $1
	`

	var r Recipe
	var ingredientsRaw, instructionsRaw, timesRaw, linkedURLsRaw []byte
	var yield, notes sql.NullString

	err := s.Pool.QueryRow(ctx, q, id).Scan(
		&r.ID, &r.Title, &ingredientsRaw, &instructionsRaw,
		&yield, &timesRaw, &notes, &r.SourceURL, &linkedURLsRaw, &r.CreatedAt,
	)
	if err != nil {
		return Recipe{}, err
	}

	if err := decodeRecipeRow(&r, ingredientsRaw, instructionsRaw, timesRaw, linkedURLsRaw, yield, notes); err != nil {
		return Recipe{}, err
	}
	return r, nil
}

func (s *Store) ListRecipes(ctx context.Context) ([]RecipeSummary, error) {
	const q = `
		SELECT id::text, title
		FROM recipes
		ORDER BY created_at DESC
	`

	rows, err := s.Pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipes []RecipeSummary
	for rows.Next() {
		var r RecipeSummary
		if err := rows.Scan(&r.ID, &r.Title); err != nil {
			return nil, err
		}
		recipes = append(recipes, r)
	}
	return recipes, rows.Err()
}

func (s *Store) UpsertRecipe(ctx context.Context, input RecipeInput) (string, error) {
	ingredientsJSON, instructionsJSON, timesJSON, linkedURLsJSON, err := marshalRecipeInput(input)
	if err != nil {
		return "", err
	}

	const q = `
		INSERT INTO recipes (title, ingredients, instructions, yield, times, notes, source_url, linked_recipe_urls)
		VALUES ($1, $2::jsonb, $3::jsonb, $4, $5::jsonb, $6, $7, $8::jsonb)
		ON CONFLICT (source_url) DO UPDATE
		SET
			title = EXCLUDED.title,
			ingredients = EXCLUDED.ingredients,
			instructions = EXCLUDED.instructions,
			yield = EXCLUDED.yield,
			times = EXCLUDED.times,
			notes = EXCLUDED.notes,
			linked_recipe_urls = EXCLUDED.linked_recipe_urls
		RETURNING id::text
	`

	var recipeID string
	err = s.Pool.QueryRow(
		ctx,
		q,
		input.Title,
		ingredientsJSON,
		instructionsJSON,
		input.Yield,
		timesJSON,
		input.Notes,
		input.SourceURL,
		linkedURLsJSON,
	).Scan(&recipeID)
	if err != nil {
		return "", err
	}
	return recipeID, nil
}
