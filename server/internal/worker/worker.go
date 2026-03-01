package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"recipe-extractor/server/internal/config"
	"recipe-extractor/server/internal/extractor"
	"recipe-extractor/server/internal/scraper"
	"recipe-extractor/server/internal/store"
)

type Worker struct {
	store        *store.Store
	scraper      *scraper.Scraper
	extractor    extractor.Extractor
	logger       *log.Logger
	pollInterval time.Duration
}

func New(cfg config.Config, s *store.Store, logger *log.Logger) *Worker {
	var (
		ext     extractor.Extractor
		timeout time.Duration
	)
	if cfg.Extractor == "anthropic" {
		timeout = time.Duration(cfg.AnthropicTimeoutSeconds) * time.Second
		ext = extractor.NewAnthropic(extractor.AnthropicConfig{
			APIKey:  cfg.AnthropicAPIKey,
			Model:   cfg.AnthropicModel,
			Timeout: timeout,
		})
	} else {
		timeout = time.Duration(cfg.OpenAITimeoutSeconds) * time.Second
		ext = extractor.NewOpenAI(extractor.OpenAIConfig{
			APIKey:         cfg.OpenAIAPIKey,
			Model:          cfg.OpenAIModel,
			BaseURL:        cfg.OpenAIBaseURL,
			ProjectID:      cfg.OpenAIProjectID,
			OrganizationID: cfg.OpenAIOrganizationID,
			Timeout:        timeout,
		})
	}
	return &Worker{
		store:        s,
		scraper:      scraper.New(timeout),
		extractor:    ext,
		logger:       logger,
		pollInterval: 2 * time.Second,
	}
}

func (w *Worker) Run(ctx context.Context) {
	w.process(ctx)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.process(ctx)
		}
	}
}

func (w *Worker) process(ctx context.Context) {
	for {
		extraction, err := w.store.ClaimNextQueuedRecipeExtraction(ctx)
		if err != nil {
			w.logger.Printf("claim queued extraction: %v", err)
			return
		}
		if extraction == nil {
			return
		}

		w.logger.Printf("processing extraction id=%s url=%s", extraction.ID, extraction.SourceURL)

		if err := w.processExtraction(ctx, extraction); err != nil {
			errorMessage := err.Error()
			if updateErr := w.store.UpdateRecipeExtractionStatus(ctx, extraction.ID, "failed", nil, &errorMessage); updateErr != nil {
				w.logger.Printf("mark extraction failed id=%s: %v", extraction.ID, updateErr)
				return
			}
			w.logger.Printf("extraction failed id=%s: %v", extraction.ID, err)
		}
	}
}

func (w *Worker) processExtraction(ctx context.Context, extraction *store.RecipeExtraction) error {
	scrapeResult, err := w.scraper.Fetch(ctx, extraction.SourceURL)
	if err != nil {
		return fmt.Errorf("scrape: %w", err)
	}

	normalizedRecipe, err := w.extractor.NormalizeRecipe(ctx, extractor.Input{
		SourceURL: scrapeResult.SourceURL,
		JSONLD:    scrapeResult.JSONLD,
		Text:      scrapeResult.Text,
		Links:     scrapeResult.Links,
	})
	if err != nil {
		return fmt.Errorf("normalize recipe: %w", err)
	}

	ingredients := make([]store.IngredientGroup, len(normalizedRecipe.Ingredients))
	for i, g := range normalizedRecipe.Ingredients {
		ingredients[i] = store.IngredientGroup{Group: g.Group, Items: g.Items}
	}

	recipeID, err := w.store.UpsertRecipe(ctx, store.RecipeInput{
		Title:            normalizedRecipe.Title,
		Ingredients:      ingredients,
		Instructions:     normalizedRecipe.Instructions,
		Yield:            normalizedRecipe.Yield,
		Times:            normalizedRecipe.Times,
		Notes:            normalizedRecipe.Notes,
		SourceURL:        extraction.SourceURL,
		LinkedRecipeURLs: normalizedRecipe.LinkedRecipeURLs,
	})
	if err != nil {
		return fmt.Errorf("store recipe: %w", err)
	}

	for _, linkedURL := range normalizedRecipe.LinkedRecipeURLs {
		if err := w.store.QueueLinkedRecipeExtraction(ctx, recipeID, linkedURL); err != nil {
			w.logger.Printf("queue linked recipe url=%s: %v", linkedURL, err)
		}
	}

	if extraction.ParentRecipeID != nil {
		if err := w.store.CreateRecipeRelationship(ctx, *extraction.ParentRecipeID, recipeID); err != nil {
			w.logger.Printf("create recipe relationship parent=%s child=%s: %v", *extraction.ParentRecipeID, recipeID, err)
		}
	}

	if err := w.store.UpdateRecipeExtractionStatus(ctx, extraction.ID, "done", &recipeID, nil); err != nil {
		return fmt.Errorf("mark done: %w", err)
	}
	return nil
}
