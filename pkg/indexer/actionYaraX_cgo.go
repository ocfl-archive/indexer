//go:build cgo

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
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"emperror.dev/errors"
	yara_x "github.com/VirusTotal/yara-x/go"
)

// ActionYaraX executes YARA-X rule matching using the in-process YARA-X library engine.
type ActionYaraX struct {
	name                 string
	rules                []*yara_x.Rules
	scannerPool          sync.Pool
	fastScan             bool
	timeout              time.Duration
	maxMatchesPerPattern int
}

// LoadYaraXRulesFromFiles loads, compiles, or deserializes YARA-X rules from a list of file paths or glob patterns.
// It automatically detects whether files are source rules (.yar, .yara) or compiled rules (.yrc) based on extension.
func LoadYaraXRulesFromFiles(patterns []string) ([]*yara_x.Rules, error) {
	var resolvedFiles []string
	seenFiles := make(map[string]bool)

	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		if strings.ContainsAny(pattern, "*?[") {
			matches, err := filepath.Glob(pattern)
			if err != nil {
				return nil, errors.Wrapf(err, "invalid glob pattern '%s'", pattern)
			}
			for _, match := range matches {
				abs, err := filepath.Abs(match)
				if err != nil {
					abs = match
				}
				if !seenFiles[abs] {
					seenFiles[abs] = true
					resolvedFiles = append(resolvedFiles, match)
				}
			}
		} else {
			abs, err := filepath.Abs(pattern)
			if err != nil {
				abs = pattern
			}
			if !seenFiles[abs] {
				seenFiles[abs] = true
				resolvedFiles = append(resolvedFiles, pattern)
			}
		}
	}

	if len(resolvedFiles) == 0 {
		return nil, errors.New("no yara-x rules files matched")
	}

	var sourceFiles []string
	var compiledFiles []string

	for _, file := range resolvedFiles {
		ext := strings.ToLower(filepath.Ext(file))
		if ext == ".yrc" {
			compiledFiles = append(compiledFiles, file)
		} else {
			sourceFiles = append(sourceFiles, file)
		}
	}

	var loadedRules []*yara_x.Rules
	cleanupLoaded := func() {
		for _, r := range loadedRules {
			if r != nil {
				r.Destroy()
			}
		}
		loadedRules = nil
	}

	// 1. Compile all source files into a single unified Rules set for maximum performance
	if len(sourceFiles) > 0 {
		compiler, err := yara_x.NewCompiler()
		if err != nil {
			cleanupLoaded()
			return nil, errors.Wrap(err, "failed to initialize yara-x compiler")
		}

		for _, sf := range sourceFiles {
			content, err := os.ReadFile(sf)
			if err != nil {
				compiler.Destroy()
				cleanupLoaded()
				return nil, errors.Wrapf(err, "cannot read yara-x source rule file '%s'", sf)
			}
			if err := compiler.AddSource(string(content), yara_x.WithOrigin(sf)); err != nil {
				compiler.Destroy()
				cleanupLoaded()
				return nil, errors.Wrapf(err, "failed to compile yara-x source rule file '%s'", sf)
			}
		}

		builtRules := compiler.Build()
		compiler.Destroy()
		if builtRules == nil {
			cleanupLoaded()
			return nil, errors.New("failed to build yara-x rules from source files")
		}
		loadedRules = append(loadedRules, builtRules)
	}

	// 2. Load pre-compiled .yrc binary rule sets
	for _, cf := range compiledFiles {
		data, err := os.ReadFile(cf)
		if err != nil {
			cleanupLoaded()
			return nil, errors.Wrapf(err, "cannot read compiled yara-x rule file '%s'", cf)
		}
		cr, err := yara_x.ReadFrom(bytes.NewReader(data))
		if err != nil {
			cleanupLoaded()
			return nil, errors.Wrapf(err, "failed to deserialize compiled yara-x rule file '%s'", cf)
		}
		loadedRules = append(loadedRules, cr)
	}

	return loadedRules, nil
}

// NewActionYaraXFromFiles creates and registers an ActionYaraX instance from a list of file paths or glob patterns.
func NewActionYaraXFromFiles(
	name string,
	rulesFiles []string,
	ad *ActionDispatcher,
	fastScan bool,
	timeout time.Duration,
	maxMatchesPerPattern int,
) (*ActionYaraX, error) {
	rules, err := LoadYaraXRulesFromFiles(rulesFiles)
	if err != nil {
		return nil, err
	}
	return NewActionYaraXWithRules(name, rules, ad, fastScan, timeout, maxMatchesPerPattern)
}

// NewActionYaraX creates and registers a new ActionYaraX instance from raw byte content.
// rulesData can be either source YARA rules (compiled=false) or precompiled rules (compiled=true).
func NewActionYaraX(
	name string,
	rulesData []byte,
	compiled bool,
	ad *ActionDispatcher,
	fastScan bool,
	timeout time.Duration,
	maxMatchesPerPattern int,
) (*ActionYaraX, error) {
	if len(rulesData) == 0 {
		return nil, errors.New("empty yara-x rules data provided")
	}

	var rules *yara_x.Rules
	var err error

	if compiled {
		rules, err = yara_x.ReadFrom(bytes.NewReader(rulesData))
		if err != nil {
			return nil, errors.Wrap(err, "failed to deserialize compiled yara-x rules")
		}
	} else {
		rules, err = yara_x.Compile(string(rulesData))
		if err != nil {
			return nil, errors.Wrap(err, "failed to compile yara-x rules source")
		}
	}

	return NewActionYaraXWithRules(name, []*yara_x.Rules{rules}, ad, fastScan, timeout, maxMatchesPerPattern)
}

