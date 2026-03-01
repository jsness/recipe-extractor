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

export type RelatedRecipe = {
  id: string;
  title: string;
  relationship: "component" | "used_in";
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
  related_recipes?: RelatedRecipe[];
};
