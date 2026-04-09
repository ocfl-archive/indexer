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
)

type duration struct {
	Duration time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

type ConfigClamAV struct {
	Enabled  bool
	Timeout  duration
	ClamScan string
	Wsl      bool
}

type TypeSubtype struct {
	Type    string
	Subtype string
}

type ConfigSiegfried struct {
	//Address string
	Enabled       bool
	SignatureFile string `toml:"signature"`
	MimeMap       map[string]string
	TypeMap       map[string]TypeSubtype
	// StreamSize sets the limit in bytes for copying streams to memory.
	// When streams exceed this size, they are copied to a temporary file.
	// The default value is 64MB.
	StreamSize int
}

type ConfigTika struct {
	AddressMeta           string
	AddressFulltext       string
	Timeout               duration
	RegexpMimeFulltext    string
	RegexpMimeFulltextNot string
	RegexpMimeMeta        string
	RegexpMimeMetaNot     string
	Online                bool
	Enabled               bool
}

type FFMPEGMime struct {
	Video  bool
	Audio  bool
	Format string
	Mime   string
}

type ConfigFFMPEG struct {
	FFProbe string
	Wsl     bool
	Timeout duration
	Online  bool
	Enabled bool
	Mime    []FFMPEGMime
}

type ConfigChecksum struct {
	Name    string
	Digest  []checksum.DigestAlgorithm
	Enabled bool
}

type ConfigImageMagick struct {
	Identify string
	Convert  string
	Wsl      bool
	Timeout  duration
	Online   bool
	Enabled  bool
}

type ConfigJSONFormat struct {
	MandatoryFields []string
	OptionalFields  []string
	NumOptionals    int
	Pronom          string
	Mime            string
	Type            string
	Subtype         string
}

type ConfigJSON struct {
	Enabled bool
	Format  map[string]ConfigJSONFormat
}

type ConfigXMLFormat struct {
	Element    string
	Regexp     bool
	Attributes map[string]string
	Pronom     string
	Mime       string
	Type       string
	Subtype    string
}

type ConfigXML struct {
	Enabled bool
	Format  map[string]ConfigXMLFormat
}

type ConfigExternalAction struct {
	Name,
	Address,
	Mimetype string
	ActionCapabilities []ActionCapability
	CallType           ExternalActionCalltype
}

type ConfigFileMap struct {
	Alias  string
	Folder string
}

type ConfigSFTP struct {
	Knownhosts string
	Password   string
	PrivateKey []string
}

type ConfigNSRL struct {
	Enabled bool
	Badger  string
}

type ConfigMimeWeight struct {
	Regexp string
	Weight int
}

type IndexerConfig struct {
	Enabled         bool
	LocalCache      bool
	TempDir         string
	HeaderTimeout   duration
	HeaderSize      int64
	DownloadMime    string `toml:"forcedownload"`
	MaxDownloadSize int64
	Siegfried       ConfigSiegfried
	Checksum        ConfigChecksum
	FFMPEG          ConfigFFMPEG
	ImageMagick     ConfigImageMagick
	Tika            ConfigTika
	XML             ConfigXML
	JSON            ConfigJSON
	External        []ConfigExternalAction
	FileMap         []ConfigFileMap
	URLRegexp       []string
	NSRL            ConfigNSRL
	Clamav          ConfigClamAV
	MimeRelevance   map[string]ConfigMimeWeight
}

func GetDefaultConfig() *IndexerConfig {
	var conf = &IndexerConfig{}
	if _, err := toml.Decode(data.DefaultConfig, conf); err != nil {
		log.Fatalln("Error decoding default config: ", err)
	}
	return conf
}
