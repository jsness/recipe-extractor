package store

import (
	"database/sql"
	"encoding/json"
)

func nullableStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	out := v.String
	return &out
}

func nullableJSONString(v []byte) *string {
	if len(v) == 0 {
		return nil
	}
	out := string(v)
	return &out
}

func marshalRecipeInput(input RecipeInput) (string, string, *string, string, error) {
	ingredientsJSON, err := json.Marshal(input.Ingredients)
	if err != nil {
		return "", "", nil, "", err
	}

	instructionsJSON, err := json.Marshal(input.Instructions)
	if err != nil {
		return "", "", nil, "", err
	}

	var timesJSON []byte
	if input.Times != nil {
		timesJSON, err = json.Marshal(input.Times)
		if err != nil {
			return "", "", nil, "", err
		}
	}

	linkedURLs := input.LinkedRecipeURLs
	if linkedURLs == nil {
		linkedURLs = []string{}
	}
	linkedURLsJSON, err := json.Marshal(linkedURLs)
	if err != nil {
		return "", "", nil, "", err
	}

	return string(ingredientsJSON), string(instructionsJSON), nullableJSONString(timesJSON), string(linkedURLsJSON), nil
}

func decodeRecipeRow(
	r *Recipe,
	ingredientsRaw, instructionsRaw, timesRaw, linkedURLsRaw []byte,
	yield, notes sql.NullString,
) error {
	if err := json.Unmarshal(ingredientsRaw, &r.Ingredients); err != nil {
		return err
	}
	if err := json.Unmarshal(instructionsRaw, &r.Instructions); err != nil {
		return err
	}
	if len(timesRaw) > 0 {
		if err := json.Unmarshal(timesRaw, &r.Times); err != nil {
			return err
		}
	}
	if len(linkedURLsRaw) > 0 {
		if err := json.Unmarshal(linkedURLsRaw, &r.LinkedRecipeURLs); err != nil {
			return err
		}
	}

	r.Yield = nullableStringPtr(yield)
	r.Notes = nullableStringPtr(notes)
	return nil
}
