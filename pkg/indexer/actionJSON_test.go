package indexer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActionJSON_Stream(t *testing.T) {
	ad := &ActionDispatcher{}
	// ActionDispatcher.RegisterAction wird von NewActionJSON aufgerufen.
	// Da ad ein Pointer auf ein leeres Struct ist, wird RegisterAction fehlschlagen,
	// wenn es auf nil-Maps zugreift. Wir initialisieren den Dispatcher daher minimal.
	ad.actions = make(map[string]Action)

	formats := map[string]ConfigJSONFormat{
		"csl": {
			MandatoryFields: []string{"id", "type"},
			OptionalFields:  []string{"author", "title"},
			NumOptionals:    0,
			Mime:            "application/vnd.citationstyles.style+json",
			Pronom:          "fmt/csl",
			Type:            "text",
			Subtype:         "csl",
		},
		"complex": {
			MandatoryFields: []string{"metadata/id", "data/items"},
			OptionalFields:  []string{"version"},
			NumOptionals:    1,
			Mime:            "application/x-complex-json",
			Pronom:          "fmt/complex",
			Type:            "text",
			Subtype:         "complex",
		},
	}

	// NewActionJSON normalisiert Felder intern auf Kleinschreibung
	action := NewActionJSON("test-json", formats, ad)
	aj := action.(*ActionJSON)

	tests := []struct {
		name        string
		json        string
		wantMime    string
		wantPronom  string
		wantErr     bool
		errContains string
	}{
		{
			name:       "valid csl simple",
			json:       `{"id": "123", "type": "article"}`,
			wantMime:   "application/vnd.citationstyles.style+json",
			wantPronom: "fmt/csl",
			wantErr:    false,
		},
		{
			name:       "valid csl with optional",
			json:       `{"id": "123", "type": "article", "author": "John Doe"}`,
			wantMime:   "application/vnd.citationstyles.style+json",
			wantPronom: "fmt/csl",
			wantErr:    false,
		},
		{
			name:        "invalid csl - missing mandatory",
			json:        `{"id": "123"}`,
			wantErr:     true,
			errContains: "no matching JSON format found",
		},
		{
			name:       "complex nested valid",
			json:       `{"metadata": {"id": "meta-1"}, "data": {"items": [1,2,3]}, "version": "1.0"}`,
			wantMime:   "application/x-complex-json",
			wantPronom: "fmt/complex",
			wantErr:    false,
		},
		{
			name:        "complex nested - missing optional",
			json:        `{"metadata": {"id": "meta-1"}, "data": {"items": [1,2,3]}}`,
			wantErr:     true,
			errContains: "no matching JSON format found",
		},
		{
			name:        "completely unrelated json",
			json:        `{"foo": "bar"}`,
			wantErr:     true,
			errContains: "no matching JSON format found",
		},
		{
			name:        "invalid json syntax",
			json:        `{"id": "123", "type": "article", "incomplete": [}`,
			wantErr:     true,
			errContains: "error extracting JSON fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.json)
			result, err := aj.Stream("application/json", reader, "test.json")

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.wantMime, result.Mimetype)
				assert.Equal(t, tt.wantPronom, result.Pronom)
				// Check if metadata contains identified fields
				assert.Contains(t, result.Metadata, "test-json")
			}
		})
	}
}