// NewActionYaraXWithRule creates and registers an ActionYaraX instance with a single compiled Rules instance.
func NewActionYaraXWithRule(
	name string,
	rule *yara_x.Rules,
	ad *ActionDispatcher,
	fastScan bool,
	timeout time.Duration,
	maxMatchesPerPattern int,
) (*ActionYaraX, error) {
	if rule == nil {
		return nil, errors.New("nil yara-x rules provided")
	}
	return NewActionYaraXWithRules(name, []*yara_x.Rules{rule}, ad, fastScan, timeout, maxMatchesPerPattern)
}

// NewActionYaraXWithRules creates and registers an ActionYaraX instance with one or more compiled Rules sets.
func NewActionYaraXWithRules(
	name string,
	rules []*yara_x.Rules,
	ad *ActionDispatcher,
	fastScan bool,
	timeout time.Duration,
	maxMatchesPerPattern int,
) (*ActionYaraX, error) {
	if len(rules) == 0 {
		return nil, errors.New("no yara-x rules provided")
	}
	for _, r := range rules {
		if r == nil {
			return nil, errors.New("nil yara-x rule in rules slice")
		}
	}

	ay := &ActionYaraX{
		name:                 name,
		rules:                rules,
		fastScan:             fastScan,
		timeout:              timeout,
		maxMatchesPerPattern: maxMatchesPerPattern,
	}

	ay.scannerPool = sync.Pool{
		New: func() any {
			scanners := make([]*yara_x.Scanner, len(ay.rules))
			for i, r := range ay.rules {
				scanner := yara_x.NewScanner(r)
				if ay.fastScan {
					scanner.FastScan(true)
				}
				if ay.timeout > 0 {
					scanner.SetTimeout(ay.timeout)
				}
				if ay.maxMatchesPerPattern > 0 {
					scanner.MaxMatchesPerPattern(ay.maxMatchesPerPattern)
				}
				scanners[i] = scanner
			}
			return scanners
		},
	}

	if ad != nil {
		ad.RegisterAction(ay)
	}
	return ay, nil
}

func (ay *ActionYaraX) CanHandle(contentType string, filename string) bool {
	return true
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
	for _, r := range ay.rules {
		if r != nil {
			r.Destroy()
		}
	}
	ay.rules = nil
	return nil
}

func (ay *ActionYaraX) DoV2(ctx context.Context, filename string) (*ResultV2, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	scanners, ok := ay.scannerPool.Get().([]*yara_x.Scanner)
	if !ok || len(scanners) == 0 {
		return nil, errors.New("cannot obtain yara-x scanner from pool")
	}
	defer ay.scannerPool.Put(scanners)

	result := NewResultV2()
	var allMatches []YaraXRuleMatch

	for _, scanner := range scanners {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		scanResults, err := scanner.ScanFile(filename)
		if err != nil {
			return nil, errors.Wrapf(err, "yara-x scanning error for file %s", filename)
		}
		ay.appendScanResults(result, &allMatches, scanResults)
	}

	result.Metadata[ay.GetName()] = allMatches
	return result, nil
}

func (ay *ActionYaraX) Stream(ctx context.Context, contentType string, reader io.Reader, filename string) (*ResultV2, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot read stream for %s", filename)
	}

	scanners, ok := ay.scannerPool.Get().([]*yara_x.Scanner)
	if !ok || len(scanners) == 0 {
		return nil, errors.New("cannot obtain yara-x scanner from pool")
	}
	defer ay.scannerPool.Put(scanners)

	result := NewResultV2()
	var allMatches []YaraXRuleMatch

	for _, scanner := range scanners {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		scanResults, err := scanner.Scan(data)
		if err != nil {
			return nil, errors.Wrapf(err, "yara-x scanning error for stream %s", filename)
		}
		ay.appendScanResults(result, &allMatches, scanResults)
	}

	result.Metadata[ay.GetName()] = allMatches
	return result, nil
}

func (ay *ActionYaraX) appendScanResults(result *ResultV2, allMatches *[]YaraXRuleMatch, scanResults *yara_x.ScanResults) {
	if scanResults == nil {
		return
	}

	matchingRules := scanResults.MatchingRules()
	for _, rule := range matchingRules {
		match := YaraXRuleMatch{
			Rule:      rule.Identifier(),
			Namespace: rule.Namespace(),
			Tags:      rule.Tags(),
			Metadata:  make(map[string]any),
		}

		for _, meta := range rule.Metadata() {
			val := meta.Value()
			match.Metadata[meta.Identifier()] = val

			switch strings.ToLower(meta.Identifier()) {
			case "mime", "mimetype":
				if s, ok := val.(string); ok && s != "" {
					if !slices.Contains(result.Mimetypes, s) {
						result.Mimetypes = append(result.Mimetypes, s)
					}
				}
			case "pronom":
				if s, ok := val.(string); ok && s != "" {
					if !slices.Contains(result.Pronoms, s) {
						result.Pronoms = append(result.Pronoms, s)
					}
				}
			case "type":
				if s, ok := val.(string); ok && s != "" {
					result.Type = s
				}
			case "subtype":
				if s, ok := val.(string); ok && s != "" {
					result.Subtype = s
				}
			}
		}

		for _, pattern := range rule.Patterns() {
			pInfo := YaraXPatternInfo{
				Identifier: pattern.Identifier(),
			}
			for _, pMatch := range pattern.Matches() {
				pInfo.Matches = append(pInfo.Matches, YaraXMatchInfo{
					Offset: pMatch.Offset(),
					Length: pMatch.Length(),
				})
			}
			match.Patterns = append(match.Patterns, pInfo)
		}

		*allMatches = append(*allMatches, match)
	}
}

var (
	_ Action    = &ActionYaraX{}
	_ io.Closer = &ActionYaraX{}
)
