//go:build !cgo

// Copyright 2024 Juergen Enge, info-age GmbH, Basel. All rights reserved.
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
	"context"
	"io"
	"time"

	"emperror.dev/errors"
)

// ErrYaraXCgoRequired is returned when YARA-X action is initialized in a non-CGO environment.
var ErrYaraXCgoRequired = errors.New("yara-x library action requires CGO and libyara_x support")

// ActionYaraX is the fallback stub for environments without CGO.
type ActionYaraX struct {
	name string
}

// NewActionYaraX returns an error when built without CGO.
func NewActionYaraX(
	name string,
	rulesData []byte,
	compiled bool,
	ad *ActionDispatcher,
	fastScan bool,
	timeout time.Duration,
	maxMatchesPerPattern int,
) (*ActionYaraX, error) {
	return nil, ErrYaraXCgoRequired
}

// NewActionYaraXFromFiles returns an error when built without CGO.
func NewActionYaraXFromFiles(
	name string,
	rulesFiles []string,
	ad *ActionDispatcher,
	fastScan bool,
	timeout time.Duration,
	maxMatchesPerPattern int,
) (*ActionYaraX, error) {
	return nil, ErrYaraXCgoRequired
}

// NewActionYaraXWithRule returns an error when built without CGO.
func NewActionYaraXWithRule(
	name string,
	rule any,
	ad *ActionDispatcher,
	fastScan bool,
	timeout time.Duration,
	maxMatchesPerPattern int,
) (*ActionYaraX, error) {
	return nil, ErrYaraXCgoRequired
}

// NewActionYaraXWithRules returns an error when built without CGO.
func NewActionYaraXWithRules(
	name string,
	rules any,
	ad *ActionDispatcher,
	fastScan bool,
	timeout time.Duration,
	maxMatchesPerPattern int,
) (*ActionYaraX, error) {
	return nil, ErrYaraXCgoRequired
}

func (ay *ActionYaraX) CanHandle(contentType string, filename string) bool {
	return false
}

func (ay *ActionYaraX) GetWeight() uint {
	return 15
}

func (ay *ActionYaraX) GetCaps() ActionCapability {
	return ACTFILEHEAD | ACTSTREAM
}

func (ay *ActionYaraX) GetName() string {
	return ay.name
}

func (ay *ActionYaraX) Close() error {
	return nil
}

func (ay *ActionYaraX) DoV2(ctx context.Context, filename string) (*ResultV2, error) {
	return nil, ErrYaraXCgoRequired
}

func (ay *ActionYaraX) Stream(ctx context.Context, contentType string, reader io.Reader, filename string) (*ResultV2, error) {
	return nil, ErrYaraXCgoRequired
}

var (
	_ Action    = &ActionYaraX{}
	_ io.Closer = &ActionYaraX{}
)
