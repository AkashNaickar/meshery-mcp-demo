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

import "time"

// DesignDetail is a single stored design including its raw PatternFile
// payload. It backs the meshery://designs/{design_id} resource.
type DesignDetail struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SchemaVer   string `json:"schema_version,omitempty"`
	PatternFile string `json:"pattern_file"`
	// SourceType is the design's import type (e.g. "Kubernetes Manifest").
	SourceType string `json:"source_type,omitempty"`
}

// ValidationIssue is one finding from validating a PatternFile.
type ValidationIssue struct {
	// Severity is "error" or "warning".
	Severity string `json:"severity"`
	Message  string `json:"message"`
	// Field is the design field the issue applies to, when known.
	Field string `json:"field,omitempty"`
}

// ValidationResult is the outcome of validate_design.
type ValidationResult struct {
	// Valid reports whether the design passed validation.
	Valid bool `json:"valid"`
	// Issues holds errors and warnings.
	Issues []ValidationIssue `json:"issues"`
}

// ClusterResource is one MeshSync-discovered resource in a cluster.
type ClusterResource struct {
	Kind      string            `json:"kind"`
	Name      string            `json:"name"`
	Namespace string            `json:"namespace,omitempty"`
	Status    string            `json:"status,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	// Data carries kind-specific detail (e.g. container count for Pods). It is
	// sanitized before being returned to the agent.
	Data map[string]any `json:"data,omitempty"`
}

// DesignSearchOptions configures a search over stored designs.
type DesignSearchOptions struct {
	// Search is a free-text filter applied by the server.
	Search string
	// Page is the 1-based page number.
	Page int
	// PageSize is the number of designs per page.
	PageSize int
}

// UndeployRequest describes tearing down a deployed design.
type UndeployRequest struct {
	// PatternID references the stored design to undeploy.
	PatternID string
	// PatternFile is an optional inline design; PatternID takes precedence.
	PatternFile string
	// Contexts are the target Kubernetes context IDs.
	Contexts []string
	// Force ignores confirmation guards and undeploys immediately.
	Force bool
}

// UndeployResponse is the result of tearing down a design.
type UndeployResponse struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Messages   []string       `json:"messages"`
	Removed    []DeployedItem `json:"removed"`
	NotRemoved []DeployedItem `json:"not_removed"`
	Error      string         `json:"error,omitempty"`
}

// UpdatedAt returns the design's last modification time in a stable format.
func (d Design) UpdatedAtRFC3339() string {
	if d.UpdatedAt.IsZero() {
		return ""
	}
	return d.UpdatedAt.Format(time.RFC3339)
}
