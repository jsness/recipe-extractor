package api

import (
	"errors"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"recipe-extractor/server/core"
	"recipe-extractor/server/internal/frontend"
)

type Config struct {
	FrontendDevProxyURL string
}

type Handler struct {
	cfg    Config
	app    *core.App
	logger *log.Logger
}

func NewHandler(cfg Config, app *core.App, logger *log.Logger) *Handler {
	return &Handler{cfg: cfg, app: app, logger: logger}
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
		r.Delete("/recipes/{id}", h.handleDeleteRecipe)
		r.Get("/recipe-extractions/{id}", h.handleGetRecipeExtraction)
	})

	if h.cfg.FrontendDevProxyURL != "" {
		proxy, err := newFrontendDevProxy(h.cfg.FrontendDevProxyURL)
		if err != nil {
			h.logger.Printf("frontend dev proxy disabled: %v", err)
		} else {
			h.logger.Printf("proxying frontend requests to %s", h.cfg.FrontendDevProxyURL)
			r.Handle("/*", proxy)
			return r
		}
	}

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

func newFrontendDevProxy(rawURL string) (http.Handler, error) {
	target, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "frontend dev server unavailable", http.StatusBadGateway)
	}

	return proxy, nil
}

func (h *Handler) handleCreateRecipe(w http.ResponseWriter, r *http.Request) {
	var req createRecipeRequest
	if err := decodeJSON(r, &req); err != nil || req.URL == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	extraction, err := h.app.CreateRecipeExtraction(r.Context(), req.URL)
	if err != nil {
		if errors.Is(err, core.ErrRecipeAlreadyExtracted) || errors.Is(err, core.ErrRecipeExtractionInProgress) {
			msg := "This URL has already been extracted."
			if errors.Is(err, core.ErrRecipeExtractionInProgress) {
				msg = "This URL is already being extracted."
			}
			writeJSON(w, http.StatusConflict, map[string]string{"error": msg})
			return
		}
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
	recipes, err := h.app.ListRecipes(r.Context())
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

	detail, err := h.app.GetRecipe(r.Context(), id)
	if err != nil {
		if core.IsNotFound(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		h.logger.Printf("get recipe %s: %v", id, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, newRecipeResponse(detail.Recipe, detail.RelatedRecipes))
}

func (h *Handler) handleDeleteRecipe(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	deleted, err := h.app.DeleteRecipe(r.Context(), id)
	if err != nil {
		h.logger.Printf("delete recipe %s: %v", id, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !deleted {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleGetRecipeExtraction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	extraction, err := h.app.GetRecipeExtraction(r.Context(), id)
	if err != nil {
		if core.IsNotFound(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		h.logger.Printf("get extraction %s: %v", id, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, newRecipeExtractionResponse(extraction))
}
