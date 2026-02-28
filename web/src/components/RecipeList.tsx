import { Button, Card, Group, Stack, Text, Title } from "@mantine/core";
import { RecipeSummary } from "../types";

type RecipeListProps = {
  recipes: RecipeSummary[];
  isLoadingRecipe: boolean;
  onView: (id: string) => void;
};

export const RecipeList = ({ recipes, isLoadingRecipe, onView }: RecipeListProps) => (
  <Stack gap="xs">
    <Title order={3}>Recipes</Title>
    {recipes.map((recipe) => (
      <Card key={recipe.id} withBorder radius="md" padding="sm">
        <Group justify="space-between" align="flex-start" wrap="nowrap">
          <Text style={{ flex: 1 }}>{recipe.title}</Text>
          <Button
            size="xs"
            variant="light"
            style={{ flexShrink: 0 }}
            loading={isLoadingRecipe}
            onClick={() => onView(recipe.id)}
          >
            View
          </Button>
        </Group>
      </Card>
    ))}
  </Stack>
);
