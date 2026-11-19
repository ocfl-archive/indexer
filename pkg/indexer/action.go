// Package indexer Copyright 2020 Juergen Enge, info-age GmbH, Basel. All rights reserved.
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
// Package indexer provides a high-performance streaming file characterization,
// format identification, and metadata extraction engine.
package indexer

import (
	"context"
	"fmt"
	"io"
	"regexp"

	"emperror.dev/errors"
)

// ActionCapability represents bitflags defining the execution capabilities
// and input source requirements of an Action.
type ActionCapability uint

const (
	// ACTFILE indicates that the action requires a local file on disk.
	ACTFILE ActionCapability = 1 << iota
	// ACTHTTP indicates that the action can directly fetch/process HTTP URLs.
	ACTHTTP
	// ACTHTTPS indicates that the action can directly fetch/process HTTPS URLs.
	ACTHTTPS
	// ACTHEAD indicates that the action can operate with just the initial header bytes.
	ACTHEAD
	// ACTSTREAM indicates that the action can process data directly from an io.Reader stream.
	ACTSTREAM

	// ACTWEB combines HTTP and HTTPS network capabilities.
	ACTWEB = ACTHTTPS | ACTHTTP
	// ACTALLPROTO combines local file, HTTP, and HTTPS protocols.
	ACTALLPROTO = ACTFILE | ACTHTTP | ACTHTTPS
	// ACTALL encompasses all supported protocols and header-only execution.
	ACTALL = ACTALLPROTO | ACTHEAD
	// ACTFILEHEAD indicates support for local files and header-only streaming.
	ACTFILEHEAD = ACTFILE | ACTHEAD
	// ACTFILEFULL indicates that the action requires the complete local file, not just a header.
	ACTFILEFULL = ACTFILE & ^ACTHEAD
)

// ACTString maps ActionCapability flags to their human-readable string names.
var ACTString map[ActionCapability]string = map[ActionCapability]string{
	ACTFILE:   "ACTFILE",
	ACTHTTP:   "ACTHTTP",
	ACTHTTPS:  "ACTHTTPS",
	ACTHEAD:   "ACTHEAD",
	ACTSTREAM: "ACTSTREAM",
}

// ACTAction maps string names to their corresponding ActionCapability bitflag.
var ACTAction map[string]ActionCapability = map[string]ActionCapability{
	"ACTFILE":   ACTFILE,
	"ACTHTTP":   ACTHTTP,
	"ACTHTTPS":  ACTHTTPS,
	"ACTHEAD":   ACTHEAD,
	"ACTSTREAM": ACTSTREAM,
}

// UnmarshalText deserializes a string token into an ActionCapability bitflag for TOML configuration.
func (a *ActionCapability) UnmarshalText(text []byte) error {
	var ok bool
	*a, ok = ACTAction[string(text)]
	if !ok {
		return fmt.Errorf("invalid actions capability: %s", string(text))
	}
	return nil
}

// ErrMimeNotApplicable is returned when an action cannot process the given MIME type.
var ErrMimeNotApplicable = errors.New("mime type not applicable for actions")

// Action represents an identification or metadata extraction component in the indexer pipeline.
type Action interface {
	// Stream executes analysis on an io.Reader data stream.
	Stream(ctx context.Context, contentType string, reader io.Reader, filename string) (*ResultV2, error)
	// DoV2 executes analysis on a local file on disk.
	DoV2(ctx context.Context, filename string) (*ResultV2, error)
	// CanHandle checks if the action can process the given content type or file name.
	CanHandle(contentType string, filename string) bool
	// GetName returns the unique name of the action.
	GetName() string
	// GetCaps returns the bitmask of ActionCapability flags supported by this action.
	GetCaps() ActionCapability
	// GetWeight returns the priority weight of the action during sorting and execution.
	GetWeight() uint
}

// MimeWeightString holds a regular expression and associated priority weight from configuration.
type MimeWeightString struct {
	Regexp string
	Weight int
}

// MimeWeight represents a compiled regular expression and weight for sorting detected MIME types.
type MimeWeight struct {
	regexp *regexp.Regexp
	weight int
}
