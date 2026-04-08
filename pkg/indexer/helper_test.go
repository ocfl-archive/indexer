package indexer

import (
	"reflect"
	"strings"
	"testing"
)

func TestExtractJSONFields(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:  "Flaches Objekt",
			input: `{"a": 1, "b": "test", "c": true}`,
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "Verschachteltes Objekt",
			input: `{"a": {"b": {"c": 3}}}`,
			want:  []string{"a", "a/b", "a/b/c"},
		},
		{
			name:  "Array mit Objekten",
			input: `{"authors": [{"family": "Einstein", "given": "Albert"}, {"family": "Newton"}]}`,
			want:  []string{"authors", "authors/family", "authors/given"},
		},
		{
			name:  "Leere Container",
			input: `{"empty_obj": {}, "empty_arr": []}`,
			want:  []string{"empty_arr", "empty_obj"},
		},
		{
			name:  "Gemischte Typen",
			input: `{"a": [1, 2, {"b": 3}], "c": "d"}`,
			want:  []string{"a", "a/b", "c"},
		},
		{
			name:  "Mehrere Ebenen mit Arrays",
			input: `{"x": [{"y": [{"z": 1}]}]}`,
			want:  []string{"x", "x/y", "x/y/z"},
		},
		{
			name:    "Ungültiges JSON",
			input:   `{"a": }`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			got, err := ExtractJSONFields(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractJSONFields() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractJSONFields() got = %v, want %v", got, tt.want)
			}
		})
	}
}
