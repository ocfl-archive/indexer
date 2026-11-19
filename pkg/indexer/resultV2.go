package indexer

import "slices"

// ResultV2 represents the unified metadata and format identification output of the indexer.
type ResultV2 struct {
	// Errors contains error messages indexed by action name.
	Errors map[string]string `json:"errors,omitempty"`
	// Mimetype is the resolved primary MIME type after weight sorting.
	Mimetype string `json:"mimetype"`
	// Mimetypes lists all candidate MIME types detected across actions.
	Mimetypes []string `json:"mimetypes"`
	// Pronom is the resolved primary PRONOM PUID (e.g. "fmt/40").
	Pronom string `json:"pronom"`
	// Pronoms lists all detected PRONOM PUIDs.
	Pronoms []string `json:"pronoms"`
	// Checksum maps digest algorithm names to lowercase hex digest strings.
	Checksum map[string]string `json:"checksum,omitempty"`
	// Width is the image or video width in pixels.
	Width uint `json:"width,omitempty"`
	// Height is the image or video height in pixels.
	Height uint `json:"height,omitempty"`
	// Duration is the audio or video duration in seconds.
	Duration uint `json:"duration,omitempty"`
	// Size is the total file/stream size in bytes.
	Size uint64 `json:"size"`
	// Metadata holds arbitrary structured metadata contributed by actions.
	Metadata map[string]any `json:"metadata"`
	// Type represents the high-level category (e.g. "image", "video", "ocr").
	Type string `json:"type"`
	// Subtype represents the specific format name or schema subtype.
	Subtype string `json:"subtype"`
}

// NewResultV2 instantiates a new empty ResultV2 with initialized maps and slices.
func NewResultV2() *ResultV2 {
	return &ResultV2{
		Errors:    map[string]string{},
		Mimetypes: []string{},
		Pronoms:   []string{},
		Checksum:  map[string]string{},
		Metadata:  map[string]any{},
	}
}

// GetMetadata retrieves a typed metadata entry from ResultV2 safely.
func GetMetadata[T any](r *ResultV2, key string) (T, bool) {
	if r == nil || r.Metadata == nil {
		var zero T
		return zero, false
	}
	val, ok := r.Metadata[key]
	if !ok {
		var zero T
		return zero, false
	}
	res, ok := val.(T)
	return res, ok
}

// SetMetadata stores a typed metadata entry into ResultV2.
func SetMetadata[T any](r *ResultV2, key string, val T) {
	if r == nil {
		return
	}
	if r.Metadata == nil {
		r.Metadata = make(map[string]any)
	}
	r.Metadata[key] = val
}

func (v *ResultV2) Merge(r *ResultV2) {
	if r == nil {
		return
	}
	if len(r.Mimetypes) > 0 {
		if cap(v.Mimetypes)-len(v.Mimetypes) < len(r.Mimetypes) {
			newMimes := make([]string, len(v.Mimetypes), len(v.Mimetypes)+len(r.Mimetypes))
			copy(newMimes, v.Mimetypes)
			v.Mimetypes = newMimes
		}
		v.Mimetypes = append(v.Mimetypes, r.Mimetypes...)
		slices.Sort(v.Mimetypes)
		v.Mimetypes = slices.Compact(v.Mimetypes)
	}
	if len(r.Pronoms) > 0 {
		if cap(v.Pronoms)-len(v.Pronoms) < len(r.Pronoms) {
			newPronoms := make([]string, len(v.Pronoms), len(v.Pronoms)+len(r.Pronoms))
			copy(newPronoms, v.Pronoms)
			v.Pronoms = newPronoms
		}
		v.Pronoms = append(v.Pronoms, r.Pronoms...)
		slices.Sort(v.Pronoms)
		v.Pronoms = slices.Compact(v.Pronoms)
	}
	for key, val := range r.Checksum {
		v.Checksum[key] = val
	}
	if r.Mimetype != "" {
		v.Mimetype = r.Mimetype
	}
	if r.Pronom != "" {
		v.Pronom = r.Pronom
	}
	if r.Width > v.Width {
		v.Width = r.Width
	}
	if r.Height > v.Height {
		v.Height = r.Height
	}
	if r.Duration > v.Duration {
		v.Duration = r.Duration
	}
	if r.Size > v.Size {
		v.Size = r.Size
	}
	for k, m := range r.Metadata {
		v.Metadata[k] = m
	}
	if r.Errors != nil {
		if v.Errors == nil {
			v.Errors = map[string]string{}
		}
		for k, e := range r.Errors {
			v.Errors[k] = e
		}
	}
	if r.Type != "" {
		v.Type = r.Type
		v.Subtype = r.Subtype
	}
}

type FullMagickResult struct {
	Magick *MagickResult `json:"magick"`
	Frames []*Geometry   `json:"frames,omitempty"`
}
