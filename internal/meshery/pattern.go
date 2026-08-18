// Copyright Meshery Authors
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

package meshery

import (
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// pattern mirrors the subset of a Meshery PatternFile (design) needed to
// enumerate its components. Only the fields the deploy fallback reads are
// modeled; everything else is ignored.
type pattern struct {
	Name       string             `yaml:"name"`
	SchemaVer  string             `yaml:"schemaVersion"`
	Components []patternComponent `yaml:"components"`
}

// patternComponent is one resource declared by a design.
type patternComponent struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

// ParsePatternComponents extracts the declared components from a design's
// PatternFile YAML. It is the exported form of parsePatternComponents, used by
// the deploy_design fallback when the server returns an empty result.
func ParsePatternComponents(yml string) []DeployedItem {
	return parsePatternComponents(yml)
}

// patternSchemaVersion is the supported PatternFile schema.
const patternSchemaVersion = "designs.meshery.io/v1beta3"

// validatePatternStructure lints a PatternFile YAML structurally: it checks
// the schema version and that every component declares a name and type. It
// never requires a reachable Meshery server.
func validatePatternStructure(yml string) []ValidationIssue {
	if strings.TrimSpace(yml) == "" {
		return []ValidationIssue{{Severity: "error", Message: "pattern_file is empty"}}
	}

	var p struct {
		Name       string `yaml:"name"`
		SchemaVer  string `yaml:"schemaVersion"`
		Components []struct {
			Name string `yaml:"name"`
			Type string `yaml:"type"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal([]byte(yml), &p); err != nil {
		return []ValidationIssue{{Severity: "error", Message: "invalid YAML: " + err.Error()}}
	}

	var issues []ValidationIssue
	if p.Name == "" {
		issues = append(issues, ValidationIssue{Severity: "warning", Field: "name", Message: "design has no name"})
	}
	if p.SchemaVer == "" {
		issues = append(issues, ValidationIssue{Severity: "warning", Field: "schemaVersion", Message: "schemaVersion is empty; expected " + patternSchemaVersion})
	} else if p.SchemaVer != patternSchemaVersion {
		issues = append(issues, ValidationIssue{Severity: "warning", Field: "schemaVersion", Message: "schemaVersion " + p.SchemaVer + " differs from " + patternSchemaVersion})
	}
	if len(p.Components) == 0 {
		issues = append(issues, ValidationIssue{Severity: "error", Field: "components", Message: "design declares no components"})
	}
	for i, comp := range p.Components {
		if comp.Name == "" {
			issues = append(issues, ValidationIssue{Severity: "error", Field: "components", Message: "component[" + strconv.Itoa(i) + "] has no name"})
		}
		if comp.Type == "" {
			issues = append(issues, ValidationIssue{Severity: "error", Field: "components", Message: "component[" + strconv.Itoa(i) + "] has no type"})
		}
	}

	if len(issues) == 0 {
		issues = append(issues, ValidationIssue{Severity: "info", Message: "design structure is valid"})
	}
	return issues
}

// parsePatternComponents extracts the declared components from a design's
// PatternFile YAML. It returns an empty slice (never an error) for input that
// is not a parseable design, so callers can fall through gracefully.
func parsePatternComponents(yml string) []DeployedItem {
	var p pattern
	if err := yaml.Unmarshal([]byte(yml), &p); err != nil {
		return nil
	}

	out := make([]DeployedItem, 0, len(p.Components))
	for _, c := range p.Components {
		kind := c.Type
		if kind == "" {
			kind = "unknown"
		}
		out = append(out, DeployedItem{
			Kind:   kind,
			Name:   c.Name,
			Status: "planned",
		})
	}
	return out
}
