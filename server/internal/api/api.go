package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"recipe-extractor/server/internal/config"
	"recipe-extractor/server/internal/store"
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

	return r
}

type createRecipeRequest struct {
	URL string `json:"url"`
}

type createRecipeResponse struct {
	ExtractionID string `json:"extraction_id"`
	Status       string `json:"status"`
}

type recipeResponse struct {
	ID           string                 `json:"id"`
	Title        string                 `json:"title"`
	Ingredients  []store.IngredientGroup `json:"ingredients"`
	Instructions []string               `json:"instructions"`
	Yield        *string                `json:"yield,omitempty"`
	Times        map[string]string      `json:"times,omitempty"`
	Notes        *string                `json:"notes,omitempty"`
	SourceURL    string                 `json:"source_url"`
	CreatedAt    time.Time              `json:"created_at"`
}

func recipeToResponse(r store.Recipe) recipeResponse {
	return recipeResponse{
		ID:           r.ID,
		Title:        r.Title,
		Ingredients:  r.Ingredients,
		Instructions: r.Instructions,
		Yield:        r.Yield,
		Times:        r.Times,
		Notes:        r.Notes,
		SourceURL:    r.SourceURL,
		CreatedAt:    r.CreatedAt,
	}
}

type getRecipeExtractionResponse struct {
	ID           string  `json:"id"`
	SourceURL    string  `json:"source_url"`
	Status       string  `json:"status"`
	RecipeID     *string `json:"recipe_id,omitempty"`
	ErrorMessage *string `json:"error_message,omitempty"`
}

func (h *Handler) handleCreateRecipe(w http.ResponseWriter, r *http.Request) {
	var req createRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		msg := "This URL has already been extracted."
		if existing.Status == "queued" || existing.Status == "extracting" {
			msg = "This URL is already being extracted."
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
		return
	}

	extraction, err := h.store.CreateRecipeExtraction(r.Context(), req.URL)
	if err != nil {
		h.logger.Printf("create extraction: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(createRecipeResponse{
		ExtractionID: extraction.ID,
		Status:       extraction.Status,
	})
}

type recipeSummaryResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
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

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
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

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(recipeToResponse(recipe))
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

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(getRecipeExtractionResponse{
		ID:           extraction.ID,
		SourceURL:    extraction.SourceURL,
		Status:       extraction.Status,
		RecipeID:     extraction.RecipeID,
		ErrorMessage: extraction.ErrorMessage,
	})
}
