import { Anchor, Badge, Button, Divider, Group, List, Stack, Text, Title } from "@mantine/core";
import { Recipe } from "../types";

type RecipeDetailProps = {
  recipe: Recipe;
  onBack: () => void;
};

export const RecipeDetail = ({ recipe, onBack }: RecipeDetailProps) => {
  const timeEntries = recipe.times ? Object.entries(recipe.times) : [];

  return (
    <Stack gap="lg">
      <Button variant="subtle" size="sm" onClick={onBack} w="fit-content" px={0}>
        ← Back to recipes
      </Button>

      <div>
        <Title order={2}>{recipe.title}</Title>
        <Anchor href={recipe.source_url} target="_blank" size="sm" c="dimmed" lineClamp={1}>
          {recipe.source_url}
        </Anchor>
      </div>

      {(recipe.yield || timeEntries.length > 0) && (
        <Group gap="xs">
          {recipe.yield && <Badge variant="light">{recipe.yield}</Badge>}
          {timeEntries.map(([key, value]) => (
            <Badge key={key} variant="outline">{key}: {value}</Badge>
          ))}
        </Group>
      )}

      <div>
        <Title order={4} mb="xs">Ingredients</Title>
        <Stack gap="sm">
          {recipe.ingredients.map((group, i) => (
            <div key={i}>
              {group.group && (
                <Text fw={600} size="sm" mb={4}>{group.group}</Text>
              )}
              <List size="sm" spacing={2}>
                {group.items.map((item, j) => (
                  <List.Item key={j}>{item}</List.Item>
                ))}
              </List>
            </div>
          ))}
        </Stack>
      </div>

      <Divider />

      <div>
        <Title order={4} mb="xs">Instructions</Title>
        <List type="ordered" size="sm" spacing="xs">
          {recipe.instructions.map((step, i) => (
            <List.Item key={i}>{step}</List.Item>
          ))}
        </List>
      </div>

      {recipe.notes && (
        <>
          <Divider />
          <div>
            <Title order={4} mb="xs">Notes</Title>
            <Text size="sm">{recipe.notes}</Text>
          </div>
        </>
      )}
    </Stack>
  );
};
