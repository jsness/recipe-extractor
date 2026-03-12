package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
)

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

func (s *Store) GetRecipeExtractionBySourceURL(ctx context.Context, sourceURL string) (*RecipeExtraction, error) {
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
		WHERE source_url = $1
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
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	extraction.RecipeID = nullableStringPtr(recipeID)
	extraction.ErrorMessage = nullableStringPtr(errorMessage)
	return &extraction, nil
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
	var parentRecipeID sql.NullString
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
			parent_recipe_id::text,
			created_at,
			updated_at
	`, id).Scan(
		&extraction.ID,
		&extraction.SourceURL,
		&extraction.Status,
		&recipeID,
		&errorMessage,
		&parentRecipeID,
		&extraction.CreatedAt,
		&extraction.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	extraction.RecipeID = nullableStringPtr(recipeID)
	extraction.ErrorMessage = nullableStringPtr(errorMessage)
	extraction.ParentRecipeID = nullableStringPtr(parentRecipeID)

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
