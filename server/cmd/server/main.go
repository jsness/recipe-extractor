package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"recipe-extractor/server/core"
	"recipe-extractor/server/httpapi"
	"recipe-extractor/server/internal/config"
)

func main() {
	cfg := config.LoadFromEnv()
	logger := log.New(os.Stdout, "", log.LstdFlags|log.LUTC)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	app, err := core.Open(ctx, core.Config{
		DatabaseURL:             cfg.DatabaseURL,
		Extractor:               cfg.Extractor,
		LLMOnlyExtraction:       cfg.LLMOnlyExtraction,
		OpenAIAPIKey:            cfg.OpenAIAPIKey,
		OpenAIModel:             cfg.OpenAIModel,
		OpenAIBaseURL:           cfg.OpenAIBaseURL,
		OpenAIProjectID:         cfg.OpenAIProjectID,
		OpenAIOrganizationID:    cfg.OpenAIOrganizationID,
		OpenAITimeoutSeconds:    cfg.OpenAITimeoutSeconds,
		AnthropicAPIKey:         cfg.AnthropicAPIKey,
		AnthropicModel:          cfg.AnthropicModel,
		AnthropicTimeoutSeconds: cfg.AnthropicTimeoutSeconds,
	}, logger)
	if err != nil {
		logger.Fatalf("app startup: %v", err)
	}
	defer app.Close()

	go app.RunWorker(context.Background())

	h := httpapi.NewHandler(httpapi.Config{
		FrontendDevProxyURL: cfg.FrontendDevProxyURL,
	}, app, logger)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Printf("listening on %s", cfg.HTTPAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("server: %v", err)
	}
}
