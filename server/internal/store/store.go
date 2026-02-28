package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{Pool: pool}
}

type RecipeExtraction struct {
	ID           string
	SourceURL    string
	Status       string
	RecipeID     *string
	ErrorMessage *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type IngredientGroup struct {
	Group string   `json:"group"`
	Items []string `json:"items"`
}

type Recipe struct {
	ID           string
	Title        string
	Ingredients  []IngredientGroup
	Instructions []string
	Yield        *string
	Times        map[string]string
	Notes        *string
	SourceURL    string
	CreatedAt    time.Time
}

type RecipeInput struct {
	Title        string
	Ingredients  []IngredientGroup
	Instructions []string
	Yield        *string
	Times        map[string]string
	Notes        *string
	SourceURL    string
}

func (s *Store) CreateRecipeExtraction(ctx context.Context, sourceURL string) (RecipeExtraction, error) {
	const q = `
		INSERT INTO recipe_extractions (source_url, status)
		VALUES ($1, 'queued')
		ON CONFLICT (source_url) DO UPDATE
		SET source_url = EXCLUDED.source_url
		RETURNING
			id::text,
			source_url,
			status,
			recipe_id::text,
			error_message,
			created_at,
			updated_at
	`

	var extraction RecipeExtraction
	var recipeID sql.NullString
	var errorMessage sql.NullString
	err := s.Pool.QueryRow(ctx, q, sourceURL).Scan(
		&extraction.ID,
		&extraction.SourceURL,
		&extraction.Status,
		&recipeID,
		&errorMessage,
		&extraction.CreatedAt,
		&extraction.UpdatedAt,
	)
	extraction.RecipeID = nullableStringPtr(recipeID)
	extraction.ErrorMessage = nullableStringPtr(errorMessage)
	return extraction, err
}

func (s *Store) GetRecipeExtractionByID(ctx context.Context, id string) (RecipeExtraction, error) {
	const q = `
		SELECT
			id::text,
			source_url,
			status,
			recipe_id::text,
			error_message,
			created_at,
			updated_at
		FROM recipe_extractions
		WHERE id = $1
	`

	var extraction RecipeExtraction
	var recipeID sql.NullString
	var errorMessage sql.NullString
	err := s.Pool.QueryRow(ctx, q, id).Scan(
		&extraction.ID,
		&extraction.SourceURL,
		&extraction.Status,
		&recipeID,
		&errorMessage,
		&extraction.CreatedAt,
		&extraction.UpdatedAt,
	)
	extraction.RecipeID = nullableStringPtr(recipeID)
	extraction.ErrorMessage = nullableStringPtr(errorMessage)
	return extraction, err
}

func (s *Store) ClaimNextQueuedRecipeExtraction(ctx context.Context) (*RecipeExtraction, error) {
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var id string
	err = tx.QueryRow(ctx, `
		SELECT id::text
		FROM recipe_extractions
		WHERE status = 'queued'
		ORDER BY created_at
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	var extraction RecipeExtraction
	var recipeID sql.NullString
	var errorMessage sql.NullString
	err = tx.QueryRow(ctx, `
		UPDATE recipe_extractions
		SET status = 'extracting', updated_at = now()
		WHERE id = $1
		RETURNING
			id::text,
			source_url,
			status,
			recipe_id::text,
			error_message,
			created_at,
			updated_at
	`, id).Scan(
		&extraction.ID,
		&extraction.SourceURL,
		&extraction.Status,
		&recipeID,
		&errorMessage,
		&extraction.CreatedAt,
		&extraction.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	extraction.RecipeID = nullableStringPtr(recipeID)
	extraction.ErrorMessage = nullableStringPtr(errorMessage)

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &extraction, nil
}

func (s *Store) UpdateRecipeExtractionStatus(ctx context.Context, id, status string, recipeID, errorMessage *string) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE recipe_extractions
		SET
			status = $2,
			recipe_id = $3,
			error_message = $4,
			updated_at = now()
		WHERE id = $1
	`, id, status, recipeID, errorMessage)
	return err
}

func (s *Store) GetRecipeByID(ctx context.Context, id string) (Recipe, error) {
	const q = `
		SELECT id::text, title, ingredients, instructions, yield, times, notes, source_url, created_at
		FROM recipes
		WHERE id = $1
	`

	var r Recipe
	var ingredientsRaw, instructionsRaw, timesRaw []byte
	var yield, notes sql.NullString

	err := s.Pool.QueryRow(ctx, q, id).Scan(
		&r.ID, &r.Title, &ingredientsRaw, &instructionsRaw,
		&yield, &timesRaw, &notes, &r.SourceURL, &r.CreatedAt,
	)
	if err != nil {
		return Recipe{}, err
	}

	if err := json.Unmarshal(ingredientsRaw, &r.Ingredients); err != nil {
		return Recipe{}, err
	}
	if err := json.Unmarshal(instructionsRaw, &r.Instructions); err != nil {
		return Recipe{}, err
	}
	if len(timesRaw) > 0 {
		if err := json.Unmarshal(timesRaw, &r.Times); err != nil {
			return Recipe{}, err
		}
	}
	r.Yield = nullableStringPtr(yield)
	r.Notes = nullableStringPtr(notes)

	return r, nil
}

type RecipeSummary struct {
	ID    string
	Title string
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
	ingredientsJSON, err := json.Marshal(input.Ingredients)
	if err != nil {
		return "", err
	}
	instructionsJSON, err := json.Marshal(input.Instructions)
	if err != nil {
		return "", err
	}

	var timesJSON []byte
	if input.Times != nil {
		timesJSON, err = json.Marshal(input.Times)
		if err != nil {
			return "", err
		}
	}

	const q = `
		INSERT INTO recipes (title, ingredients, instructions, yield, times, notes, source_url)
		VALUES ($1, $2::jsonb, $3::jsonb, $4, $5::jsonb, $6, $7)
		ON CONFLICT (source_url) DO UPDATE
		SET
			title = EXCLUDED.title,
			ingredients = EXCLUDED.ingredients,
			instructions = EXCLUDED.instructions,
			yield = EXCLUDED.yield,
			times = EXCLUDED.times,
			notes = EXCLUDED.notes
		RETURNING id::text
	`

	var recipeID string
	err = s.Pool.QueryRow(
		ctx,
		q,
		input.Title,
		string(ingredientsJSON),
		string(instructionsJSON),
		input.Yield,
		nullableJSONString(timesJSON),
		input.Notes,
		input.SourceURL,
	).Scan(&recipeID)
	if err != nil {
		return "", err
	}
	return recipeID, nil
}

func nullableStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	out := v.String
	return &out
}

func nullableJSONString(v []byte) *string {
	if len(v) == 0 {
		return nil
	}
	out := string(v)
	return &out
}
