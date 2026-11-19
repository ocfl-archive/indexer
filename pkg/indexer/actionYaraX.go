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

// YaraXMatchInfo contains detailed offset and length information for pattern matches.
type YaraXMatchInfo struct {
	Offset uint64 `json:"offset"`
	Length uint64 `json:"length"`
}

// YaraXPatternInfo contains matched patterns and their match spans for a rule.
type YaraXPatternInfo struct {
	Identifier string           `json:"identifier"`
	Matches    []YaraXMatchInfo `json:"matches,omitempty"`
}

// YaraXRuleMatch represents the structured result of a matched YARA-X rule.
type YaraXRuleMatch struct {
	Rule      string             `json:"rule"`
	Namespace string             `json:"namespace,omitempty"`
	Tags      []string           `json:"tags,omitempty"`
	Metadata  map[string]any     `json:"metadata,omitempty"`
	Patterns  []YaraXPatternInfo `json:"patterns,omitempty"`
}
