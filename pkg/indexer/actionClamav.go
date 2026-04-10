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
	"time"

	"emperror.dev/errors"
)

func NewActionClamAV(clamav string, wsl bool, timeout time.Duration, ad *ActionDispatcher) Action {
	var caps = ACTFILEFULL
	ac := &ActionClamAV{name: "clamav", clamav: clamav, wsl: wsl, timeout: timeout, caps: caps}
	ad.RegisterAction(ac)
	return ac
}

type ActionClamAV struct {
	name    string
	clamav  string
	wsl     bool
	timeout time.Duration
	caps    ActionCapability
}

func (ac *ActionClamAV) DoV2(filename string) (*ResultV2, error) {
	//TODO implement me
	panic("implement me")
}

func (ac *ActionClamAV) CanHandle(contentType string, filename string) bool {
	return true
}

func (ac *ActionClamAV) Stream(contentType string, reader io.Reader, filename string) (*ResultV2, error) {
	return nil, errors.New("clamav does not support streaming")
}

func (ac *ActionClamAV) GetWeight() uint {
	return 100
}

func (ac *ActionClamAV) GetCaps() ActionCapability {
	return ac.caps
}

func (ac *ActionClamAV) GetName() string {
	return ac.name
}

var (
	_ Action = &ActionClamAV{}
)
