package store

import (
	"database/sql"
	"reflect"
	"testing"
)

func TestMarshalRecipeInput(t *testing.T) {
	tests := []struct {
		name             string
		input            RecipeInput
		wantIngredients  string
		wantInstructions string
		wantTimes        *string
		wantLinkedURLs   string
	}{
		{
			name: "complete recipe input",
			input: RecipeInput{
				Ingredients:      []IngredientGroup{{Group: "Sauce", Items: []string{"tomato", "salt"}}},
				Instructions:     []string{"Stir.", "Simmer."},
				Times:            map[string]string{"Cook Time": "20 minutes"},
				LinkedRecipeURLs: []string{"https://example.com/sauce"},
			},
			wantIngredients:  `[{"group":"Sauce","items":["tomato","salt"]}]`,
			wantInstructions: `["Stir.","Simmer."]`,
			wantTimes:        stringPtr(`{"Cook Time":"20 minutes"}`),
			wantLinkedURLs:   `["https://example.com/sauce"]`,
		},
		{
			name: "nil times and linked urls",
			input: RecipeInput{
				Ingredients:  []IngredientGroup{},
				Instructions: []string{},
			},
			wantIngredients:  `[]`,
			wantInstructions: `[]`,
			wantLinkedURLs:   `[]`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotIngredients, gotInstructions, gotTimes, gotLinkedURLs, err := marshalRecipeInput(tc.input)
			if err != nil {
				t.Fatalf("marshalRecipeInput() error = %v, want nil", err)
			}
			if gotIngredients != tc.wantIngredients {
				t.Fatalf("ingredients = %q, want %q", gotIngredients, tc.wantIngredients)
			}
			if gotInstructions != tc.wantInstructions {
				t.Fatalf("instructions = %q, want %q", gotInstructions, tc.wantInstructions)
			}
			if !reflect.DeepEqual(gotTimes, tc.wantTimes) {
				t.Fatalf("times = %#v, want %#v", gotTimes, tc.wantTimes)
			}
			if gotLinkedURLs != tc.wantLinkedURLs {
				t.Fatalf("linked URLs = %q, want %q", gotLinkedURLs, tc.wantLinkedURLs)
			}
		})
	}
}

func TestDecodeRecipeRow(t *testing.T) {
	tests := []struct {
		name            string
		ingredientsRaw  []byte
		instructionsRaw []byte
		timesRaw        []byte
		linkedURLsRaw   []byte
		yield           sql.NullString
		notes           sql.NullString
		want            Recipe
		wantErr         bool
	}{
		{
			name:            "complete row",
			ingredientsRaw:  []byte(`[{"group":"Sauce","items":["tomato"]}]`),
			instructionsRaw: []byte(`["Stir."]`),
			timesRaw:        []byte(`{"Cook Time":"20 minutes"}`),
			linkedURLsRaw:   []byte(`["https://example.com/sauce"]`),
			yield:           sql.NullString{String: "2 servings", Valid: true},
			notes:           sql.NullString{String: "Warm", Valid: true},
			want: Recipe{
				Ingredients:      []IngredientGroup{{Group: "Sauce", Items: []string{"tomato"}}},
				Instructions:     []string{"Stir."},
				Times:            map[string]string{"Cook Time": "20 minutes"},
				LinkedRecipeURLs: []string{"https://example.com/sauce"},
				Yield:            stringPtr("2 servings"),
				Notes:            stringPtr("Warm"),
			},
		},
		{
			name:            "empty optional json and null strings",
			ingredientsRaw:  []byte(`[]`),
			instructionsRaw: []byte(`[]`),
			want: Recipe{
				Ingredients:  []IngredientGroup{},
				Instructions: []string{},
			},
		},
		{
			name:            "invalid ingredients json",
			ingredientsRaw:  []byte(`{`),
			instructionsRaw: []byte(`[]`),
			wantErr:         true,
		},
		{
			name:            "invalid instructions json",
			ingredientsRaw:  []byte(`[]`),
			instructionsRaw: []byte(`{`),
			wantErr:         true,
		},
		{
			name:            "invalid times json",
			ingredientsRaw:  []byte(`[]`),
			instructionsRaw: []byte(`[]`),
			timesRaw:        []byte(`{`),
			wantErr:         true,
		},
		{
			name:            "invalid linked urls json",
			ingredientsRaw:  []byte(`[]`),
			instructionsRaw: []byte(`[]`),
			linkedURLsRaw:   []byte(`{`),
			wantErr:         true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got Recipe
			err := decodeRecipeRow(
				&got,
				tc.ingredientsRaw,
				tc.instructionsRaw,
				tc.timesRaw,
				tc.linkedURLsRaw,
				tc.yield,
				tc.notes,
			)
			if tc.wantErr {
				if err == nil {
					t.Fatal("decodeRecipeRow() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("decodeRecipeRow() error = %v, want nil", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("decodeRecipeRow() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestNullableStringPtr(t *testing.T) {
	tests := []struct {
		name string
		in   sql.NullString
		want *string
	}{
		{name: "valid", in: sql.NullString{String: "value", Valid: true}, want: stringPtr("value")},
		{name: "null", in: sql.NullString{String: "ignored", Valid: false}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := nullableStringPtr(tc.in); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("nullableStringPtr() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
