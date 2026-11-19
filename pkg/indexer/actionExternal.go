// Copyright 2020 Juergen Enge, info-age GmbH, Basel. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

// Start-Process -FilePath c:/daten/go/bin/sf.exe -Args "-serve localhost:5138" -Wait -NoNewWindow
// c:/daten/go/bin/sf.exe -serve localhost:5138

package indexer

import (
	"context"
	"fmt"
	"io"
	"regexp"

	"emperror.dev/errors"
)

// ExternalActionCalltype defines the invocation protocol for an external service action.
type ExternalActionCalltype uint

const (
	// EACTURL substitutes the target path in the URL query/path template.
	EACTURL ExternalActionCalltype = 1 << iota
	// EACTJSONPOST posts a JSON payload containing file information to the service.
	EACTJSONPOST
)

// EACTString maps ExternalActionCalltype values to human-readable strings.
var EACTString map[ExternalActionCalltype]string = map[ExternalActionCalltype]string{
	EACTURL:      "EACTURL",
	EACTJSONPOST: "EACTJSONPOST",
}

// EACTAction maps string tokens to ExternalActionCalltype enum values.
var EACTAction map[string]ExternalActionCalltype = map[string]ExternalActionCalltype{
	"EACTURL":      EACTURL,
	"EACTJSONPOST": EACTJSONPOST,
}

// UnmarshalText deserializes a string into an ExternalActionCalltype.
func (a *ExternalActionCalltype) UnmarshalText(text []byte) error {
	var ok bool
	*a, ok = EACTAction[string(text)]
	if !ok {
		return fmt.Errorf("invalid actions capability: %s", string(text))
	}
	return nil
}

// NewActionExternal creates and registers a new ActionExternal instance.
func NewActionExternal(name, address string, capability ActionCapability, callType ExternalActionCalltype, mimetype string, ad *ActionDispatcher) Action {
	ae := &ActionExternal{
		name:       name,
		url:        address,
		capability: capability,
		callType:   callType,
		mimetype:   regexp.MustCompile(mimetype),
	}
	ad.RegisterAction(ae)
	return ae
}

// ActionExternal invokes an external HTTP microservice or webhook for format analysis.
type ActionExternal struct {
	name       string
	url        string
	capability ActionCapability
	callType   ExternalActionCalltype
	mimetype   *regexp.Regexp
}

func (as *ActionExternal) DoV2(ctx context.Context, filename string) (*ResultV2, error) {
	//TODO implement me
	panic("implement me")
}

func (as *ActionExternal) CanHandle(contentType string, filename string) bool {
	return true
}

func (as *ActionExternal) Stream(ctx context.Context, contentType string, reader io.Reader, filename string) (*ResultV2, error) {
	return nil, errors.New("external actions does not support streaming")
}

func (as *ActionExternal) GetWeight() uint {
	return 100
}

func (as *ActionExternal) GetCaps() ActionCapability {
	return as.capability
}

func (as *ActionExternal) GetName() string {
	return as.name
}

var (
	_ Action = (*ActionExternal)(nil)
)
