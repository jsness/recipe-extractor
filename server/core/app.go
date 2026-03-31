package core

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	migratelite "github.com/jsness/go-migrate-lite"

	"recipe-extractor/server/internal/db"
	"recipe-extractor/server/migrations"
	"recipe-extractor/server/store"
	"recipe-extractor/server/wayback"
	"recipe-extractor/server/worker"
)

var (
	ErrRecipeAlreadyExtracted     = errors.New("recipe already extracted")
	ErrRecipeExtractionInProgress = errors.New("recipe extraction in progress")
)

type RecipeDetail struct {
	Recipe         store.Recipe
	RelatedRecipes []store.RelatedRecipe
}

type App struct {
	pool     *pgxpool.Pool
	ownsPool bool
	store    *store.Store
	worker   *worker.Worker
	wayback  *wayback.Client
}

func Open(ctx context.Context, cfg Config, logger *log.Logger) (*App, error) {
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	if err := Migrate(ctx, pool, logger); err != nil {
		pool.Close()
		return nil, err
	}

	app, err := New(pool, cfg, logger)
	if err != nil {
		pool.Close()
		return nil, err
	}
	app.ownsPool = true
	return app, nil
}

func New(pool *pgxpool.Pool, cfg Config, logger *log.Logger) (*App, error) {
	s := store.New(pool)
	w, err := worker.New(cfg.workerConfig(), s, logger)
	if err != nil {
		return nil, err
	}

	return &App{
		pool:    pool,
		store:   s,
		worker:  w,
		wayback: wayback.New(10 * time.Second),
	}, nil
}

func Migrate(ctx context.Context, pool *pgxpool.Pool, logger *log.Logger) error {
	if logger != nil {
		logger.Printf("running migrations")
	}
	return migratelite.Run(ctx, pool, migrations.SQL)
}

func (a *App) Close() {
	if a.ownsPool && a.pool != nil {
		a.pool.Close()
	}
}

func (a *App) Store() *store.Store {
	return a.store
}

func (a *App) Worker() *worker.Worker {
	return a.worker
}

func (a *App) RunWorker(ctx context.Context) {
	a.worker.Run(ctx)
}

func (a *App) ListProfiles(ctx context.Context) ([]store.Profile, error) {
	return a.store.ListProfiles(ctx)
}

func (a *App) CreateProfile(ctx context.Context, name string) (store.Profile, error) {
	return a.store.CreateProfile(ctx, name)
}

func (a *App) CreateRecipeExtraction(ctx context.Context, profileID, sourceURL string) (store.RecipeExtraction, error) {
	existing, err := a.store.GetRecipeExtractionBySourceURL(ctx, profileID, sourceURL)
	if err != nil {
		return store.RecipeExtraction{}, err
	}
	if existing != nil {
		switch existing.Status {
		case "done":
			return store.RecipeExtraction{}, ErrRecipeAlreadyExtracted
		case "queued", "extracting":
			return store.RecipeExtraction{}, ErrRecipeExtractionInProgress
		}
	}

	return a.store.CreateRecipeExtraction(ctx, profileID, sourceURL)
}

func (a *App) ListRecipes(ctx context.Context, profileID string) ([]store.RecipeSummary, error) {
	return a.store.ListRecipes(ctx, profileID)
}

func (a *App) GetRecipe(ctx context.Context, profileID, id string) (RecipeDetail, error) {
	recipe, err := a.store.GetRecipeByID(ctx, profileID, id)
	if err != nil {
		return RecipeDetail{}, err
	}

	related, err := a.store.GetRelatedRecipes(ctx, profileID, id)
	if err != nil {
		return RecipeDetail{}, err
	}

	return RecipeDetail{
		Recipe:         recipe,
		RelatedRecipes: related,
	}, nil
}

func (a *App) GetRecipeExtraction(ctx context.Context, profileID, id string) (store.RecipeExtraction, error) {
	return a.store.GetRecipeExtractionByID(ctx, profileID, id)
}

func (a *App) DeleteRecipe(ctx context.Context, profileID, id string) (bool, error) {
	return a.store.DeleteRecipe(ctx, profileID, id)
}

func (a *App) FindArchivedSnapshot(ctx context.Context, sourceURL string) (*wayback.Snapshot, error) {
	return a.wayback.Lookup(ctx, sourceURL)
}

func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
