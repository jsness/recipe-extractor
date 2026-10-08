package api

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jsness/recipe-extractor/server/core"
	"github.com/jsness/recipe-extractor/server/migrations"
	"github.com/jsness/recipe-extractor/server/store"
)

func TestDecodeReminder(t *testing.T) {
	for _, body := range []string{`{}`, `null`, `{"reminder":null}`, `{"reminder":5}`, `{"reminder":true}`, `{"reminder":[]}`, `{`, `{"reminder":"ok"} {}`, `{"reminder":"ok","notes":"change"}`} {
		t.Run(body, func(t *testing.T) {
			if _, err := decodeReminder(httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(body))); err == nil {
				t.Fatal("expected invalid request to be rejected")
			}
		})
	}
	for _, reminder := range []string{"", "Use less salt.\nBake longer."} {
		body, _ := json.Marshal(map[string]string{"reminder": reminder})
		got, err := decodeReminder(httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(string(body))))
		if err != nil || got != reminder {
			t.Fatalf("decodeReminder = %q, %v; want %q", got, err, reminder)
		}
	}
}

func TestReminderResponse(t *testing.T) {
	reminder, notes := "Use less salt", "Extracted notes"
	for _, value := range []*string{nil, &reminder} {
		body, err := json.Marshal(newRecipeResponse(store.Recipe{Notes: &notes, Reminder: value}, nil))
		if err != nil {
			t.Fatal(err)
		}
		var response map[string]any
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatal(err)
		}
		if response["notes"] != notes {
			t.Fatal("notes changed")
		}
		got, present := response["reminder"]
		if value == nil && present {
			t.Fatal("absent reminder should be omitted")
		}
		if value != nil && got != reminder {
			t.Fatal("saved reminder missing")
		}
	}
}

// Set RECIPE_TEST_DATABASE_URL to a disposable PostgreSQL database. This test
// applies migrations and creates fixtures; never point it at a deployed database.
func TestReminderIntegration(t *testing.T) {
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
	profile, err := app.CreateProfile(ctx, "Reminder test")
	if err != nil {
		t.Fatal(err)
	}
	other, err := app.CreateProfile(ctx, "Other reminder test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM recipes WHERE profile_id IN ($1, $2)`, profile.ID, other.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM profiles WHERE id IN ($1, $2)`, profile.ID, other.ID)
	})
	notes := "Extracted notes"
	input := store.RecipeInput{Title: "Test recipe", SourceURL: "https://example.com/" + profile.ID, Notes: &notes,
		Ingredients: []store.IngredientGroup{{Items: []string{"Salt"}}}, Instructions: []string{"Bake"}}
	id, err := app.Store().UpsertRecipe(ctx, profile.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("rename preserves saved text", func(t *testing.T) {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		// Recreate the previous column name inside a rolled-back transaction.
		if _, err := tx.Exec(ctx, `ALTER TABLE recipes RENAME COLUMN reminder TO notice`); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, `UPDATE recipes SET notice = $2 WHERE id = $1`, id, "Saved before rename"); err != nil {
			t.Fatal(err)
		}
		sql, err := migrations.SQL.ReadFile("006_recipe_reminders.sql")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			t.Fatal(err)
		}
		var reminder, extractedNotes string
		if err := tx.QueryRow(ctx, `SELECT reminder, notes FROM recipes WHERE id = $1`, id).Scan(&reminder, &extractedNotes); err != nil {
			t.Fatal(err)
		}
		if reminder != "Saved before rename" || extractedNotes != notes {
			t.Fatal("migration changed saved text")
		}
	})
	routes := NewHandler(Config{}, app, logger).Routes()
	request := func(profileID, recipeID, body string, status int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/recipes/"+recipeID+"/reminder", strings.NewReader(body))
		if profileID != "" {
			req.Header.Set("X-Profile-Id", profileID)
		}
		res := httptest.NewRecorder()
		routes.ServeHTTP(res, req)
		if res.Code != status {
			t.Fatalf("status %d, want %d: %s", res.Code, status, res.Body.String())
		}
		return res
	}
	initial, err := app.GetRecipe(ctx, profile.ID, id)
	if err != nil || initial.Recipe.Reminder != nil {
		t.Fatalf("initial reminder: %+v, %v", initial, err)
	}
	request("", id, `{"reminder":"x"}`, 400)
	request("invalid", id, `{"reminder":"x"}`, 400)
	request(profile.ID, "invalid", `{"reminder":"x"}`, 400)
	request(profile.ID, id, `{"reminder":null}`, 400)
	request(other.ID, id, `{"reminder":"cross-profile"}`, 404)
	request(profile.ID, "00000000-0000-0000-0000-000000000000", `{"reminder":"x"}`, 404)
	for _, text := range []string{"  Use less salt.\nBake longer.  ", "Edited reminder"} {
		body, _ := json.Marshal(map[string]string{"reminder": text})
		res := request(profile.ID, id, string(body), 200)
		var saved recipeResponse
		if err := json.Unmarshal(res.Body.Bytes(), &saved); err != nil {
			t.Fatal(err)
		}
		if saved.Reminder == nil || *saved.Reminder != strings.TrimSpace(text) {
			t.Fatal("reminder not normalized and saved")
		}
		if saved.Notes == nil || *saved.Notes != notes || saved.Title != input.Title || len(saved.Instructions) != 1 {
			t.Fatal("other recipe fields changed")
		}
	}
	newNotes := "New extracted notes"
	input.Notes = &newNotes
	if _, err := app.Store().UpsertRecipe(ctx, profile.ID, input); err != nil {
		t.Fatal(err)
	}
	saved, err := app.GetRecipe(ctx, profile.ID, id)
	if err != nil || saved.Recipe.Reminder == nil || *saved.Recipe.Reminder != "Edited reminder" || *saved.Recipe.Notes != newNotes {
		t.Fatalf("upsert did not preserve reminder: %+v, %v", saved, err)
	}
	request(profile.ID, id, `{"reminder":" \n\t "}`, 200)
	if _, err := app.Store().UpsertRecipe(ctx, profile.ID, input); err != nil {
		t.Fatal(err)
	}
	saved, err = app.GetRecipe(ctx, profile.ID, id)
	if err != nil || saved.Recipe.Reminder != nil {
		t.Fatalf("reminder removal not persisted: %+v, %v", saved, err)
	}
}
