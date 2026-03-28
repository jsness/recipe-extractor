package store

import (
	"context"
	"database/sql"
)

func (s *Store) GetRecipeByID(ctx context.Context, profileID, id string) (Recipe, error) {
	const q = `
		SELECT id::text, title, ingredients, instructions, yield, times, notes, source_url, linked_recipe_urls, created_at
		FROM recipes
		WHERE id = $1 AND profile_id = $2
	`

	var r Recipe
	var ingredientsRaw, instructionsRaw, timesRaw, linkedURLsRaw []byte
	var yield, notes sql.NullString

	err := s.Pool.QueryRow(ctx, q, id, profileID).Scan(
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

func (s *Store) ListRecipes(ctx context.Context, profileID string) ([]RecipeSummary, error) {
	const q = `
		SELECT id::text, title
		FROM recipes
		WHERE profile_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.Pool.Query(ctx, q, profileID)
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

func (s *Store) DeleteRecipe(ctx context.Context, profileID, id string) (bool, error) {
	const q = `
		DELETE FROM recipes
		WHERE id = $1 AND profile_id = $2
	`

	result, err := s.Pool.Exec(ctx, q, id, profileID)
	if err != nil {
		return false, err
	}

	return result.RowsAffected() > 0, nil
}

func (s *Store) UpsertRecipe(ctx context.Context, profileID string, input RecipeInput) (string, error) {
	ingredientsJSON, instructionsJSON, timesJSON, linkedURLsJSON, err := marshalRecipeInput(input)
	if err != nil {
		return "", err
	}

	const q = `
		INSERT INTO recipes (profile_id, title, ingredients, instructions, yield, times, notes, source_url, linked_recipe_urls)
		VALUES ($1, $2, $3::jsonb, $4::jsonb, $5, $6::jsonb, $7, $8, $9::jsonb)
		ON CONFLICT (profile_id, source_url) DO UPDATE
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
		profileID,
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
