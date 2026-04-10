// Package indexer
// Copyright 2020 Juergen Enge, info-age GmbH, Basel. All rights reserved.
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
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"emperror.dev/errors"
	"golang.org/x/exp/slices"
)

var regexIdentifyMime = regexp.MustCompile("^image/")

type ActionIdentifyV2 struct {
	name         string
	identify     string
	convert      string
	wsl          bool
	timeout      time.Duration
	caps         ActionCapability
	mimeMap      map[string]string
	extensionMap map[*regexp.Regexp]string
}

func (ai *ActionIdentifyV2) CanHandle(contentType string, filename string) bool {
	if regexIdentifyMime.MatchString(contentType) {
		return true
	}
	for re, _ := range ai.extensionMap {
		if re.MatchString(filename) {
			return true
		}
	}
	return false
}

func NewActionIdentifyV2(name, identify, convert string, wsl bool, timeout time.Duration, online bool, ad *ActionDispatcher) Action {
	var caps ActionCapability = ACTFILEHEAD
	if online {
		caps |= ACTALLPROTO
	}
	ai := &ActionIdentifyV2{
		name:         name,
		identify:     identify,
		convert:      convert,
		wsl:          wsl,
		timeout:      timeout,
		caps:         caps,
		mimeMap:      map[string]string{},
		extensionMap: map[*regexp.Regexp]string{},
	}
	if mime, err := GetMagickMime(); err == nil {
		if mime != nil {
			for _, m := range mime {
				var pattern, acronym string
				if m.Pattern != nil {
					pattern = *m.Pattern
				}
				if m.Acronym != nil {
					acronym = *m.Acronym
				}
				if pattern != "" {
					ai.extensionMap[regexp.MustCompile(wildCardToRegexp(pattern))] = acronym
				}
				if acronym != "" {
					ai.mimeMap[m.Type] = acronym
				} else {
					m.Type = strings.ToLower(m.Type)
					if strings.HasPrefix(m.Type, "image/") {
						t := m.Type[6:]
						if t != "" && !strings.ContainsAny(t, ".-") {
							ai.mimeMap[m.Type] = t
						}
					}
				}
			}
		}
	}
	ad.RegisterAction(ai)
	return ai
}

func (ai *ActionIdentifyV2) GetWeight() uint {
	return 50
}

func (ai *ActionIdentifyV2) GetCaps() ActionCapability {
	return ACTFILEHEAD | ACTSTREAM
}

func (ai *ActionIdentifyV2) GetName() string {
	return ai.name
}

func (ai *ActionIdentifyV2) Stream(contentType string, reader io.Reader, filename string) (*ResultV2, error) {
	if slices.Contains([]string{"audio", "video", "pdf"}, contentType) {
		return nil, nil
	}
	infile := "-"
	for re, t := range ai.extensionMap {
		if re.MatchString(filename) {
			if t != "" {
				infile = t + ":-"
			}
			break
		}
	}

	var cmdParts = []string{}
	if ai.wsl {
		cmdParts = append(cmdParts, "wsl")
	}
	cmdParts = append(cmdParts, strings.Split(ai.convert, " ")...)
	cmdParts = append(cmdParts, infile, "json:-")

	var out bytes.Buffer
	out.Grow(1024 * 1024) // 1MB size
	ctx, cancel := context.WithTimeout(context.Background(), ai.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)
	cmd.Stdin = reader
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, errors.Wrapf(err, "error executing (%s) for file '%s': %v", strings.Join(cmdParts, " "), filename, out.String())
	}

	var meta = []*MagickResult{}
	data := out.String()

	if data[0] == '{' {
		data = "[" + data + "]"
	}
	if err := json.Unmarshal([]byte(data), &meta); err != nil {
		return nil, errors.Wrapf(err, "cannot unmarshall metadata: %s", data)
	}
	if len(meta) == 0 {
		return nil, errors.New("no metadata from imagemagick found")
	}

	var metadata = FullMagickResult{
		Frames: []*Geometry{},
	}

	metadata.Magick = meta[0]
	if metadata.Magick.Image != nil {
		metadata.Magick.Image.Name = filename
	}
	var result = NewResultV2()
	mimetypes := []string{}
	for _, m := range meta {
		if m.Image == nil {
			continue
		}
		if m.Image.MimeType != "" {
			mimetypes = append(mimetypes, m.Image.MimeType)
		}
		if m.Image.Geometry != nil {
			metadata.Frames = append(metadata.Frames, m.Image.Geometry)
			if uint(m.Image.Geometry.Width+m.Image.Geometry.X) > result.Width {
				result.Width = uint(m.Image.Geometry.Width + m.Image.Geometry.X)
			}
			if uint(m.Image.Geometry.Height+m.Image.Geometry.Y) > result.Height {
				result.Height = uint(m.Image.Geometry.Height + m.Image.Geometry.Y)
			}
		}
	}
	slices.Sort(mimetypes)
	result.Mimetypes = slices.Compact(mimetypes)
	result.Metadata[ai.GetName()] = metadata
	result.Type = "image"
	result.Subtype = metadata.Magick.Image.Format
	if result.Subtype == "PDF" {
		result.Type = "text"
	}

	return result, nil
}

func (ai *ActionIdentifyV2) DoV2(filename string) (*ResultV2, error) {
	infile := filename
	for re, t := range ai.extensionMap {
		if re.MatchString(filename) {
			infile = t + ":" + filename
			break
		}
	}
	cmdparam := []string{infile, "json:-"}
	cmdfile := ai.convert
	if ai.wsl {
		cmdparam = append([]string{cmdfile}, cmdparam...)
		cmdfile = "wsl"
	}

	var out bytes.Buffer
	out.Grow(1024 * 1024) // 1MB size
	ctx, cancel := context.WithTimeout(context.Background(), ai.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdfile, cmdparam...)
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, errors.Wrapf(err, "error executing (%s %s) for file '%s': %v", cmdfile, cmdparam, filename, out.String())
	}

	var meta = []*MagickResult{}
	data := out.String()

	if data[0] == '{' {
		data = "[" + data + "]"
	}
	if err := json.Unmarshal([]byte(data), &meta); err != nil {
		return nil, errors.Wrapf(err, "cannot unmarshall metadata: %s", data)
	}
	if len(meta) == 0 {
		return nil, errors.New("no metadata from imagemagick found")
	}

	var metadata = FullMagickResult{
		Frames: []*Geometry{},
	}

	metadata.Magick = meta[0]
	if metadata.Magick.Image != nil {
		metadata.Magick.Image.Name = filename
	}
	var result = NewResultV2()
	mimetypes := []string{}
	for _, m := range meta {
		if m.Image == nil {
			continue
		}
		if m.Image.MimeType != "" {
			mimetypes = append(mimetypes, m.Image.MimeType)
		}
		if m.Image.Geometry != nil {
			metadata.Frames = append(metadata.Frames, m.Image.Geometry)
			if uint(m.Image.Geometry.Width+m.Image.Geometry.X) > result.Width {
				result.Width = uint(m.Image.Geometry.Width + m.Image.Geometry.X)
			}
			if uint(m.Image.Geometry.Height+m.Image.Geometry.Y) > result.Height {
				result.Height = uint(m.Image.Geometry.Height + m.Image.Geometry.Y)
			}
		}
	}
	slices.Sort(mimetypes)
	result.Mimetypes = slices.Compact(mimetypes)
	result.Metadata[ai.GetName()] = metadata

	return result, nil
}

var (
	_ Action = (*ActionIdentifyV2)(nil)
)
