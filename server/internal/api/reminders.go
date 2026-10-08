package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jsness/recipe-extractor/server/core"
)

func validUUID(value string) bool {
	var id pgtype.UUID
	return id.Scan(value) == nil && id.Valid
}

func decodeReminder(r *http.Request) (string, error) {
	var req struct {
		Reminder *string `json:"reminder"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return "", err
	}
	if req.Reminder == nil {
		return "", errors.New("reminder must be a string")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return "", errors.New("expected a single JSON object")
	}
	return *req.Reminder, nil
}

func (h *Handler) handleUpdateRecipeReminder(w http.ResponseWriter, r *http.Request) {
	profileID, ok := profileIDFromRequest(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if !validUUID(id) || !validUUID(profileID) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Recipe and profile IDs must be valid UUIDs."})
		return
	}
	reminder, err := decodeReminder(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Provide a JSON object with a reminder string."})
		return
	}
	detail, err := h.app.UpdateRecipeReminder(r.Context(), profileID, id, reminder)
	if err != nil {
		if core.IsNotFound(err) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Recipe not found."})
			return
		}
		h.logger.Printf("update recipe reminder %s: %v", id, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Unable to save reminder."})
		return
	}
	writeJSON(w, http.StatusOK, newRecipeResponse(detail.Recipe, detail.RelatedRecipes))
}
