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

// Package prompts implements the MCP prompt surface of the Meshery MCP demo
// server. Prompts steer agents through guided workflows.
package prompts

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Register registers every prompt exposed by the server.
func Register(s *server.MCPServer) error {
	return registerDeployApplication(s)
}

// registerDeployApplication registers a guided prompt for deploying a design.
func registerDeployApplication(s *server.MCPServer) error {
	prompt := mcp.NewPrompt("deploy_application",
		mcp.WithPromptTitle("Deploy an application"),
		mcp.WithPromptDescription("Guided workflow for deploying a design to a Kubernetes context managed by Meshery."),
	)
	s.AddPrompt(prompt, deployApplicationHandler)
	return nil
}

// deployApplicationHandler returns a guided deployment prompt.
func deployApplicationHandler(_ context.Context, _ mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	const instructions = `You are operating a Meshery MCP server that manages Kubernetes clusters through Meshery.

To deploy an application design:

1. Call list_kubernetes_contexts to see which clusters Meshery is managing.
2. Call list_designs to find the design to deploy, or accept a design YAML from the user.
3. Call deploy_design with the pattern_file argument set to the design YAML, and
   context_id set to the target cluster if one was chosen.
4. If the user wants to validate first, set dry_run to true.
5. After deployment, confirm the result by reading the cluster topology resource:
   meshery://clusters/{cluster_id}/topology.

Never invent designs, contexts, or IDs. Work only with data returned by the tools
and resources.`

	return &mcp.GetPromptResult{
		Description: "Guided workflow for deploying a design to a Kubernetes context managed by Meshery.",
		Messages: []mcp.PromptMessage{
			{Role: mcp.RoleUser, Content: mcp.TextContent{Type: "text", Text: instructions}},
		},
	}, nil
}
