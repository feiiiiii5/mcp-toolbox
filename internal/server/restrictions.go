// Copyright 2026 Google LLC
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

package server

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	// AllowedSourcesEnvVar names the environment variable holding a
	// comma-separated allowlist of source types. When it is unset or empty,
	// every source type is allowed.
	AllowedSourcesEnvVar = "TOOLBOX_ALLOWED_SOURCES"
	// DisallowedToolsEnvVar names the environment variable holding a
	// comma-separated denylist of tool types. When it is unset or empty, every
	// tool type is allowed.
	DisallowedToolsEnvVar = "TOOLBOX_DISALLOWED_TOOLS"
)

// configRestrictions holds the source and tool type restrictions read from the
// environment. A nil allowedSources means no source restriction. A non-nil but
// empty allowedSources allows no source types.
type configRestrictions struct {
	allowedSources  map[string]struct{}
	disallowedTools map[string]struct{}
}

// parseTypeList parses a comma-separated list of types. It trims whitespace
// around each entry and drops empty entries. It returns nil when the whole
// value is empty or only whitespace, which means the list is not set.
func parseTypeList(value string) map[string]struct{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	set := make(map[string]struct{})
	for _, entry := range strings.Split(value, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		set[entry] = struct{}{}
	}
	return set
}

// restrictionsFromEnv reads the source and tool restrictions from the
// TOOLBOX_ALLOWED_SOURCES and TOOLBOX_DISALLOWED_TOOLS environment variables.
func restrictionsFromEnv() configRestrictions {
	return configRestrictions{
		allowedSources:  parseTypeList(os.Getenv(AllowedSourcesEnvVar)),
		disallowedTools: parseTypeList(os.Getenv(DisallowedToolsEnvVar)),
	}
}

// active reports whether any restriction is set.
func (r configRestrictions) active() bool {
	return r.allowedSources != nil || len(r.disallowedTools) > 0
}

// validate checks the source and tool types in cfg against the restrictions.
// It runs on the raw configs, before any source opens a connection. It returns
// one error that lists every violation in sorted order.
func (r configRestrictions) validate(cfg ServerConfig) error {
	var violations []string

	if r.allowedSources != nil {
		names := make([]string, 0, len(cfg.SourceConfigs))
		for name := range cfg.SourceConfigs {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			sc := cfg.SourceConfigs[name]
			if sc == nil {
				continue
			}
			sourceType := sc.SourceConfigType()
			if _, ok := r.allowedSources[sourceType]; !ok {
				violations = append(violations, fmt.Sprintf("source %q has type %q, which is not allowed by %s", name, sourceType, AllowedSourcesEnvVar))
			}
		}
	}

	if len(r.disallowedTools) > 0 {
		names := make([]string, 0, len(cfg.ToolConfigs))
		for name := range cfg.ToolConfigs {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			tc := cfg.ToolConfigs[name]
			if tc == nil {
				continue
			}
			toolType := tc.ToolConfigType()
			if _, ok := r.disallowedTools[toolType]; ok {
				violations = append(violations, fmt.Sprintf("tool %q has type %q, which is disallowed by %s", name, toolType, DisallowedToolsEnvVar))
			}
		}
	}

	if len(violations) > 0 {
		return fmt.Errorf("config contains disallowed sources or tools: %s", strings.Join(violations, "; "))
	}
	return nil
}

// sortedKeys returns the keys of set in sorted order.
func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
