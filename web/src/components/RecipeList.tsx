import { Badge, Button, Card, Group, Stack, Text, TextInput, Title } from "@mantine/core";
import { RecipeSummary } from "../types";

type RecipeListProps = {
  recipes: RecipeSummary[];
  isLoadingRecipe: boolean;
  onView: (id: string) => void;
  newRecipeId?: string | null;
  searchQuery: string;
  setSearchQuery: (value: string) => void;
};

export const RecipeList = ({
  recipes,
  isLoadingRecipe,
  onView,
  newRecipeId,
  searchQuery,
  setSearchQuery,
}: RecipeListProps) => (
  <Stack gap="xs">
    <Title order={3}>Recipes</Title>
    <TextInput
      value={searchQuery}
      onChange={(event) => setSearchQuery(event.currentTarget.value)}
      placeholder="Search recipes by title"
    />
    {recipes.length === 0 && (
      <Text c="dimmed" size="sm">
        No recipes match that title.
      </Text>
    )}
    {recipes.map((recipe) => (
      <Card key={recipe.id} withBorder radius="md" padding="sm">
        <Group justify="space-between" align="flex-start" wrap="nowrap">
          <Group gap="xs" style={{ flex: 1, minWidth: 0 }}>
            <Text style={{ flex: 1 }}>{recipe.title}</Text>
            {recipe.id === newRecipeId && (
              <Badge color="cyan" variant="light" size="xs" style={{ flexShrink: 0 }}>New</Badge>
            )}
          </Group>
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
