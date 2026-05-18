import { useEffect, useMemo, useState } from "react";
import { Anchor, Badge, Button, Divider, Group, List, NumberInput, Stack, Text, Title } from "@mantine/core";
import { Recipe } from "../types";
import { printRecipe } from "../utils/printRecipe";
import { extractServingCount, scaleIngredientGroups } from "../utils/servingScaler";

type RecipeDetailProps = {
  recipe: Recipe;
  onBack: () => void;
  onDelete: (id: string) => Promise<void>;
  onSelectRecipe: (id: string) => void;
};

const formatServingCount = (value: number) => {
  const rounded = Math.round(value);
  return Math.abs(value - rounded) < 0.00001 ? String(rounded) : String(value);
};

export const RecipeDetail = ({ recipe, onBack, onDelete, onSelectRecipe }: RecipeDetailProps) => {
  const baseServingCount = useMemo(() => extractServingCount(recipe.yield), [recipe.yield]);
  const [targetServingCount, setTargetServingCount] = useState<number | null>(baseServingCount);
  const timeEntries = recipe.times ? Object.entries(recipe.times) : [];
  const activeServingCount = targetServingCount ?? baseServingCount;
  const ingredientScale = baseServingCount && activeServingCount
    ? activeServingCount / baseServingCount
    : 1;
  const scaledIngredientGroups = useMemo(
    () => scaleIngredientGroups(recipe.ingredients, ingredientScale),
    [ingredientScale, recipe.ingredients],
  );
  const scaledYield = baseServingCount && activeServingCount
    ? `${formatServingCount(activeServingCount)} servings`
    : recipe.yield;
  const isScaled = baseServingCount != null
    && activeServingCount != null
    && Math.abs(activeServingCount - baseServingCount) > 0.00001;

  useEffect(() => {
    setTargetServingCount(baseServingCount);
  }, [baseServingCount, recipe.id]);

  const updateTargetServingCount = (value: number | string) => {
    setTargetServingCount(typeof value === "number" ? value : baseServingCount);
  };

  const adjustTargetServingCount = (delta: number) => {
    if (!baseServingCount) {
      return;
    }

    const nextServingCount = Math.max(
      1,
      Math.min(999, (activeServingCount ?? baseServingCount) + delta),
    );
    setTargetServingCount(nextServingCount);
  };

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
        <Button
          variant="light"
          size="xs"
          onClick={() => printRecipe({
            ...recipe,
            ingredients: scaledIngredientGroups,
            yield: isScaled ? scaledYield : recipe.yield,
          })}
        >
          Print
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
          {recipe.yield && <Badge variant="light">{isScaled ? scaledYield : recipe.yield}</Badge>}
          {timeEntries.map(([key, value]) => (
            <Badge key={key} variant="outline">{key}: {value}</Badge>
          ))}
        </Group>
      )}

      <div>
        <Group justify="space-between" align="flex-end" mb="xs">
          <Title order={4}>Ingredients</Title>
          <Group gap={6} align="flex-end" wrap="nowrap">
            <Button
              variant="light"
              size="md"
              px="sm"
              disabled={!baseServingCount || (activeServingCount ?? 1) <= 1}
              aria-label="Decrease servings"
              onClick={() => adjustTargetServingCount(-1)}
            >
              -
            </Button>
            <NumberInput
              label="Servings"
              value={targetServingCount ?? ""}
              min={1}
              max={999}
              step={1}
              allowDecimal={false}
              hideControls
              disabled={!baseServingCount}
              w={92}
              size="md"
              inputMode="numeric"
              onChange={updateTargetServingCount}
            />
            <Button
              variant="light"
              size="md"
              px="sm"
              disabled={!baseServingCount || (activeServingCount ?? 999) >= 999}
              aria-label="Increase servings"
              onClick={() => adjustTargetServingCount(1)}
            >
              +
            </Button>
            <Button
              variant="subtle"
              size="md"
              px="sm"
              disabled={!baseServingCount || !isScaled}
              onClick={() => setTargetServingCount(baseServingCount)}
            >
              Reset
            </Button>
          </Group>
        </Group>
        {!baseServingCount && (
          <Text c="dimmed" size="xs" mb="xs">
            Add a numeric yield to this recipe to scale ingredient amounts.
          </Text>
        )}
        <Stack gap="sm">
          {scaledIngredientGroups.map((group, i) => (
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

      <Divider />

      <Group justify="flex-end">
        <Button color="red" variant="light" size="xs" onClick={() => void handleDelete()}>
          Delete recipe
        </Button>
      </Group>
    </Stack>
  );
};
