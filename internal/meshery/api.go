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

// ListDesigns returns the designs stored on the Meshery Server.
func (c *Client) ListDesigns(ctx context.Context, page, pageSize int) (*DesignList, error) {
	query := url.Values{}
	if page > 0 {
		query.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		query.Set("pagesize", strconv.Itoa(pageSize))
	}

	var out DesignList
	if err := c.request(ctx, "GET", "/api/pattern", query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

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
}

// DeployedItem is one resource applied or planned by a deployment.
type DeployedItem struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// DeployDesign deploys (or dry-run validates) a design to the target contexts.
func (c *Client) DeployDesign(ctx context.Context, req DeployRequest) (*DeployResponse, error) {
	query := url.Values{}
	query.Set("dryRun", strconv.FormatBool(req.DryRun))
	query.Set("skipCRD", strconv.FormatBool(req.SkipCRD))
	query.Set("upgrade", strconv.FormatBool(req.Upgrade))

	body := map[string]any{
		"pattern_file": req.PatternFile,
	}
	if req.PatternID != "" {
		body["pattern_id"] = req.PatternID
	}
	if len(req.Contexts) > 0 {
		body["contexts"] = req.Contexts
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

	return &Topology{
		Components:    raw.Components,
		Relationships: raw.Relationships,
		Evaluated:     raw.Evaluated,
	}, nil
}
