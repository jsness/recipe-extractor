import type { RecipeSummary } from "../types";

export type RecipeSort = "extracted-desc" | "extracted-asc" | "title-asc" | "title-desc";

export const RECIPE_SORT_STORAGE_KEY = "recipe-extractor.recipe-sort";
export const DEFAULT_RECIPE_SORT: RecipeSort = "extracted-desc";
export const RECIPE_SORT_OPTIONS: { value: RecipeSort; label: string }[] = [
  { value: "extracted-desc", label: "Extracted: newest first" },
  { value: "extracted-asc", label: "Extracted: oldest first" },
  { value: "title-asc", label: "Title: A to Z" },
  { value: "title-desc", label: "Title: Z to A" },
];

export const isRecipeSort = (value: unknown): value is RecipeSort => (
  RECIPE_SORT_OPTIONS.some((option) => option.value === value)
);

export const loadRecipeSort = (): RecipeSort => {
  try {
    const stored = window.localStorage.getItem(RECIPE_SORT_STORAGE_KEY);
    if (isRecipeSort(stored)) return stored;
  } catch {
    // Sorting still works when browser storage is unavailable.
  }
  return DEFAULT_RECIPE_SORT;
};

const titleCollator = new Intl.Collator(undefined, { sensitivity: "base", numeric: true });

const compareDates = (a: number, b: number, direction: number): number => {
  if (!Number.isFinite(a)) return Number.isFinite(b) ? 1 : 0;
  if (!Number.isFinite(b)) return -1;
  return (a - b) * direction;
};

export const sortRecipes = (
  recipes: readonly RecipeSummary[],
  order: RecipeSort,
): RecipeSummary[] => {
  const direction = order.endsWith("asc") ? 1 : -1;
  const byTitle = order.startsWith("title");
  return recipes
    .map((recipe) => ({ recipe, timestamp: Date.parse(recipe.created_at) }))
    .sort((a, b) => {
      const title = titleCollator.compare(a.recipe.title.trim(), b.recipe.title.trim());
      const primary = byTitle
        ? title * direction
        : compareDates(a.timestamp, b.timestamp, direction);
      const secondary = byTitle ? compareDates(a.timestamp, b.timestamp, -1) : title;
      const id = a.recipe.id < b.recipe.id ? -1 : a.recipe.id > b.recipe.id ? 1 : 0;
      return primary || secondary || id;
    })
    .map(({ recipe }) => recipe);
};
