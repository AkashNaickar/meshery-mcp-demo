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
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestRegisterAndGetPrompt(t *testing.T) {
	s := server.NewMCPServer("test", "0.0.0", server.WithHooks(&server.Hooks{}))
	if err := Register(s); err != nil {
		t.Fatalf("Register: %v", err)
	}

	prompts := s.ListPrompts()
	entry, ok := prompts["deploy_application"]
	if !ok {
		t.Fatalf("deploy_application prompt not registered")
	}

	result, err := entry.Handler(context.Background(), mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{Name: "deploy_application"},
	})
	if err != nil {
		t.Fatalf("get prompt: %v", err)
	}

	if len(result.Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1", len(result.Messages))
	}

	text, ok := result.Messages[0].Content.(mcp.TextContent)
	if !ok {
		t.Fatalf("content type = %T, want TextContent", result.Messages[0].Content)
	}
	for _, want := range []string{"list_kubernetes_contexts", "list_designs", "deploy_design", "topology"} {
		if !strings.Contains(text.Text, want) {
			t.Errorf("prompt text missing %q", want)
		}
	}
}

func TestClusterHealthAuditPrompt(t *testing.T) {
	s := server.NewMCPServer("test", "0.0.0", server.WithHooks(&server.Hooks{}))
	if err := Register(s); err != nil {
		t.Fatalf("Register: %v", err)
	}

	prompts := s.ListPrompts()
	entry, ok := prompts["cluster_health_audit"]
	if !ok {
		t.Fatalf("cluster_health_audit prompt not registered")
	}

	result, err := entry.Handler(context.Background(), mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{Name: "cluster_health_audit"},
	})
	if err != nil {
		t.Fatalf("get prompt: %v", err)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1", len(result.Messages))
	}
	text, ok := result.Messages[0].Content.(mcp.TextContent)
	if !ok {
		t.Fatalf("content type = %T", result.Messages[0].Content)
	}
	for _, want := range []string{"get_cluster_resources", "topology", "drift"} {
		if !strings.Contains(text.Text, want) {
			t.Errorf("prompt text missing %q", want)
		}
	}
}
