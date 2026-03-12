package api

import (
	"errors"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"recipe-extractor/server/internal/config"
	"recipe-extractor/server/internal/frontend"
	"recipe-extractor/server/store"
)

type Handler struct {
	cfg    config.Config
	store  *store.Store
	logger *log.Logger
}

func NewHandler(cfg config.Config, s *store.Store, logger *log.Logger) *Handler {
	return &Handler{cfg: cfg, store: s, logger: logger}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/recipes", h.handleCreateRecipe)
		r.Get("/recipes", h.handleListRecipes)
		r.Get("/recipes/{id}", h.handleGetRecipe)
		r.Get("/recipe-extractions/{id}", h.handleGetRecipeExtraction)
	})

	// Serve embedded frontend (SPA). Falls back to index.html for client-side routing.
	// Skipped gracefully if no frontend is embedded (local dev uses Vite on :5173).
	if _, err := frontend.FS.Open("dist/index.html"); err == nil {
		distFS, _ := fs.Sub(frontend.FS, "dist")
		fileServer := http.FileServer(http.FS(distFS))
		r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}
			if _, err := distFS.Open(path); err != nil {
				r.URL.Path = "/"
			}
			fileServer.ServeHTTP(w, r)
		}))
	}

	return r
}

func (h *Handler) handleCreateRecipe(w http.ResponseWriter, r *http.Request) {
	var req createRecipeRequest
	if err := decodeJSON(r, &req); err != nil || req.URL == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	existing, err := h.store.GetRecipeExtractionBySourceURL(r.Context(), req.URL)
	if err != nil {
		h.logger.Printf("lookup extraction by url: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if existing != nil && (existing.Status == "done" || existing.Status == "queued" || existing.Status == "extracting") {
		msg := "This URL has already been extracted."
		if existing.Status == "queued" || existing.Status == "extracting" {
			msg = "This URL is already being extracted."
		}
		writeJSON(w, http.StatusConflict, map[string]string{"error": msg})
		return
	}

	extraction, err := h.store.CreateRecipeExtraction(r.Context(), req.URL)
	if err != nil {
		h.logger.Printf("create extraction: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusAccepted, createRecipeResponse{
		ExtractionID: extraction.ID,
		Status:       extraction.Status,
	})
}

func (h *Handler) handleListRecipes(w http.ResponseWriter, r *http.Request) {
	recipes, err := h.store.ListRecipes(r.Context())
	if err != nil {
		h.logger.Printf("list recipes: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp := make([]recipeSummaryResponse, len(recipes))
	for i, r := range recipes {
		resp[i] = recipeSummaryResponse{ID: r.ID, Title: r.Title}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleGetRecipe(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	recipe, err := h.store.GetRecipeByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		h.logger.Printf("get recipe %s: %v", id, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	related, err := h.store.GetRelatedRecipes(r.Context(), id)
	if err != nil {
		h.logger.Printf("get related recipes %s: %v", id, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, newRecipeResponse(recipe, related))
}

func (h *Handler) handleGetRecipeExtraction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	extraction, err := h.store.GetRecipeExtractionByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		h.logger.Printf("get extraction %s: %v", id, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, newRecipeExtractionResponse(extraction))
}
