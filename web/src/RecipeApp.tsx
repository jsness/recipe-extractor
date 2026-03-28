import React, { useEffect, useMemo, useState } from "react";
import { Alert, Anchor, Button, Container, Group, Stack, Text, Title } from "@mantine/core";
import {
  CreateRecipeResponse,
  ExtractionStatusResponse,
  Profile,
  Recipe,
  RecipeSummary,
} from "./types";
import { ExtractForm } from "./components/ExtractForm";
import { ExtractionCard } from "./components/ExtractionCard";
import { ProfilePanel } from "./components/ProfilePanel";
import { RecipeList } from "./components/RecipeList";
import { RecipeDetail } from "./components/RecipeDetail";
import GithubLogo from "./icons/GithubLogo";

const ACTIVE_PROFILE_STORAGE_KEY = "recipe-extractor.active-profile-id";

export const RecipeApp = () => {
  const [url, setURL] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [activeProfileId, setActiveProfileId] = useState<string | null>(null);
  const [isProfilePanelVisible, setIsProfilePanelVisible] = useState(true);
  const [isLoadingProfiles, setIsLoadingProfiles] = useState(true);
  const [isCreatingProfile, setIsCreatingProfile] = useState(false);
  const [createProfileName, setCreateProfileName] = useState("");
  const [profileError, setProfileError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState("");
  const [extraction, setExtraction] = useState<ExtractionStatusResponse | null>(null);
  const [recipes, setRecipes] = useState<RecipeSummary[]>([]);
  const [selectedRecipe, setSelectedRecipe] = useState<Recipe | null>(null);
  const [loadingRecipeId, setLoadingRecipeId] = useState<string | null>(null);
  const [newRecipeId, setNewRecipeId] = useState<string | null>(null);

  const terminalStatuses = useMemo(() => new Set(["done", "failed"]), []);
  const normalizedSearchQuery = searchQuery.trim().toLowerCase();
  const activeProfile = useMemo(
    () => profiles.find((profile) => profile.id === activeProfileId) ?? null,
    [activeProfileId, profiles],
  );
  const filteredRecipes = useMemo(() => {
    if (normalizedSearchQuery === "") {
      return recipes;
    }

    return recipes.filter((recipe) => recipe.title.toLowerCase().includes(normalizedSearchQuery));
  }, [normalizedSearchQuery, recipes]);

  const profileHeaders = (profileId: string) => ({
    "X-Profile-Id": profileId,
  });

  const loadProfiles = async () => {
    const res = await fetch("/api/v1/profiles");
    if (!res.ok) {
      throw new Error(`Failed to load profiles (${res.status})`);
    }

    const profileList = (await res.json()) as Profile[];
    setProfiles(profileList);

    const storedProfileId = window.localStorage.getItem(ACTIVE_PROFILE_STORAGE_KEY);
    if (storedProfileId && profileList.some((profile) => profile.id === storedProfileId)) {
      setActiveProfileId(storedProfileId);
      setIsProfilePanelVisible(false);
      return;
    }

    if (storedProfileId) {
      window.localStorage.removeItem(ACTIVE_PROFILE_STORAGE_KEY);
    }
    setActiveProfileId(null);
    setIsProfilePanelVisible(true);
  };

  const loadRecipes = async (profileId: string) => {
    const res = await fetch("/api/v1/recipes", {
      headers: profileHeaders(profileId),
    });
    if (!res.ok) {
      throw new Error(`Failed to load recipes (${res.status})`);
    }
    setRecipes((await res.json()) as RecipeSummary[]);
  };

  useEffect(() => {
    loadProfiles()
      .catch((error) => {
        const message = error instanceof Error ? error.message : "Unknown profile load error";
        setProfileError(message);
      })
      .finally(() => setIsLoadingProfiles(false));
  }, []);

  useEffect(() => {
    if (!activeProfileId || !extraction || terminalStatuses.has(extraction.status)) {
      return;
    }

    const interval = setInterval(async () => {
      try {
        const res = await fetch(`/api/v1/recipe-extractions/${extraction.id}`, {
          headers: profileHeaders(activeProfileId),
        });
        if (!res.ok) {
          throw new Error(`Status check failed (${res.status})`);
        }
        const body = (await res.json()) as ExtractionStatusResponse;
        setExtraction(body);
        if (body.status === "done" && body.recipe_id) {
          await loadRecipes(activeProfileId);
          setNewRecipeId(body.recipe_id);
        }
      } catch (error) {
        const message = error instanceof Error ? error.message : "Unknown polling error";
        setSubmitError(message);
      }
    }, 1500);

    return () => clearInterval(interval);
  }, [activeProfileId, extraction, terminalStatuses]);

  useEffect(() => {
    if (activeProfileId == null) {
      setRecipes([]);
      setSelectedRecipe(null);
      setExtraction(null);
      setNewRecipeId(null);
      return;
    }

    window.localStorage.setItem(ACTIVE_PROFILE_STORAGE_KEY, activeProfileId);
    setSelectedRecipe(null);
    setExtraction(null);
    setNewRecipeId(null);
    setSearchQuery("");
    setSubmitError("");

    loadRecipes(activeProfileId).catch((error) => {
      const message = error instanceof Error ? error.message : "Unknown recipe load error";
      setSubmitError(message);
    });
  }, [activeProfileId]);

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!activeProfileId) {
      setSubmitError("Select a profile before extracting recipes.");
      return;
    }
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
        headers: {
          "Content-Type": "application/json",
          ...profileHeaders(activeProfileId),
        },
        body: JSON.stringify({ url: parsedURL.toString() }),
      });

      if (!res.ok) {
        let message = `Create request failed (${res.status})`;
        try {
          const body = await res.json();
          if (body.error) message = body.error;
        } catch {
          // ignore JSON parse failures for error handling
        }
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
    if (!activeProfileId) {
      return;
    }

    setLoadingRecipeId(id);
    try {
      const res = await fetch(`/api/v1/recipes/${id}`, {
        headers: profileHeaders(activeProfileId),
      });
      if (!res.ok) {
        throw new Error(`Failed to load recipe (${res.status})`);
      }
      setSelectedRecipe((await res.json()) as Recipe);
    } catch {
      // TODO: surface error
    } finally {
      setLoadingRecipeId(null);
    }
  };

  const handleDeleteRecipe = async (id: string) => {
    if (!activeProfileId) {
      return;
    }

    setSubmitError("");

    try {
      const res = await fetch(`/api/v1/recipes/${id}`, {
        method: "DELETE",
        headers: profileHeaders(activeProfileId),
      });
      if (!res.ok) {
        throw new Error(`Failed to delete recipe (${res.status})`);
      }

      setSelectedRecipe(null);
      if (newRecipeId === id) {
        setNewRecipeId(null);
      }
      await loadRecipes(activeProfileId);
    } catch (error) {
      const message = error instanceof Error ? error.message : "Unknown delete error";
      setSubmitError(message);
    }
  };

  const handleCreateProfile = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setProfileError("");

    const name = createProfileName.trim();
    if (name === "") {
      setProfileError("Profile name is required.");
      return;
    }

    setIsCreatingProfile(true);
    try {
      const res = await fetch("/api/v1/profiles", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name }),
      });
      if (!res.ok) {
        let message = `Create profile failed (${res.status})`;
        try {
          const body = await res.json();
          if (body.error) {
            message = body.error;
          }
        } catch {
          // ignore JSON parse failures for error handling
        }
        throw new Error(message);
      }

      const profile = (await res.json()) as Profile;
      setProfiles((current) => [...current, profile]);
      setActiveProfileId(profile.id);
      setIsProfilePanelVisible(false);
      setCreateProfileName("");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Unknown profile creation error";
      setProfileError(message);
    } finally {
      setIsCreatingProfile(false);
    }
  };

  const isPolling = extraction != null && !terminalStatuses.has(extraction.status);
  const handleSelectProfile = (profileId: string | null) => {
    setActiveProfileId(profileId);
    setIsProfilePanelVisible(profileId == null);
  };

  return (
    <Container size="sm" pt="xl">
      <Stack gap="lg">
        <div>
          <Group justify="space-between" align="flex-start">
            <div>
              <Title order={1}>Recipe Extractor</Title>
              <Text c="dimmed">
                {activeProfile
                  ? `Active profile: ${activeProfile.name}`
                  : "Choose a profile before loading recipes or extracting new ones."}
              </Text>
            </div>
            {activeProfile && !isProfilePanelVisible && (
              <Button
                size="compact-xs"
                variant="subtle"
                onClick={() => setIsProfilePanelVisible(true)}
              >
                Switch Profile
              </Button>
            )}
          </Group>
        </div>

        {isProfilePanelVisible && (
          <ProfilePanel
            profiles={profiles}
            activeProfileId={activeProfileId}
            isLoadingProfiles={isLoadingProfiles}
            isCreatingProfile={isCreatingProfile}
            createProfileName={createProfileName}
            setCreateProfileName={setCreateProfileName}
            onSelectProfile={handleSelectProfile}
            onCreateProfile={handleCreateProfile}
          />
        )}

        {profileError && (
          <Alert color="red" title="Profile Error" withCloseButton onClose={() => setProfileError("")}>
            {profileError}
          </Alert>
        )}

        {activeProfileId && (
          <ExtractForm
            url={url}
            setURL={setURL}
            onSubmit={handleSubmit}
            isSubmitting={isSubmitting}
          />
        )}

        {submitError && (
          <Alert color="red" title="Error" withCloseButton onClose={() => setSubmitError("")}>
            {submitError}
          </Alert>
        )}

        {extraction && (
          <ExtractionCard extraction={extraction} isPolling={isPolling} />
        )}

        {activeProfileId && selectedRecipe ? (
          <RecipeDetail
            recipe={selectedRecipe}
            onBack={() => setSelectedRecipe(null)}
            onDelete={handleDeleteRecipe}
            onSelectRecipe={handleViewRecipe}
          />
        ) : activeProfileId ? (
          (recipes.length > 0 || searchQuery.trim() !== "") && (
            <RecipeList
              recipes={filteredRecipes}
              loadingRecipeId={loadingRecipeId}
              onView={handleViewRecipe}
              newRecipeId={newRecipeId}
              searchQuery={searchQuery}
              setSearchQuery={setSearchQuery}
            />
          )
        ) : null}

        {activeProfileId && recipes.length === 0 && searchQuery.trim() === "" && !selectedRecipe && !isLoadingProfiles && (
          <Text c="dimmed" size="sm">
            No recipes saved for this profile yet.
          </Text>
        )}

        <Group justify="center">
          <Anchor
            href="https://github.com/jsness/recipe-extractor"
            target="_blank"
            c="dimmed"
          >
            <GithubLogo />
          </Anchor>
        </Group>
      </Stack>
    </Container>
  );
};
