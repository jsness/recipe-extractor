# Using `recipe-extractor` From Another Go App

If you want to build a separate hosted product on top of this repository, you do not need to run the standalone `cmd/server` app directly. You can import the reusable packages and compose them into your own HTTP server.

## Main Reusable Packages

- `recipe-extractor/server/core`
  - opens the database
  - runs migrations
  - builds the store and worker
  - exposes service-style methods for recipe operations
- `recipe-extractor/server/httpapi`
  - exposes the existing HTTP API as an `http.Handler`
  - can be mounted inside your own router

## Option 1: Let `recipe-extractor` Open The Database

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"

	recipecore "recipe-extractor/server/core"
	recipehttp "recipe-extractor/server/httpapi"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags|log.LUTC)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	app, err := recipecore.Open(ctx, recipecore.Config{
		DatabaseURL:          "postgres://postgres:postgres@localhost:5433/recipes?sslmode=disable",
		Extractor:            "openai",
		OpenAIAPIKey:         os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:          "gpt-5-mini",
		OpenAIBaseURL:        "https://api.openai.com/v1",
		OpenAITimeoutSeconds: 45,
	}, logger)
	if err != nil {
		logger.Fatal(err)
	}
	defer app.Close()

	go app.RunWorker(context.Background())

	r := chi.NewRouter()
	r.Mount("/recipe-extractor", recipehttp.NewHandler(recipehttp.Config{}, app, logger))

	logger.Fatal(http.ListenAndServe(":8090", r))
}
```

## Option 2: Use Your Own Database Pool

If your app already manages the PostgreSQL connection pool, run migrations first and then build the reusable app around your existing pool.

```go
pool, err := pgxpool.New(ctx, databaseURL)
if err != nil {
	return err
}

if err := recipecore.Migrate(ctx, pool, logger); err != nil {
	return err
}

app, err := recipecore.New(pool, recipecore.Config{
	DatabaseURL:          databaseURL,
	Extractor:            "anthropic",
	AnthropicAPIKey:      os.Getenv("ANTHROPIC_API_KEY"),
	AnthropicModel:       "claude-sonnet-4-6",
	AnthropicTimeoutSeconds: 45,
}, logger)
if err != nil {
	return err
}
```

`core.New` does not own the pool you pass in, so your app remains responsible for closing it.

## Service Methods

The `core.App` type exposes reusable operations that do not require the built-in HTTP API:

- `CreateRecipeExtraction`
- `ListRecipes`
- `GetRecipe`
- `GetRecipeExtraction`
- `DeleteRecipe`
- `RunWorker`

Example:

```go
detail, err := app.GetRecipe(ctx, recipeID)
if err != nil {
	if recipecore.IsNotFound(err) {
		// handle 404 equivalent
	}
	return err
}

log.Printf("loaded recipe %s with %d related recipes", detail.Recipe.Title, len(detail.RelatedRecipes))
```

## HTTP Integration

`httpapi.NewHandler(...)` returns an `http.Handler`, so you can:

- serve it as your whole server
- mount it under a prefix in your own router
- put your own auth, rate limiting, or middleware in front of it

For hosted products, this is usually the cleanest setup:

- your app owns the main router
- your app decides which middleware and auth apply
- `recipe-extractor` provides extraction/storage behavior and optionally its API endpoints

## Notes

- `cmd/server` is still the easiest way to run the standalone app.
- The reusable packages do not require `.env`; pass config explicitly from your own app.
- The embedded frontend and Vite dev proxy behavior only apply when using the standalone HTTP shell or `httpapi` with that config enabled.
