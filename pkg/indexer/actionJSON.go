package indexer

import (
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"emperror.dev/errors"
)

type ActionJSON struct {
	name   string
	format map[string]ConfigJSONFormat
}

func (as *ActionJSON) CanHandle(contentType string, filename string) bool {
	if strings.ToLower(filepath.Ext(filename)) == ".json" {
		return true
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		//log.Printf("cannot parse media type %s", contentType)
		return false
	}
	if slices.Contains([]string{"application/json", "text/plain"}, mediaType) {
		return true
	}
	return false
}

func NewActionJSON(name string, format map[string]ConfigJSONFormat, ad *ActionDispatcher) Action {
	as := &ActionJSON{
		name:   name,
		format: map[string]ConfigJSONFormat{},
	}
	for key, value := range format {
		var mandatoryFields []string
		for _, field := range value.MandatoryFields {
			mandatoryFields = append(mandatoryFields, strings.ToLower(field))
		}
		var optionalFields []string
		for _, field := range value.OptionalFields {
			optionalFields = append(optionalFields, strings.ToLower(field))
		}
		cf := ConfigJSONFormat{
			MandatoryFields: mandatoryFields,
			OptionalFields:  optionalFields,
			NumOptionals:    value.NumOptionals,
			Pronom:          value.Pronom,
			Mime:            value.Mime,
			Type:            value.Type,
			Subtype:         value.Subtype,
		}
		as.format[key] = cf
	}
	ad.RegisterAction(as)
	return as
}

func (as *ActionJSON) GetWeight() uint {
	return 10
}

func (as *ActionJSON) GetCaps() ActionCapability {
	return ACTFILEHEAD | ACTSTREAM
}

func (as *ActionJSON) GetName() string {
	return as.name
}

func (as *ActionJSON) Stream(contentType string, reader io.Reader, filename string) (*ResultV2, error) {
	fields, err := ExtractJSONFields(reader)
	if err != nil {
		return nil, errors.Wrapf(err, "error extracting JSON fields")
	}

	var foundFormat *ConfigJSONFormat
	for _, format := range as.format {
		match := true
		for _, key := range format.MandatoryFields {
			if !slices.Contains(fields, key) {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		var optionalCount int
		for _, key := range format.OptionalFields {
			if slices.Contains(fields, key) {
				optionalCount++
				if optionalCount > format.NumOptionals {
					break
				}
			}
		}
		if optionalCount < format.NumOptionals {
			continue
		}
		foundFormat = &format
		break
	}

	if foundFormat == nil {
		return nil, errors.New("no matching JSON format found")
	}

	var result = NewResultV2()
	result.Mimetype = foundFormat.Mime
	result.Mimetypes = []string{foundFormat.Mime}
	result.Pronom = foundFormat.Pronom
	result.Pronoms = []string{foundFormat.Pronom}
	result.Type = foundFormat.Type
	result.Subtype = foundFormat.Subtype
	//result.Metadata[as.GetName()]

	return result, nil
}

func (as *ActionJSON) DoV2(filename string) (*ResultV2, error) {
	reader, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot open file '%s'", filename)
	}
	defer reader.Close()
	var head = make([]byte, 500)
	n, err := reader.Read(head)
	if err != nil && err != io.EOF {
		return nil, errors.Wrapf(err, "cannot read file '%s'", filename)
	}
	contentType := http.DetectContentType(head[:n])
	parts := strings.Split(contentType, ";")
	contentType = parts[0]
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, errors.Wrapf(err, "cannot seek to start of file '%s'", filename)
	}
	return as.Stream(contentType, reader, filename)
}

var (
	_ Action = &ActionJSON{}
)

