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
	"io"
	"strings"
	"time"

	"emperror.dev/errors"
)

type ActionIdentify struct {
	name     string
	identify string
	convert  string
	wsl      bool
	timeout  time.Duration
	caps     ActionCapability
	mimeMap  map[string]string
}

func (ai *ActionIdentify) DoV2(filename string) (*ResultV2, error) {
	//TODO implement me
	panic("implement me")
}

func (ai *ActionIdentify) CanHandle(contentType string, filename string) bool {
	return regexIdentifyMime.MatchString(contentType)
}

func (ai *ActionIdentify) Stream(contentType string, reader io.Reader, filename string) (*ResultV2, error) {
	return nil, errors.New("identify actions does not support streaming")
}

func NewActionIdentify(name, identify, convert string, wsl bool, timeout time.Duration, online bool, ad *ActionDispatcher) Action {
	var caps ActionCapability = ACTFILEHEAD
	if online {
		caps |= ACTALLPROTO
	}
	ai := &ActionIdentify{
		name:     name,
		identify: identify,
		convert:  convert,
		wsl:      wsl,
		timeout:  timeout,
		caps:     caps,
		mimeMap:  map[string]string{},
	}
	if mime, err := GetMagickMime(); err == nil {
		if mime != nil {
			for _, m := range mime {
				if m.Acronym != nil && *m.Acronym != "" {
					ai.mimeMap[m.Type] = *m.Acronym
				} else {
					m.Type = strings.ToLower(m.Type)
					if strings.HasPrefix(m.Type, "image/") {
						t := strings.TrimPrefix(m.Type, "image/")
						if t != "" {
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

func (ai *ActionIdentify) GetWeight() uint {
	return 50
}

func (ai *ActionIdentify) GetCaps() ActionCapability {
	return ACTFILEHEAD
}

func (ai *ActionIdentify) GetName() string {
	return ai.name
}

var (
	_ Action = (*ActionIdentify)(nil)
)
