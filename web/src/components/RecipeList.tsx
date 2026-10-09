import { Badge, Button, Card, Flex, Group, Select, Stack, Text, TextInput, Title } from "@mantine/core";
import { RecipeSummary } from "../types";
import { isRecipeSort, RECIPE_SORT_OPTIONS, type RecipeSort } from "../utils/sortRecipes";

type RecipeListProps = {
  recipes: RecipeSummary[];
  loadingRecipeId: string | null;
  onView: (id: string) => void;
  newRecipeId?: string | null;
  searchQuery: string;
  setSearchQuery: (value: string) => void;
  sortOrder: RecipeSort;
  onSortChange: (value: RecipeSort) => void;
};

export const RecipeList = ({
  recipes,
  loadingRecipeId,
  onView,
  newRecipeId,
  searchQuery,
  setSearchQuery,
  sortOrder,
  onSortChange,
}: RecipeListProps) => (
  <Stack gap="xs">
    <Title order={3}>Recipes</Title>
    <Flex direction={{ base: "column", sm: "row" }} align="stretch" gap="sm">
      <TextInput
        label="Search recipes"
        value={searchQuery}
        onChange={(event) => setSearchQuery(event.currentTarget.value)}
        placeholder="Search recipes by title"
        flex={1}
        miw={0}
      />
      <Select
        label="Sort by"
        data={RECIPE_SORT_OPTIONS}
        value={sortOrder}
        onChange={(value) => {
          if (isRecipeSort(value)) onSortChange(value);
        }}
        clearable={false}
        allowDeselect={false}
        w={{ base: "100%", sm: 220 }}
      />
    </Flex>
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
            loading={loadingRecipeId === recipe.id}
            onClick={() => onView(recipe.id)}
          >
            View
          </Button>
        </Group>
      </Card>
    ))}
  </Stack>
);
