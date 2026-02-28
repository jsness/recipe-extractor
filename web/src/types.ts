export type CreateRecipeResponse = {
  extraction_id: string;
  status: string;
};

export type ExtractionStatusResponse = {
  id: string;
  source_url: string;
  status: string;
  recipe_id?: string;
  error_message?: string;
};

export type RecipeSummary = {
  id: string;
  title: string;
};

export type IngredientGroup = {
  group: string;
  items: string[];
};

export type Recipe = {
  id: string;
  title: string;
  ingredients: IngredientGroup[];
  instructions: string[];
  yield?: string;
  times?: Record<string, string>;
  notes?: string;
  source_url: string;
  created_at: string;
};
