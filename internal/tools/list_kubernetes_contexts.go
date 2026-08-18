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
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AkashNaickar/meshery-mcp-demo/internal/meshery"
)

// registerListKubernetesContexts registers the list_kubernetes_contexts tool.
func registerListKubernetesContexts(s *server.MCPServer, mc *meshery.Client) error {
	tool := mcp.NewTool("list_kubernetes_contexts",
		mcp.WithDescription("List the Kubernetes contexts that Meshery is managing."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	)
	s.AddTool(tool, listKubernetesContextsHandler(mc))
	return nil
}

// listKubernetesContextsHandler returns the Kubernetes contexts Meshery manages.
func listKubernetesContextsHandler(mc *meshery.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		list, err := mc.ListKubernetesContexts(ctx)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("list kubernetes contexts", err), nil
		}

		type contextView struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			Server       string `json:"server"`
			Deployment   string `json:"deployment_type,omitempty"`
			ConnectionID string `json:"connection_id,omitempty"`
		}

		contexts := make([]contextView, 0, len(list.Contexts))
		for _, c := range list.Contexts {
			contexts = append(contexts, contextView{
				ID:           c.ID,
				Name:         c.Name,
				Server:       c.Server,
				Deployment:   c.DeploymentType,
				ConnectionID: c.ConnectionID,
			})
		}

		var names []string
		for _, c := range contexts {
			names = append(names, c.Name)
		}
		fallback := fmt.Sprintf("%d Kubernetes context(s) found: %s", len(contexts), strings.Join(names, ", "))
		return mcp.NewToolResultStructured(map[string]any{
			"contexts": contexts,
			"count":    len(contexts),
		}, fallback), nil
	}
}
