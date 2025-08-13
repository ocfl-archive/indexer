// Package util
// Copyright 2021 Juergen Enge, info-age GmbH, Basel. All rights reserved.
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
package util

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"os"
	"os/user"
	"path/filepath"

	"emperror.dev/errors"
	"github.com/BurntSushi/toml"
	"github.com/je4/utils/v2/pkg/stashconfig"
	"github.com/ocfl-archive/indexer/v3/pkg/indexer"
)

type Config struct {
	Indexer *indexer.IndexerConfig
	Log     stashconfig.Config `toml:"log"`
}

func OptimizeConfig(conf *Config) error {
	if conf.Indexer.Siegfried.SignatureFile == "" {
		user, err := user.Current()
		if err != nil {
			return errors.Wrap(err, "cannot get current user")
		}
		fp := filepath.Join(user.HomeDir, "siegfried", "default.sig")
		fi, err := os.Stat(fp)
		if err == nil && !fi.IsDir() {
			conf.Indexer.Siegfried.SignatureFile = fp
		}
	}
	if conf.Indexer.FFMPEG.Enabled {
		if conf.Indexer.FFMPEG.FFProbe == "" {
			if ffprobepath, ok := checkProgram("ffprobe"); ok {
				conf.Indexer.FFMPEG.FFProbe = ffprobepath
			} else {
				conf.Indexer.FFMPEG.Enabled = false
			}
		}
	}
	if conf.Indexer.ImageMagick.Enabled {
		if conf.Indexer.ImageMagick.Convert == "" {
			if convertpath, ok := checkProgram("magickconvert"); ok {
				conf.Indexer.ImageMagick.Convert = convertpath
			} else {
				conf.Indexer.ImageMagick.Enabled = false
			}
			if identifypath, ok := checkProgram("magickidentify"); ok {
				conf.Indexer.ImageMagick.Identify = identifypath
			} else {
				conf.Indexer.ImageMagick.Enabled = false
			}
		}
	}
	if conf.Indexer.Tika.Enabled {
		tikaoptimize := func() error {
			if conf.Indexer.Tika.AddressMeta == "" {
				conf.Indexer.Tika.AddressMeta = "http://localhost:9998"
			}
			baseImage := image.NewRGBA(image.Rect(0, 0, 10, 10))
			imageBuffer := bytes.NewBuffer(nil)
			if err := png.Encode(imageBuffer, baseImage); err != nil {
				return errors.Wrap(err, "png.Encode")
			}
			var meta = map[string]any{}
			resp, err := http.Post(conf.Indexer.Tika.AddressMeta, "application/octet-stream", imageBuffer)
			if err != nil {
				conf.Indexer.Tika.Enabled = false
				return nil
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				conf.Indexer.Tika.Enabled = false
				return nil
			}
			if err := json.NewDecoder(resp.Body).Decode(meta); err != nil {
				conf.Indexer.Tika.Enabled = false
				return nil
			}
			return nil
		}
		if err := tikaoptimize(); err != nil {
			return errors.Wrap(err, "tikaoptimize")
		}
	}
	return nil
}

func LoadConfig(tomlBytes []byte) (*Config, error) {
	var conf = &Config{
		Indexer: &indexer.IndexerConfig{},
		Log: stashconfig.Config{
			Level: "ERROR",
		},
	}

	if err := toml.Unmarshal(tomlBytes, conf); err != nil {
		return nil, errors.Wrapf(err, "Error unmarshalling config")
	}
	if err := OptimizeConfig(conf); err != nil {
		return nil, errors.Wrap(err, "Error optimizing config")
	}
	return conf, nil
}
