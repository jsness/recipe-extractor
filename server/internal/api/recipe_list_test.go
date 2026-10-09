package api

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jsness/recipe-extractor/server/core"
	"github.com/jsness/recipe-extractor/server/store"
)

func TestRecipeSummaryJSON(t *testing.T) {
	createdAt := time.Date(2026, 10, 9, 12, 34, 56, 123456000, time.UTC)
	body, err := json.Marshal(recipeSummaryResponse{ID: "recipe-1", Title: "Soup", CreatedAt: createdAt})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]string
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["id"] != "recipe-1" || fields["title"] != "Soup" || fields["created_at"] != "2026-10-09T12:34:56.123456Z" {
		t.Fatalf("unexpected recipe summary: %s", body)
	}
}

// Use a disposable database, as in TestReminderIntegration.
func TestRecipeListIntegration(t *testing.T) {
	databaseURL := os.Getenv("RECIPE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("RECIPE_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	logger := log.New(io.Discard, "", 0)
	if err := core.Migrate(ctx, pool, logger); err != nil {
		t.Fatal(err)
	}
	app, err := core.New(pool, core.Config{}, logger)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := app.CreateProfile(ctx, "Recipe list test")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(ctx, `DELETE FROM profiles WHERE id = $1`, profile.ID)
	other, err := app.CreateProfile(ctx, "Other recipe list test")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(ctx, `DELETE FROM profiles WHERE id = $1`, other.ID)
	defer pool.Exec(ctx, `DELETE FROM recipes WHERE profile_id IN ($1, $2)`, profile.ID, other.ID)

	routes := NewHandler(Config{}, app, logger).Routes()
	list := func(profileID string) []recipeSummaryResponse {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/recipes", nil)
		req.Header.Set("X-Profile-Id", profileID)
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("list status %d: %s", rec.Code, rec.Body.String())
		}
		var summaries []recipeSummaryResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &summaries); err != nil {
			t.Fatal(err)
		}
		if summaries == nil {
			t.Fatal("expected a JSON array, including for empty profiles")
		}
		return summaries
	}
	if got := list(profile.ID); len(got) != 0 {
		t.Fatalf("new profile has recipes: %+v", got)
	}

	older := time.Date(2026, 10, 9, 8, 0, 0, 123456000, time.UTC)
	newer := older.Add(time.Hour)
	create := func(profileID, title string, createdAt time.Time) string {
		t.Helper()
		id, err := app.Store().UpsertRecipe(ctx, profileID, store.RecipeInput{
			Title: title, SourceURL: "https://example.com/" + profileID + "/" + title,
			Ingredients: []store.IngredientGroup{}, Instructions: []string{},
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `UPDATE recipes SET created_at = $2 WHERE id = $1`, id, createdAt); err != nil {
			t.Fatal(err)
		}
		return id
	}
	olderID := create(profile.ID, "Older", older)
	newerID := create(profile.ID, "Newer", newer)
	otherID := create(other.ID, "Other", newer.Add(time.Hour))
	got := list(profile.ID)
	if len(got) != 2 || got[0].ID != newerID || got[1].ID != olderID {
		t.Fatalf("expected only active profile recipes, newest first: %+v", got)
	}
	if got[0].Title != "Newer" || !got[0].CreatedAt.Equal(newer) || !got[1].CreatedAt.Equal(older) {
		t.Fatalf("timestamps or title did not survive store and response mapping: %+v", got)
	}
	if got := list(other.ID); len(got) != 1 || got[0].ID != otherID {
		t.Fatalf("other profile list leaked recipes: %+v", got)
	}

	_, err = app.Store().UpsertRecipe(ctx, profile.ID, store.RecipeInput{
		Title: "Updated", SourceURL: "https://example.com/" + profile.ID + "/Older",
		Ingredients: []store.IngredientGroup{}, Instructions: []string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	got = list(profile.ID)
	if len(got) != 2 || got[1].ID != olderID || !got[1].CreatedAt.Equal(older) {
		t.Fatalf("upsert changed first-saved date: %+v", got)
	}
}
