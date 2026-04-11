// Package indexer Copyright 2021 Juergen Enge, info-age GmbH, Basel. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package indexer

import (
	"log"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/je4/utils/v2/pkg/checksum"
	"github.com/ocfl-archive/indexer/v3/data"
)

const (
	NameSiegfried = "siegfried"
	NameXML       = "xml"
	NameChecksum  = "checksum"
	NameTika      = "tika"
	NameFFProbe   = "ffprobe"
	NameIdentify  = "identify"
	NameFullText  = "fulltext"
	NameJSON      = "json"
	NameClamav    = "clamav"
	NameNSRL      = "nsrl"
)

type duration struct {
	Duration time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

// ConfigClamAV represents the configuration for ClamAV antivirus scanning.
type ConfigClamAV struct {
	// Enabled indicates whether ClamAV scanning is active.
	Enabled bool `toml:"enabled"`
	// Timeout specifies the maximum duration for a scan.
	Timeout duration `toml:"timeout"`
	// ClamScan is the path to the clamscan executable.
	ClamScan string `toml:"clamscan"`
	// Wsl indicates whether to run clamscan via Windows Subsystem for Linux.
	Wsl bool `toml:"wsl"`
}

// TypeSubtype represents a media type and its corresponding subtype.
type TypeSubtype struct {
	// Type is the primary media type (e.g., "image", "video").
	Type string `toml:"type"`
	// Subtype is the specific format (e.g., "jpeg", "mp4").
	Subtype string `toml:"subtype"`
}

// ConfigSiegfried represents the configuration for the Siegfried file identification tool.
type ConfigSiegfried struct {
	// Enabled indicates whether Siegfried identification is active.
	Enabled bool `toml:"enabled"`
	// SignatureFile is the path to the Siegfried signature file.
	SignatureFile string `toml:"signature"`
	// MimeMap allows mapping identified MIME types to alternative strings.
	MimeMap map[string]string `toml:"mimemap"`
	// TypeMap allows mapping identified formats to specific TypeSubtype structures.
	TypeMap map[string]TypeSubtype `toml:"typemap"`
	// StreamSize sets the limit in bytes for copying streams to memory.
	// When streams exceed this size, they are copied to a temporary file.
	// The default value is 64MB.
	StreamSize int `toml:"streamsize"`
}

// ConfigTika represents the configuration for the Apache Tika metadata and fulltext extraction tool.
type ConfigTika struct {
	// AddressMeta is the URL of the Tika server for metadata extraction.
	AddressMeta string `toml:"addressmeta"`
	// AddressFulltext is the URL of the Tika server for fulltext extraction.
	AddressFulltext string `toml:"addressfulltext"`
	// Timeout specifies the maximum duration for a Tika request.
	Timeout duration `toml:"timeout"`
	// RegexpMimeFulltext is a regular expression to include MIME types for fulltext extraction.
	RegexpMimeFulltext string `toml:"regexpmimefulltext"`
	// RegexpMimeFulltextNot is a regular expression to exclude MIME types from fulltext extraction.
	RegexpMimeFulltextNot string `toml:"regexpmimefulltextnot"`
	// RegexpMimeMeta is a regular expression to include MIME types for metadata extraction.
	RegexpMimeMeta string `toml:"regexpmimemeta"`
	// RegexpMimeMetaNot is a regular expression to exclude MIME types from metadata extraction.
	RegexpMimeMetaNot string `toml:"regexpmimemetanot"`
	// Online indicates whether Tika should be used if it's an online service.
	Online bool `toml:"online"`
	// Enabled indicates whether Tika extraction is active.
	Enabled bool `toml:"enabled"`
}

// FFMPEGMime defines the relationship between FFmpeg formats and MIME types.
type FFMPEGMime struct {
	// Video indicates if the format supports video.
	Video bool `toml:"video"`
	// Audio indicates if the format supports audio.
	Audio bool `toml:"audio"`
	// Format is the FFmpeg format name.
	Format string `toml:"format"`
	// Mime is the corresponding MIME type.
	Mime string `toml:"mime"`
}

// ConfigFFMPEG represents the configuration for FFmpeg/FFProbe media analysis.
type ConfigFFMPEG struct {
	// FFProbe is the path to the ffprobe executable.
	FFProbe string `toml:"ffprobe"`
	// Wsl indicates whether to run ffprobe via Windows Subsystem for Linux.
	Wsl bool `toml:"wsl"`
	// Timeout specifies the maximum duration for an analysis.
	Timeout duration `toml:"timeout"`
	// Online indicates whether FFmpeg should be used for online resources.
	Online bool `toml:"online"`
	// Enabled indicates whether FFmpeg analysis is active.
	Enabled bool `toml:"enabled"`
	// Mime is a list of MIME type mappings for FFmpeg.
	Mime []FFMPEGMime `toml:"mime"`
}

// ConfigChecksum represents the configuration for checksum generation.
type ConfigChecksum struct {
	// Name is a descriptive name for this checksum configuration.
	Name string `toml:"name"`
	// Digest is a list of checksum algorithms to use (e.g., SHA-256, MD5).
	Digest []checksum.DigestAlgorithm `toml:"digest"`
	// Enabled indicates whether checksum generation is active.
	Enabled bool `toml:"enabled"`
}

// ConfigImageMagick represents the configuration for ImageMagick image identification.
type ConfigImageMagick struct {
	// Identify is the path to the ImageMagick identify executable.
	Identify string `toml:"identify"`
	// Convert is the path to the ImageMagick convert executable.
	Convert string `toml:"convert"`
	// Wsl indicates whether to run ImageMagick via Windows Subsystem for Linux.
	Wsl bool `toml:"wsl"`
	// Timeout specifies the maximum duration for an image analysis.
	Timeout duration `toml:"timeout"`
	// Online indicates whether ImageMagick should be used for online resources.
	Online bool `toml:"online"`
	// Enabled indicates whether ImageMagick analysis is active.
	Enabled bool `toml:"enabled"`
}

// ConfigJSONFormat defines the rules for identifying files based on JSON content.
type ConfigJSONFormat struct {
	// MandatoryFields is a list of fields that must be present in the JSON.
	MandatoryFields []string `toml:"mandatoryfields"`
	// OptionalFields is a list of fields that can be optionally present.
	OptionalFields []string `toml:"optionalfields"`
	// NumOptionals is the number of optional fields required for a match.
	NumOptionals int `toml:"numoptionals"`
	// Pronom is the PRONOM unique identifier for this format.
	Pronom string `toml:"pronom"`
	// Mime is the MIME type assigned to this format.
	Mime string `toml:"mime"`
	// Type is the primary media type.
	Type string `toml:"type"`
	// Subtype is the specific format subtype.
	Subtype string `toml:"subtype"`
}

// ConfigJSON represents the configuration for JSON-based file identification.
type ConfigJSON struct {
	// Enabled indicates whether JSON identification is active.
	Enabled bool `toml:"enabled"`
	// Format is a map of JSON format identification rules.
	Format map[string]ConfigJSONFormat `toml:"format"`
}

// ConfigXMLFormat defines the rules for identifying files based on XML content.
type ConfigXMLFormat struct {
	// Element is the XML element name to look for.
	Element string `toml:"element"`
	// Regexp indicates if the Element field should be treated as a regular expression.
	Regexp bool `toml:"regexp"`
	// Attributes is a map of required XML attributes and their expected values.
	Attributes map[string]string `toml:"attributes"`
	// Pronom is the PRONOM unique identifier for this format.
	Pronom string `toml:"pronom"`
	// Mime is the MIME type assigned to this format.
	Mime string `toml:"mime"`
	// Type is the primary media type.
	Type string `toml:"type"`
	// Subtype is the specific format subtype.
	Subtype string `toml:"subtype"`
}

// ConfigXML represents the configuration for XML-based file identification.
type ConfigXML struct {
	// Enabled indicates whether XML identification is active.
	Enabled bool `toml:"enabled"`
	// Format is a map of XML format identification rules.
	Format map[string]ConfigXMLFormat `toml:"format"`
}

// ConfigExternalAction represents the configuration for calling an external service for analysis.
type ConfigExternalAction struct {
	// Name is a descriptive name for the external action.
	Name string `toml:"name"`
	// Address is the URL or endpoint of the external service.
	Address string `toml:"address"`
	// Mimetype is the MIME type that triggers this external action.
	Mimetype string `toml:"mimetype"`
	// ActionCapabilities is a list of capabilities this action supports.
	ActionCapabilities []ActionCapability `toml:"actioncapabilities"`
	// CallType specifies how the external action is called (e.g., GET, POST).
	CallType ExternalActionCalltype `toml:"calltype"`
}

// ConfigFileMap represents a mapping from a virtual path (alias) to a local folder.
type ConfigFileMap struct {
	// Alias is the virtual path or identifier.
	Alias string `toml:"alias"`
	// Folder is the actual local directory path.
	Folder string `toml:"folder"`
}

// ConfigSFTP represents the configuration for SFTP access.
type ConfigSFTP struct {
	// Knownhosts is the path to the SSH known_hosts file.
	Knownhosts string `toml:"knownhosts"`
	// Password is the password for SFTP authentication.
	Password string `toml:"password"`
	// PrivateKey is a list of paths to SSH private key files.
	PrivateKey []string `toml:"privatekey"`
}

// ConfigNSRL represents the configuration for the National Software Reference Library (NSRL) lookup.
type ConfigNSRL struct {
	// Enabled indicates whether NSRL lookup is active.
	Enabled bool `toml:"enabled"`
	// Badger is the path to the Badger database containing NSRL data.
	Badger string `toml:"badger"`
}

// ConfigMimeWeight represents a weight assigned to certain MIME types for relevance ranking.
type ConfigMimeWeight struct {
	// Regexp is a regular expression to match MIME types.
	Regexp string `toml:"regexp"`
	// Weight is the numeric priority or weight assigned to matching MIME types.
	Weight int `toml:"weight"`
}

// IndexerConfig is the main configuration structure for the indexer.
type IndexerConfig struct {
	// Enabled indicates whether the indexer is globally active.
	Enabled bool `toml:"enabled"`
	// LocalCache indicates whether to use a local cache for files.
	LocalCache bool `toml:"localcache"`
	// TempDir is the directory for temporary files.
	TempDir string `toml:"tempdir"`
	// Optimize indicates whether to optimize identification processes.
	Optimize bool `toml:"optimize"`
	// HeaderTimeout specifies the timeout for reading file headers.
	HeaderTimeout duration `toml:"headertimeout"`
	// HeaderSize is the number of bytes to read from the beginning of a file for identification.
	HeaderSize int64 `toml:"headersize"`
	// DownloadMime is a MIME type that forces a download for analysis.
	DownloadMime string `toml:"forcedownload"`
	// MaxDownloadSize is the maximum allowed size for downloads.
	MaxDownloadSize int64 `toml:"maxdownloadsize"`
	// Siegfried is the configuration for Siegfried identification.
	Siegfried ConfigSiegfried `toml:"siegfried"`
	// Checksum is the configuration for checksum generation.
	Checksum ConfigChecksum `toml:"checksum"`
	// FFMPEG is the configuration for FFmpeg/FFProbe analysis.
	FFMPEG ConfigFFMPEG `toml:"ffmpeg"`
	// ImageMagick is the configuration for ImageMagick analysis.
	ImageMagick ConfigImageMagick `toml:"imagemagick"`
	// Tika is the configuration for Apache Tika analysis.
	Tika ConfigTika `toml:"tika"`
	// XML is the configuration for XML-based identification.
	XML ConfigXML `toml:"xml"`
	// JSON is the configuration for JSON-based identification.
	JSON ConfigJSON `toml:"json"`
	// External is a list of configurations for external actions.
	External []ConfigExternalAction `toml:"external"`
	// FileMap is a list of virtual-to-local path mappings.
	FileMap []ConfigFileMap `toml:"filemap"`
	// URLRegexp is a list of regular expressions for identifying relevant URLs.
	URLRegexp []string `toml:"urlregexp"`
	// NSRL is the configuration for NSRL lookups.
	NSRL ConfigNSRL `toml:"nsrl"`
	// Clamav is the configuration for ClamAV antivirus scanning.
	Clamav ConfigClamAV `toml:"clamav"`
	// MimeRelevance is a map of MIME type relevance weights.
	MimeRelevance map[string]ConfigMimeWeight `toml:"mimerelevance"`
}

func GetDefaultConfig() *IndexerConfig {
	var conf = &IndexerConfig{}
	if _, err := toml.Decode(data.DefaultConfig, conf); err != nil {
		log.Fatalln("Error decoding default config: ", err)
	}
	return conf
}
