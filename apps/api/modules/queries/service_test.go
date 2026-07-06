package queries

import (
	"testing"

	"github.com/FacileStudio/Journal/apps/api/schemas"
)

func TestValidateParams(t *testing.T) {
	cases := []struct {
		name   string
		params schemas.SavedQueryParams
		valid  bool
	}{
		{"empty params", schemas.SavedQueryParams{}, true},
		{"all fields", schemas.SavedQueryParams{App: "nuage", Levels: []string{"error", "warn"}, Q: "boom", RequestID: "abc"}, true},
		{"every valid level", schemas.SavedQueryParams{Levels: []string{"debug", "info", "warn", "error"}}, true},
		{"unknown level", schemas.SavedQueryParams{Levels: []string{"fatal"}}, false},
		{"empty level", schemas.SavedQueryParams{Levels: []string{""}}, false},
		{"mixed valid and invalid", schemas.SavedQueryParams{Levels: []string{"error", "trace"}}, false},
		{"uppercase level", schemas.SavedQueryParams{Levels: []string{"ERROR"}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateParams(tc.params)
			if tc.valid && err != nil {
				t.Fatalf("validateParams(%+v) = %v, want nil", tc.params, err)
			}
			if !tc.valid && err == nil {
				t.Fatalf("validateParams(%+v) = nil, want error", tc.params)
			}
		})
	}
}
