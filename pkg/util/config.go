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
	"context"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"emperror.dev/errors"
	"github.com/BurntSushi/toml"
	"github.com/je4/utils/v2/pkg/stashconfig"
	"github.com/ocfl-archive/indexer/v3/pkg/indexer"
)

type Config struct {
	Indexer *indexer.IndexerConfig
	Log     stashconfig.Config `toml:"log"`
}

func OptimizeConfig(conf *indexer.IndexerConfig) error {
	if conf.Siegfried.SignatureFile == "" {
		user, err := user.Current()
		if err != nil {
			return errors.Wrap(err, "cannot get current user")
		}
		fp := filepath.Join(user.HomeDir, "siegfried", "default.sig")
		fi, err := os.Stat(fp)
		if err == nil && !fi.IsDir() {
			conf.Siegfried.SignatureFile = fp
		} else {
			conf.Siegfried.SignatureFile = "internal:default"
		}
	}
	if conf.FFMPEG.Enabled {
		if conf.FFMPEG.FFProbe == "" {
			if ffprobepath, ok := checkProgram("ffprobe"); ok {
				conf.FFMPEG.FFProbe = ffprobepath
			} else {
				conf.FFMPEG.Enabled = false
			}
		}
	}
	if conf.ImageMagick.Enabled {
		if conf.ImageMagick.Convert == "" {
			if convertpath, ok := checkProgram("magickconvert"); ok {
				conf.ImageMagick.Convert = convertpath
			} else {
				conf.ImageMagick.Enabled = false
			}
			if identifypath, ok := checkProgram("magickidentify"); ok {
				conf.ImageMagick.Identify = identifypath
			} else {
				conf.ImageMagick.Enabled = false
			}
		}
	}
	if conf.Tika.Enabled {
		tikaoptimize := func() error {
			if conf.Tika.AddressMeta == "" {
				conf.Tika.AddressMeta = "http://localhost:9998/tika"
			}
			baseImage := image.NewRGBA(image.Rect(0, 0, 10, 10))
			imageBuffer := bytes.NewBuffer(nil)
			if err := png.Encode(imageBuffer, baseImage); err != nil {
				return errors.Wrap(err, "png.Encode")
			}
			client := &http.Client{}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			reader := bytes.NewBuffer(imageBuffer.Bytes())
			req, err := http.NewRequestWithContext(ctx, http.MethodPut, conf.Tika.AddressMeta, reader)
			if err != nil {
				return errors.Wrapf(err, "cannot create tika request - %v", conf.Tika.AddressMeta)
			}
			req.Header.Add("Accept", "application/json")
			resp, err := client.Do(req)
			if err != nil {
				return errors.Wrapf(err, "error in tika request - %v", conf.Tika.AddressMeta)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				conf.Tika.Enabled = false
				return nil
			}
			bodyData, err := io.ReadAll(resp.Body)
			if err != nil {
				return errors.Wrapf(err, "cannot read tika response - %v", conf.Tika.AddressMeta)
			}
			var meta = &struct {
				Width  string
				Height string
			}{}
			if err := json.Unmarshal(bodyData, meta); err != nil {
				conf.Tika.Enabled = false
				return nil
			}
			if meta.Width != "10" || meta.Height != "10" {
				conf.Tika.Enabled = false
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
	if err := OptimizeConfig(conf.Indexer); err != nil {
		return nil, errors.Wrap(err, "Error optimizing config")
	}
	return conf, nil
}