// CSLName beschreibt eine Person gemäß CSL-JSON Namensmodell.
// Siehe data/csl-data.json -> definitions.name-variable
// Hinweis: Einige Felder erlauben mehrere Typen (string|number|boolean),
// weshalb hier der moderne Go-Typ "any" verwendet wird.
type CSLName struct {
	Family              string `json:"family,omitzero"`                // Nachname / Familienname
	Given               string `json:"given,omitzero"`                 // Vorname(n)
	DroppingParticle    string `json:"dropping-particle,omitzero"`     // abfallender Namenszusatz (z. B. „von“)
	NonDroppingParticle string `json:"non-dropping-particle,omitzero"` // nicht‑abfallender Namenszusatz
	Suffix              string `json:"suffix,omitzero"`                // Namenszusatz nachgestellt (z. B. „Jr.“)
	CommaSuffix         any    `json:"comma-suffix,omitzero"`          // Ob Suffix durch Komma abgetrennt ist (CSL: bool/sonst.)
	StaticOrdering      any    `json:"static-ordering,omitzero"`       // Ob die Namensreihenfolge fix ist (kein Umdrehen)
	Literal             string `json:"literal,omitzero"`               // Vollständiger Name als Literal (falls nicht zerlegt)
	ParseNames          any    `json:"parse-names,omitzero"`           // Parser-Hinweis: Literal ggf. in Teile zerlegen
}

// CSLDate beschreibt ein Datum gemäß CSL-JSON.
// Das CSL-Input-Modell unterstützt zwei Repräsentationen:
// 1) EDTF-String (bevorzugt) – hier typischerweise in "raw"/"literal"
// 2) Strukturierte Felder (date-parts, season, circa, ...)
type CSLDate struct {
	DateParts [][]any `json:"date-parts,omitzero"` // Verschachtelte Arrays: [[Jahr,Monat,Tag], [Ende-Jahr,Monat,Tag]]
	Season    any     `json:"season,omitzero"`     // Saison/Jahreszeit (z. B. 1–4 oder Text)
	Circa     any     `json:"circa,omitzero"`      // Ungefähres Datum (ca.) – bool/sonst.
	Literal   string  `json:"literal,omitzero"`    // Menschlich lesbare Datumsangabe
	Raw       string  `json:"raw,omitzero"`        // Unverarbeitete Datumsangabe (z. B. EDTF)
}

