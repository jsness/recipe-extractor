package store

import "context"

// QueueLinkedRecipeExtraction queues a linked URL for extraction tied to a parent recipe.
// If the URL is already extracted, it creates the relationship immediately.
// If already queued or extracting, it no-ops (relationship will be created on completion).
func (s *Store) QueueLinkedRecipeExtraction(ctx context.Context, parentRecipeID, linkedURL string) error {
	existing, err := s.GetRecipeExtractionBySourceURL(ctx, linkedURL)
	if err != nil {
		return err
	}
	if existing != nil {
		if existing.Status == "done" && existing.RecipeID != nil {
			return s.CreateRecipeRelationship(ctx, parentRecipeID, *existing.RecipeID)
		}
		return nil
	}

	_, err = s.Pool.Exec(ctx, `
		INSERT INTO recipe_extractions (source_url, status, parent_recipe_id)
		VALUES ($1, 'queued', $2)
	`, linkedURL, parentRecipeID)
	return err
}

// CreateRecipeRelationship records a parent->child recipe relationship (idempotent).
func (s *Store) CreateRecipeRelationship(ctx context.Context, parentID, childID string) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO recipe_relationships (parent_recipe_id, child_recipe_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, parentID, childID)
	return err
}

// GetRelatedRecipes returns recipes linked from (component) or linking to (used_in) the given recipe.
func (s *Store) GetRelatedRecipes(ctx context.Context, recipeID string) ([]RelatedRecipe, error) {
	const q = `
		SELECT r.id::text, r.title, 'component' AS relationship
		FROM recipe_relationships rr
		JOIN recipes r ON r.id = rr.child_recipe_id
		WHERE rr.parent_recipe_id = $1
		UNION
		SELECT r.id::text, r.title, 'used_in' AS relationship
		FROM recipe_relationships rr
		JOIN recipes r ON r.id = rr.parent_recipe_id
		WHERE rr.child_recipe_id = $1
	`

	rows, err := s.Pool.Query(ctx, q, recipeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var related []RelatedRecipe
	for rows.Next() {
		var r RelatedRecipe
		if err := rows.Scan(&r.ID, &r.Title, &r.Relationship); err != nil {
			return nil, err
		}
		related = append(related, r)
	}
	return related, rows.Err()
}
