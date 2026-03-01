import React, { useEffect, useMemo, useState } from "react";
import { Alert, Container, Stack, Text, Title } from "@mantine/core";
import { CreateRecipeResponse, ExtractionStatusResponse, Recipe, RecipeSummary } from "./types";
import { ExtractForm } from "./components/ExtractForm";
import { ExtractionCard } from "./components/ExtractionCard";
import { RecipeList } from "./components/RecipeList";
import { RecipeDetail } from "./components/RecipeDetail";

export const App = () => {
  const [url, setURL] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState("");
  const [extraction, setExtraction] = useState<ExtractionStatusResponse | null>(null);
  const [recipes, setRecipes] = useState<RecipeSummary[]>([]);
  const [selectedRecipe, setSelectedRecipe] = useState<Recipe | null>(null);
  const [isLoadingRecipe, setIsLoadingRecipe] = useState(false);
  const [newRecipeId, setNewRecipeId] = useState<string | null>(null);

  const terminalStatuses = useMemo(() => new Set(["done", "failed"]), []);

  useEffect(() => {
    fetch("/api/v1/recipes")
      .then((res) => res.json())
      .then((data) => setRecipes(data as RecipeSummary[]))
      .catch(() => {});
  }, []);

  useEffect(() => {
    if (!extraction || terminalStatuses.has(extraction.status)) {
      return;
    }

    const interval = setInterval(async () => {
      try {
        const res = await fetch(`/api/v1/recipe-extractions/${extraction.id}`);
        if (!res.ok) {
          throw new Error(`Status check failed (${res.status})`);
        }
        const body = (await res.json()) as ExtractionStatusResponse;
        setExtraction(body);
        if (body.status === "done" && body.recipe_id) {
          const recipesRes = await fetch("/api/v1/recipes");
          if (recipesRes.ok) {
            setRecipes(await recipesRes.json() as RecipeSummary[]);
            setNewRecipeId(body.recipe_id);
          }
        }
      } catch (error) {
        const message = error instanceof Error ? error.message : "Unknown polling error";
        setSubmitError(message);
      }
    }, 1500);

    return () => clearInterval(interval);
  }, [extraction, terminalStatuses]);

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSubmitError("");
    setExtraction(null);
    setNewRecipeId(null);

    let parsedURL: URL;
    try {
      parsedURL = new URL(url);
      if (parsedURL.protocol !== "http:" && parsedURL.protocol !== "https:") {
        throw new Error("URL must start with http:// or https://");
      }
    } catch {
      setSubmitError("Please enter a valid URL.");
      return;
    }

    setIsSubmitting(true);

    try {
      const res = await fetch("/api/v1/recipes", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url: parsedURL.toString() }),
      });

      if (!res.ok) {
        let message = `Create request failed (${res.status})`;
        try {
          const body = await res.json();
          if (body.error) message = body.error;
        } catch {}
        throw new Error(message);
      }

      const body = (await res.json()) as CreateRecipeResponse;
      setExtraction({
        id: body.extraction_id,
        source_url: parsedURL.toString(),
        status: body.status,
      });
    } catch (error) {
      const message = error instanceof Error ? error.message : "Unknown request error";
      setSubmitError(message);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleViewRecipe = async (id: string) => {
    setIsLoadingRecipe(true);
    try {
      const res = await fetch(`/api/v1/recipes/${id}`);
      if (!res.ok) {
        throw new Error(`Failed to load recipe (${res.status})`);
      }
      setSelectedRecipe(await res.json() as Recipe);
    } catch {
      // TODO: surface error
    } finally {
      setIsLoadingRecipe(false);
    }
  };

  const isPolling = extraction != null && !terminalStatuses.has(extraction.status);

  return (
    <Container size="sm" pt="xl">
      <Stack gap="lg">
        <div>
          <Title order={1}>Recipe Extractor</Title>
          <Text c="dimmed">Paste a recipe URL to extract and save it.</Text>
        </div>

        <ExtractForm url={url} setURL={setURL} onSubmit={handleSubmit} isSubmitting={isSubmitting} />

        {submitError && (
          <Alert color="red" title="Error" withCloseButton onClose={() => setSubmitError("")}>
            {submitError}
          </Alert>
        )}

        {extraction && (
          <ExtractionCard extraction={extraction} isPolling={isPolling} />
        )}

        {selectedRecipe ? (
          <RecipeDetail recipe={selectedRecipe} onBack={() => setSelectedRecipe(null)} onSelectRecipe={handleViewRecipe} />
        ) : (
          recipes.length > 0 && (
            <RecipeList recipes={recipes} isLoadingRecipe={isLoadingRecipe} onView={handleViewRecipe} newRecipeId={newRecipeId} />
          )
        )}
      </Stack>
    </Container>
  );
};
