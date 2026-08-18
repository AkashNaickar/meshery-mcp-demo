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

package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AkashNaickar/meshery-mcp-demo/internal/meshery"
)

// registerGetClusterResources registers the get_cluster_resources tool, which
// inspects live MeshSync-discovered workloads for post-deployment verification.
func registerGetClusterResources(s *server.MCPServer, mc *meshery.Client) error {
	tool := mcp.NewTool("get_cluster_resources",
		mcp.WithDescription("Query MeshSync state to inspect live workloads (pods, services, deployments) on a Kubernetes context."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithString("context_id",
			mcp.Description("The ID of the Kubernetes context to inspect. Empty uses the server's current context."),
		),
		mcp.WithString("namespace",
			mcp.Description("Optional namespace filter; empty returns all namespaces."),
		),
		mcp.WithString("kind",
			mcp.Description("Optional resource kind filter (e.g. Pod, Service, Deployment)."),
		),
	)
	s.AddTool(tool, getClusterResourcesHandler(mc))
	return nil
}

// getClusterResourcesHandler returns live cluster resources, sanitized.
func getClusterResourcesHandler(mc *meshery.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clusterID := req.GetString("context_id", "")
		namespace := req.GetString("namespace", "")
		kind := req.GetString("kind", "")

		resources, err := mc.GetClusterResources(ctx, clusterID, namespace, kind)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get cluster resources", err), nil
		}

		type view struct {
			Kind      string         `json:"kind"`
			Name      string         `json:"name"`
			Namespace string         `json:"namespace,omitempty"`
			Status    string         `json:"status,omitempty"`
			Data      map[string]any `json:"data,omitempty"`
		}
		items := make([]view, 0, len(resources))
		for _, r := range resources {
			items = append(items, view{
				Kind:      r.Kind,
				Name:      r.Name,
				Namespace: r.Namespace,
				Status:    r.Status,
				Data:      r.Data,
			})
		}

		fallback := fmt.Sprintf("%d resource(s) found", len(items))
		return mcp.NewToolResultStructured(map[string]any{
			"resources": items,
			"count":     len(items),
		}, fallback), nil
	}
}