// CSLData beschreibt einen kompletten CSL-JSON Eintrag.
// Quelle: data/csl-data.json (Schema v1.0)
// Hinweis: Viele numerische Felder erlauben auch Strings; deshalb werden
// „any“-Typen verwendet, um string|number|boolean-Unionen abzudecken.
type CSLData struct {
	ID                       any            `json:"id"`                                   // Pflicht: Identifier (string|number)
	Type                     string         `json:"type"`                                 // Pflicht: Dokumententyp (z. B. "article-journal")
	CitationKey              string         `json:"citation-key,omitzero"`                // Zitationsschlüssel (z. B. BibTeX-Key)
	Categories               []string       `json:"categories,omitzero"`                  // Freie Kategorien/Tags
	Language                 string         `json:"language,omitzero"`                    // Sprache (z. B. "de", "en")
	JournalAbbreviation      string         `json:"journalAbbreviation,omitzero"`         // Journalabkürzung
	ShortTitle               string         `json:"shortTitle,omitzero"`                  // Kurztitel
	Author                   []CSLName      `json:"author,omitzero"`                      // Autorenliste
	Chair                    []CSLName      `json:"chair,omitzero"`                       // Vorsitz/Chair
	CollectionEditor         []CSLName      `json:"collection-editor,omitzero"`           // Herausgeber einer Sammlung
	Compiler                 []CSLName      `json:"compiler,omitzero"`                    // Kompilator/zusammenstellende Person
	Composer                 []CSLName      `json:"composer,omitzero"`                    // Komponist
	ContainerAuthor          []CSLName      `json:"container-author,omitzero"`            // Autor des übergeordneten Werks
	Contributor              []CSLName      `json:"contributor,omitzero"`                 // Mitwirkende
	Curator                  []CSLName      `json:"curator,omitzero"`                     // Kurator
	Director                 []CSLName      `json:"director,omitzero"`                    // Regisseur
	Editor                   []CSLName      `json:"editor,omitzero"`                      // Herausgeber
	EditorialDirector        []CSLName      `json:"editorial-director,omitzero"`          // Editorial Director
	ExecutiveProducer        []CSLName      `json:"executive-producer,omitzero"`          // Executive Producer
	Guest                    []CSLName      `json:"guest,omitzero"`                       // Gast
	Host                     []CSLName      `json:"host,omitzero"`                        // Host/Moderator
	Interviewer              []CSLName      `json:"interviewer,omitzero"`                 // Interviewer
	Illustrator              []CSLName      `json:"illustrator,omitzero"`                 // Illustrator
	Narrator                 []CSLName      `json:"narrator,omitzero"`                    // Erzähler
	Organizer                []CSLName      `json:"organizer,omitzero"`                   // Organisator
	OriginalAuthor           []CSLName      `json:"original-author,omitzero"`             // Originalautor
	Performer                []CSLName      `json:"performer,omitzero"`                   // Darsteller/Performer
	Producer                 []CSLName      `json:"producer,omitzero"`                    // Produzent
	Recipient                []CSLName      `json:"recipient,omitzero"`                   // Empfänger
	ReviewedAuthor           []CSLName      `json:"reviewed-author,omitzero"`             // Autor des rezensierten Werks
	ScriptWriter             []CSLName      `json:"script-writer,omitzero"`               // Drehbuchautor
	SeriesCreator            []CSLName      `json:"series-creator,omitzero"`              // Serien-Schöpfer
	Translator               []CSLName      `json:"translator,omitzero"`                  // Übersetzer
	Accessed                 *CSLDate       `json:"accessed,omitzero"`                    // Zugriffsdatum
	AvailableDate            *CSLDate       `json:"available-date,omitzero"`              // Verfügbarkeitsdatum
	EventDate                *CSLDate       `json:"event-date,omitzero"`                  // Ereignisdatum
	Issued                   *CSLDate       `json:"issued,omitzero"`                      // Veröffentlichungsdatum
	OriginalDate             *CSLDate       `json:"original-date,omitzero"`               // Ursprungsdatum
	Submitted                *CSLDate       `json:"submitted,omitzero"`                   // Eingereicht am
	Abstract                 string         `json:"abstract,omitzero"`                    // Zusammenfassung
	Annote                   string         `json:"annote,omitzero"`                      // Anmerkung
	Archive                  string         `json:"archive,omitzero"`                     // Archiv
	ArchiveCollection        string         `json:"archive_collection,omitzero"`          // Archiv-Sammlung
	ArchiveLocation          string         `json:"archive_location,omitzero"`            // Archiv-Ort (Signatur/Location)
	ArchivePlace             string         `json:"archive-place,omitzero"`               // Archiv-Ort (geografisch)
	Authority                string         `json:"authority,omitzero"`                   // Zuständige Stelle/Authority
	CallNumber               string         `json:"call-number,omitzero"`                 // Signatur
	ChapterNumber            any            `json:"chapter-number,omitzero"`              // Kapitelnummer (string|number)
	CitationNumber           any            `json:"citation-number,omitzero"`             // Zitationsnummer (string|number)
	CitationLabel            string         `json:"citation-label,omitzero"`              // Zitationslabel
	CollectionNumber         any            `json:"collection-number,omitzero"`           // Sammlungsnummer
	CollectionTitle          string         `json:"collection-title,omitzero"`            // Sammlungstitel
	ContainerTitle           string         `json:"container-title,omitzero"`             // Titel des übergeordneten Werks
	ContainerTitleShort      string         `json:"container-title-short,omitzero"`       // Kurztitel des Containers
	Dimensions               string         `json:"dimensions,omitzero"`                  // Abmessungen
	Division                 string         `json:"division,omitzero"`                    // Abteilung/Division
	DOI                      string         `json:"DOI,omitzero"`                         // Digital Object Identifier
	Edition                  any            `json:"edition,omitzero"`                     // Auflage (string|number)
	Event                    string         `json:"event,omitzero"`                       // Veraltet – nutze "event-title" (laut Schema 1.1 entfernt)
	EventTitle               string         `json:"event-title,omitzero"`                 // Ereignis-/Konferenztitel
	EventPlace               string         `json:"event-place,omitzero"`                 // Ort des Ereignisses
	FirstReferenceNoteNumber any            `json:"first-reference-note-number,omitzero"` // Erstverweis-Fußnotennummer
	Genre                    string         `json:"genre,omitzero"`                       // Genre/Typ
	ISBN                     string         `json:"ISBN,omitzero"`                        // International Standard Book Number
	ISSN                     string         `json:"ISSN,omitzero"`                        // International Standard Serial Number
	Issue                    any            `json:"issue,omitzero"`                       // Heft-/Ausgaben-Nummer
	Jurisdiction             string         `json:"jurisdiction,omitzero"`                // Rechtsraum/Justizbereich
	Keyword                  string         `json:"keyword,omitzero"`                     // Schlagwörter
	Locator                  any            `json:"locator,omitzero"`                     // Locator/Positionsangabe (z. B. Abschnitt)
	Medium                   string         `json:"medium,omitzero"`                      // Medium/Trägertyp
	Note                     string         `json:"note,omitzero"`                        // Notiz (für Benutzeranmerkungen)
	Number                   any            `json:"number,omitzero"`                      // Nummer
	NumberOfPages            any            `json:"number-of-pages,omitzero"`             // Seitenanzahl
	NumberOfVolumes          any            `json:"number-of-volumes,omitzero"`           // Bändeanzahl
	OriginalPublisher        string         `json:"original-publisher,omitzero"`          // Ursprünglicher Verlag
	OriginalPublisherPlace   string         `json:"original-publisher-place,omitzero"`    // Ort des ursprünglichen Verlags
	OriginalTitle            string         `json:"original-title,omitzero"`              // Ursprungstitel
	Page                     any            `json:"page,omitzero"`                        // Seitenangabe
	PageFirst                any            `json:"page-first,omitzero"`                  // Erste Seite
	Part                     any            `json:"part,omitzero"`                        // Teil/Part
	PartTitle                string         `json:"part-title,omitzero"`                  // Titel des Teils
	PMCID                    string         `json:"PMCID,omitzero"`                       // PubMed Central ID
	PMID                     string         `json:"PMID,omitzero"`                        // PubMed ID
	Printing                 any            `json:"printing,omitzero"`                    // Druckauflage
	Publisher                string         `json:"publisher,omitzero"`                   // Verlag/Publisher
	PublisherPlace           string         `json:"publisher-place,omitzero"`             // Verlagsort
	References               string         `json:"references,omitzero"`                  // Literaturverweise im Freitext
	ReviewedGenre            string         `json:"reviewed-genre,omitzero"`              // Genre des rezensierten Werks
	ReviewedTitle            string         `json:"reviewed-title,omitzero"`              // Titel des rezensierten Werks
	Scale                    string         `json:"scale,omitzero"`                       // Maßstab (z. B. Karten)
	Section                  string         `json:"section,omitzero"`                     // Abschnitt/Section
	Source                   string         `json:"source,omitzero"`                      // Quelle
	Status                   string         `json:"status,omitzero"`                      // Veröffentlichungsstatus
	Supplement               any            `json:"supplement,omitzero"`                  // Supplement/Beiheft
	Title                    string         `json:"title,omitzero"`                       // Haupttitel
	TitleShort               string         `json:"title-short,omitzero"`                 // Kurztitel
	URL                      string         `json:"URL,omitzero"`                         // URL
	Version                  string         `json:"version,omitzero"`                     // Version
	Volume                   any            `json:"volume,omitzero"`                      // Band
	VolumeTitle              string         `json:"volume-title,omitzero"`                // Bandtitel
	VolumeTitleShort         string         `json:"volume-title-short,omitzero"`          // Kurztitel des Bands
	YearSuffix               string         `json:"year-suffix,omitzero"`                 // Jahres-Suffix (zur Unterscheidung gleichjähriger Werke)
	Custom                   map[string]any `json:"custom,omitzero"`                      // Freie Key-Value-Paare (bevorzugt ggü. "note" für strukturierte Zusatzinfos)
}
