-- Store which URLs a recipe linked to (even before those are extracted)
ALTER TABLE recipes
    ADD COLUMN linked_recipe_urls JSONB NOT NULL DEFAULT '[]'::jsonb;

-- Track which recipe triggered an auto-queued extraction
ALTER TABLE recipe_extractions
    ADD COLUMN parent_recipe_id UUID REFERENCES recipes(id) ON DELETE SET NULL;

-- Permanent relationship record once both recipes exist
CREATE TABLE recipe_relationships (
    parent_recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    child_recipe_id  UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (parent_recipe_id, child_recipe_id)
);
