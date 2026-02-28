CREATE TABLE IF NOT EXISTS recipe_extractions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source_url TEXT UNIQUE NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('queued', 'extracting', 'done', 'failed')),
  recipe_id UUID REFERENCES recipes(id) ON DELETE SET NULL,
  error_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_recipe_extractions_status_created_at
  ON recipe_extractions (status, created_at);
