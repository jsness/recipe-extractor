CREATE TABLE IF NOT EXISTS profiles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE recipes
  ADD COLUMN profile_id UUID REFERENCES profiles(id);

ALTER TABLE recipe_extractions
  ADD COLUMN profile_id UUID REFERENCES profiles(id);

ALTER TABLE recipe_relationships
  ADD COLUMN profile_id UUID REFERENCES profiles(id);

WITH legacy_profile AS (
  INSERT INTO profiles (name)
  VALUES ('Legacy')
  RETURNING id
)
UPDATE recipes
SET profile_id = (SELECT id FROM legacy_profile)
WHERE profile_id IS NULL;

WITH legacy_profile AS (
  SELECT id
  FROM profiles
  WHERE name = 'Legacy'
  ORDER BY created_at
  LIMIT 1
)
UPDATE recipe_extractions
SET profile_id = (SELECT id FROM legacy_profile)
WHERE profile_id IS NULL;

UPDATE recipe_relationships rr
SET profile_id = r.profile_id
FROM recipes r
WHERE rr.parent_recipe_id = r.id
  AND rr.profile_id IS NULL;

ALTER TABLE recipes
  ALTER COLUMN profile_id SET NOT NULL;

ALTER TABLE recipe_extractions
  ALTER COLUMN profile_id SET NOT NULL;

ALTER TABLE recipe_relationships
  ALTER COLUMN profile_id SET NOT NULL;

ALTER TABLE recipes
  DROP CONSTRAINT IF EXISTS recipes_source_url_key;

ALTER TABLE recipes
  ADD CONSTRAINT recipes_profile_id_source_url_key UNIQUE (profile_id, source_url);

ALTER TABLE recipe_extractions
  DROP CONSTRAINT IF EXISTS recipe_extractions_source_url_key;

ALTER TABLE recipe_extractions
  ADD CONSTRAINT recipe_extractions_profile_id_source_url_key UNIQUE (profile_id, source_url);

ALTER TABLE recipe_relationships
  DROP CONSTRAINT IF EXISTS recipe_relationships_pkey;

ALTER TABLE recipe_relationships
  ADD CONSTRAINT recipe_relationships_pkey PRIMARY KEY (profile_id, parent_recipe_id, child_recipe_id);

CREATE INDEX IF NOT EXISTS idx_recipes_profile_id_created_at
  ON recipes (profile_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_recipe_extractions_profile_id_status_created_at
  ON recipe_extractions (profile_id, status, created_at);
