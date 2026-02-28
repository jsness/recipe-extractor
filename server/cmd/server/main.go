package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"recipe-extractor/server/internal/api"
	"recipe-extractor/server/internal/config"
	"recipe-extractor/server/internal/db"
	"recipe-extractor/server/internal/store"
	"recipe-extractor/server/internal/worker"
)

func main() {
	cfg := config.LoadFromEnv()
	logger := log.New(os.Stdout, "", log.LstdFlags|log.LUTC)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	s := store.New(pool)
	h := api.NewHandler(cfg, s, logger)
	w := worker.New(cfg, s, logger)

	go w.Run(context.Background())

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           h.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Printf("listening on %s", cfg.HTTPAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("server: %v", err)
	}
}
