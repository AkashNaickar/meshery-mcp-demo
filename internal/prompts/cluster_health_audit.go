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

package prompts

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerClusterHealthAudit registers a guided prompt that audits cluster
// health by reading live topology and resources.
func registerClusterHealthAudit(s *server.MCPServer) error {
	prompt := mcp.NewPrompt("cluster_health_audit",
		mcp.WithPromptTitle("Cluster health audit"),
		mcp.WithPromptDescription("Audit a Kubernetes context: read live topology, inspect workloads, and surface configuration drift or optimization recommendations."),
	)
	s.AddPrompt(prompt, clusterHealthAuditHandler)
	return nil
}

// clusterHealthAuditHandler returns the guided health audit workflow.
func clusterHealthAuditHandler(_ context.Context, _ mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	const instructions = `You are performing a health audit of a Kubernetes cluster managed through Meshery.

1. Call list_kubernetes_contexts and identify the target context.
2. Read the cluster topology resource: meshery://clusters/{cluster_id}/topology.
3. Call get_cluster_resources to inspect live pods, services, and deployments.
4. Analyze the results for:
   - Configuration drift: resources whose live spec differs from the expected design.
   - Health signals: failed or pending pods, unavailable replicas, services with no backing pods.
   - Optimization opportunities: under/over-provisioned replicas, unused services, large unmanaged resources.
5. Produce a concise report grouped by: Healthy, Needs Attention, and Recommendations.

Never invent resources or metrics. Base every finding on data returned by the tools
and resources.`

	return &mcp.GetPromptResult{
		Description: "Audit cluster health and surface drift or optimization recommendations.",
		Messages: []mcp.PromptMessage{
			{Role: mcp.RoleUser, Content: mcp.TextContent{Type: "text", Text: instructions}},
		},
	}, nil
}
