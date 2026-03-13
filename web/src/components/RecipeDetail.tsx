import { Anchor, Badge, Button, Divider, Group, List, Stack, Text, Title } from "@mantine/core";
import { Recipe } from "../types";

type RecipeDetailProps = {
  recipe: Recipe;
  onBack: () => void;
  onDelete: (id: string) => Promise<void>;
  onSelectRecipe: (id: string) => void;
};

export const RecipeDetail = ({ recipe, onBack, onDelete, onSelectRecipe }: RecipeDetailProps) => {
  const timeEntries = recipe.times ? Object.entries(recipe.times) : [];

  const handleDelete = async () => {
    const confirmed = window.confirm(`Delete "${recipe.title}"? This cannot be undone.`);
    if (!confirmed) {
      return;
    }

    await onDelete(recipe.id);
  };

  return (
    <Stack gap="lg">
      <Group justify="space-between" align="center">
        <Button variant="subtle" size="sm" onClick={onBack} w="fit-content" px={0}>
          {"<- Back to recipes"}
        </Button>
        <Button color="red" variant="light" size="xs" onClick={() => void handleDelete()}>
          Delete recipe
        </Button>
      </Group>

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

      {recipe.related_recipes?.some((r) => r.relationship === "component") && (
        <>
          <Divider />
          <div>
            <Title order={4} mb="xs">Sub-recipes</Title>
            <Stack gap={4}>
              {recipe.related_recipes
                .filter((r) => r.relationship === "component")
                .map((r) => (
                  <Button
                    key={r.id}
                    variant="subtle"
                    size="sm"
                    onClick={() => onSelectRecipe(r.id)}
                    justify="flex-start"
                    px={0}
                  >
                    {r.title}
                  </Button>
                ))}
            </Stack>
          </div>
        </>
      )}

      {recipe.related_recipes?.some((r) => r.relationship === "used_in") && (
        <>
          <Divider />
          <div>
            <Title order={4} mb="xs">Used in</Title>
            <Stack gap={4}>
              {recipe.related_recipes
                .filter((r) => r.relationship === "used_in")
                .map((r) => (
                  <Button
                    key={r.id}
                    variant="subtle"
                    size="sm"
                    onClick={() => onSelectRecipe(r.id)}
                    justify="flex-start"
                    px={0}
                  >
                    {r.title}
                  </Button>
                ))}
            </Stack>
          </div>
        </>
      )}
    </Stack>
  );
};
