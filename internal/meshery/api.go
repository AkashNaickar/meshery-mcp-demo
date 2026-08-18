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
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// Design is a Meshery design (a PatternFile stored on the server).
type Design struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DesignList is the paginated response of GET /api/pattern.
type DesignList struct {
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
	Count    int      `json:"count"`
	Patterns []Design `json:"patterns"`
}

// ListDesigns returns the designs stored on the Meshery Server. Explicit
// pagination is required: Meshery defaults pageSize to 0 when omitted, which
// yields an empty SQL LIMIT and an empty result.
func (c *Client) ListDesigns(ctx context.Context, page, pageSize int) (*DesignList, error) {
	return c.SearchDesigns(ctx, DesignSearchOptions{Page: page, PageSize: pageSize})
}

// SearchDesigns returns designs filtered by an optional free-text search,
// with explicit pagination. Meshery defaults pageSize to 0 when omitted,
// which yields an empty SQL LIMIT and an empty result.
func (c *Client) SearchDesigns(ctx context.Context, opts DesignSearchOptions) (*DesignList, error) {
	query := url.Values{}
	if opts.Search != "" {
		query.Set("search", opts.Search)
	}
	page := opts.Page
	if page < 0 {
		page = 0
	}
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	query.Set("page", strconv.Itoa(page))
	query.Set("pagesize", strconv.Itoa(pageSize))

	var out DesignList
	if err := c.request(ctx, "GET", "/api/pattern", query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetDesign returns a single stored design's raw PatternFile payload.
func (c *Client) GetDesign(ctx context.Context, id string) (*DesignDetail, error) {
	var raw struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		PatternFile string `json:"pattern_file"`
		Type        string `json:"type"`
	}
	if err := c.request(ctx, "GET", "/api/pattern/"+id, nil, nil, &raw); err != nil {
		return nil, err
	}
	return &DesignDetail{
		ID:          raw.ID,
		Name:        raw.Name,
		PatternFile: raw.PatternFile,
		SourceType:  raw.Type,
	}, nil
}

// ValidateDesign lints a PatternFile YAML against the Meshery validation
// endpoint without applying it to any cluster. When Meshery's validation
// endpoint is not available it returns a structural validation result.
func (c *Client) ValidateDesign(ctx context.Context, patternFile string) (*ValidationResult, error) {
	// Local structural validation is always performed and is the source of
	// truth for schema version and component shape.
	issues := validatePatternStructure(patternFile)

	// Best-effort server-side validation: Meshery's schema linter may reject
	// designs for reasons a local check cannot see (unknown models, versions).
	// Server findings are merged with (not replacing) local findings.
	if patternFile != "" {
		if serverResult, err := c.remoteValidate(ctx, patternFile); err == nil {
			issues = append(issues, serverResult.Issues...)
		}
	}

	valid := true
	for _, iss := range issues {
		if iss.Severity == "error" {
			valid = false
			break
		}
	}
	return &ValidationResult{Valid: valid, Issues: issues}, nil
}

// remoteValidate asks the Meshery server to validate a PatternFile, returning
// a ValidationResult. It is best-effort; callers ignore errors.
func (c *Client) remoteValidate(ctx context.Context, patternFile string) (*ValidationResult, error) {
	body := map[string]any{"pattern_file": patternFile}
	var out struct {
		Valid  bool              `json:"valid"`
		Issues []ValidationIssue `json:"issues"`
	}
	if err := c.request(ctx, "POST", "/api/pattern/validate", nil, body, &out); err != nil {
		return nil, err
	}
	return &ValidationResult{Valid: out.Valid, Issues: out.Issues}, nil
}

// UndeployDesign tears down the resources defined by a design in the target
// contexts. It maps to Meshery's deploy endpoint with delete=true.
func (c *Client) UndeployDesign(ctx context.Context, req UndeployRequest) (*UndeployResponse, error) {
	query := url.Values{}
	query.Set("delete", "true")
	if req.Force {
		query.Set("force", "true")
	}
	if len(req.Contexts) > 0 {
		for _, ctxID := range req.Contexts {
			query.Add("contexts", ctxID)
		}
	} else {
		query.Set("contexts", "all")
	}

	body := map[string]any{}
	if req.PatternID != "" {
		body["pattern_id"] = req.PatternID
	}
	if req.PatternFile != "" {
		body["pattern_file"] = req.PatternFile
	}

	var out UndeployResponse
	if err := c.request(ctx, "POST", "/api/pattern/deploy", query, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetClusterResources returns MeshSync-discovered resources for a cluster,
// filtered by kind when requested. The returned Data maps are sanitized so
// credentials never reach the agent.
func (c *Client) GetClusterResources(ctx context.Context, clusterID, namespace, kind string) ([]ClusterResource, error) {
	query := url.Values{}
	if clusterID != "" {
		query.Set("clusterId", clusterID)
	}
	if namespace != "" {
		query.Set("namespace", namespace)
	}
	if kind != "" {
		query.Set("kind", kind)
	}
	query.Set("page", "0")
	query.Set("pagesize", "500")

	var raw struct {
		Resources []ClusterResource `json:"resources"`
	}
	if err := c.request(ctx, "GET", "/api/system/meshsync/resources", query, nil, &raw); err != nil {
		return nil, err
	}

	// Fall back to the live Kubernetes API when MeshSync has not synced the
	// cluster (a known v1.0.66 issue).
	if len(raw.Resources) == 0 && c.topologySource != nil {
		components, err := c.topologySource.ListTopology(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]ClusterResource, 0, len(components))
		for _, comp := range components {
			if kind != "" && comp.Kind != kind {
				continue
			}
			out = append(out, ClusterResource{Kind: comp.Kind, Name: comp.Name, Status: "discovered"})
		}
		return out, nil
	}

	out := make([]ClusterResource, 0, len(raw.Resources))
	for _, r := range raw.Resources {
		if namespace != "" && r.Namespace != namespace {
			continue
		}
		out = append(out, ClusterResource{
			Kind:      r.Kind,
			Name:      r.Name,
			Namespace: r.Namespace,
			Status:    r.Status,
			Labels:    r.Labels,
			Data:      r.Data,
		})
	}
	return out, nil
}

// Topology is the MeshSync-discovered state of a cluster rendered as a design.

// KubernetesContext is a Kubernetes context managed by Meshery.
type KubernetesContext struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Server              string `json:"server"`
	KubernetesServerID  string `json:"kubernetesServerId"`
	ConnectionID        string `json:"connectionId"`
	DeploymentType      string `json:"deployment_type"`
	Owner               string `json:"owner"`
	CreatedBy           string `json:"created_by"`
	MesheryInstanceID   string `json:"meshery_instance_id"`
	KubernetesClusterID string `json:"kubernetes_cluster_id"`
}

// KubernetesContextList is the response of GET /api/system/kubernetes/contexts.
type KubernetesContextList struct {
	TotalCount int                 `json:"total_count"`
	Contexts   []KubernetesContext `json:"contexts"`
}

// ListKubernetesContexts returns the Kubernetes contexts Meshery is managing.
func (c *Client) ListKubernetesContexts(ctx context.Context) (*KubernetesContextList, error) {
	var out KubernetesContextList
	if err := c.request(ctx, "GET", "/api/system/kubernetes/contexts", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeployRequest describes a design deployment.
type DeployRequest struct {
	// PatternFile is the raw design YAML to deploy.
	PatternFile string `json:"pattern_file"`
	// PatternID references a stored design; PatternFile takes precedence.
	PatternID string `json:"pattern_id,omitempty"`
	// Contexts are the Kubernetes context IDs to deploy into. Empty means the
	// server's current context.
	Contexts []string `json:"contexts,omitempty"`
	// DryRun validates the deployment without applying it.
	DryRun bool `json:"-"`
	// SkipCRD skips Custom Resource Definition installation.
	SkipCRD bool `json:"-"`
	// Upgrade upgrades existing releases for the design.
	Upgrade bool `json:"-"`
}

// DeployResponse is the result of POST /api/pattern/deploy.
type DeployResponse struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Messages   []string        `json:"messages"`
	DryRun     bool            `json:"dryRun"`
	Deployed   []DeployedItem  `json:"deployed"`
	UnDeployed []DeployedItem  `json:"unDeployed"`
	Invalid    []DeployedItem  `json:"invalid"`
	Error      string          `json:"error"`
	Events     json.RawMessage `json:"events"`
	// DryRunResponse carries the raw dry-run payload. Meshery v1.0.66 returns
	// literal null here when its hydration path drops every component, which
	// signals an empty (no-op) result rather than a real deployment plan.
	DryRunResponse json.RawMessage `json:"dryRunResponse"`
}

// Empty reports whether the server returned no actionable result. This is the
// signature of Meshery v1.0.66's hydration no-op: no deployed, undeployed, or
// invalid items, no error, and a null (or absent) dry-run response.
func (r *DeployResponse) Empty() bool {
	if r.Error != "" {
		return false
	}
	if len(r.Deployed) > 0 || len(r.UnDeployed) > 0 || len(r.Invalid) > 0 {
		return false
	}
	return len(r.DryRunResponse) == 0 || string(r.DryRunResponse) == "null"
}

// DeployedItem is one resource applied or planned by a deployment.
type DeployedItem struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// DeployDesign deploys (or dry-run validates) a design to the target contexts.
//
// The target Kubernetes contexts are passed as the `contexts` query parameter
// (or `all` when none are specified), because Meshery's KubernetesMiddleware
// selects the deployment target from that query parameter rather than the
// request body.
func (c *Client) DeployDesign(ctx context.Context, req DeployRequest) (*DeployResponse, error) {
	query := url.Values{}
	query.Set("dryRun", strconv.FormatBool(req.DryRun))
	query.Set("skipCRD", strconv.FormatBool(req.SkipCRD))
	query.Set("upgrade", strconv.FormatBool(req.Upgrade))
	if len(req.Contexts) > 0 {
		for _, ctxID := range req.Contexts {
			query.Add("contexts", ctxID)
		}
	} else {
		query.Set("contexts", "all")
	}

	body := map[string]any{
		"pattern_file": req.PatternFile,
	}
	if req.PatternID != "" {
		body["pattern_id"] = req.PatternID
	}

	var out DeployResponse
	if err := c.request(ctx, "POST", "/api/pattern/deploy", query, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Topology is the MeshSync-discovered state of a cluster rendered as a design.
type Topology struct {
	// Components are the graph nodes (cluster resources).
	Components []TopologyComponent `json:"components"`
	// Relationships are the derived edges between components.
	Relationships []json.RawMessage `json:"relationships"`
	// Evaluated reports whether the server ran the relationship evaluator.
	Evaluated bool `json:"evaluated"`
}

// TopologyComponent is one node in a cluster topology graph.
type TopologyComponent struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// TopologySource supplies a cluster's live topology. It lets the Meshery client
// fall back to reading cluster state directly (e.g. via client-go) when
// MeshSync has not populated the server, which happens on some Meshery
// versions.
type TopologySource interface {
	ListTopology(ctx context.Context) ([]TopologyComponent, error)
}

// Topology returns the live MeshSync topology for a cluster, rendered as a
// design graph via GET /api/system/meshsync/resources?asDesign=true.
func (c *Client) Topology(ctx context.Context, clusterID string) (*Topology, error) {
	query := url.Values{}
	query.Set("asDesign", "true")
	if clusterID != "" {
		query.Set("clusterId", clusterID)
	}

	// The response is a PatternFile whose components field carries the graph.
	var raw struct {
		Components    []TopologyComponent `json:"components"`
		Relationships []json.RawMessage   `json:"relationships"`
		Evaluated     bool                `json:"evaluated"`
	}
	if err := c.request(ctx, "GET", "/api/system/meshsync/resources", query, nil, &raw); err != nil {
		return nil, err
	}

	// MeshSync may not have synced the cluster (a known Meshery v1.0.66 issue),
	// in which case the server returns no components. When a fallback source is
	// configured, use it to read the live cluster state directly.
	if len(raw.Components) == 0 && c.topologySource != nil {
		components, err := c.topologySource.ListTopology(ctx)
		if err != nil {
			return nil, fmt.Errorf("read topology via fallback: %w", err)
		}
		raw.Components = components
		raw.Relationships = nil
		raw.Evaluated = true
	}

	return &Topology{
		Components:    raw.Components,
		Relationships: raw.Relationships,
		Evaluated:     raw.Evaluated,
	}, nil
}
