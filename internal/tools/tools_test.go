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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AkashNaickar/meshery-mcp-demo/internal/meshery"
)

// newTestMC builds an MCP server with all tools registered, backed by a mock
// Meshery Server.
func newTestMC(t *testing.T, handler http.HandlerFunc) *server.MCPServer {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	mc, err := meshery.New(meshery.Options{BaseURL: srv.URL, Token: "test-token"})
	if err != nil {
		t.Fatalf("meshery.New: %v", err)
	}

	s := server.NewMCPServer("test", "0.0.0")
	if err := Register(s, mc); err != nil {
		t.Fatalf("Register: %v", err)
	}
	return s
}

// callTool invokes a tool handler directly through the server's tool map.
func callTool(t *testing.T, s *server.MCPServer, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	tool := s.GetTool(name)
	if tool == nil {
		t.Fatalf("tool %q not registered", name)
	}
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	}
	res, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	return res
}

func TestListDesignsTool(t *testing.T) {
	s := newTestMC(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/pattern" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"page":1,"page_size":25,"count":1,"patterns":[{"id":"a","name":"demo","updated_at":"2026-01-01T00:00:00Z"}]}`))
	})

	res := callTool(t, s, "list_designs", map[string]any{})
	if res.IsError {
		t.Fatalf("unexpected error result: %+v", res)
	}

	encoded, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured: %v", err)
	}
	var designs []map[string]any
	if err := json.Unmarshal(encoded, &designs); err != nil {
		t.Fatalf("decode structured: %v", err)
	}
	if len(designs) != 1 {
		t.Errorf("len(designs) = %d, want 1", len(designs))
	}
	if designs[0]["name"] != "demo" {
		t.Errorf("design name = %v, want demo", designs[0]["name"])
	}
}

func TestListKubernetesContextsTool(t *testing.T) {
	s := newTestMC(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/system/kubernetes/contexts" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total_count":1,"contexts":[{"id":"c1","name":"docker-desktop","server":"https://127.0.0.1:6443"}]}`))
	})

	res := callTool(t, s, "list_kubernetes_contexts", map[string]any{})
	if res.IsError {
		t.Fatalf("unexpected error result")
	}
	if !strings.Contains(res.Content[0].(mcp.TextContent).Text, "1 Kubernetes context") {
		t.Errorf("fallback text = %q", res.Content[0].(mcp.TextContent).Text)
	}
}

func TestDeployDesignTool(t *testing.T) {
	s := newTestMC(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/pattern/deploy" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"d1","name":"demo","deployed":[{"kind":"Deployment","name":"emoji","status":"applied"}]}`))
	})

	res := callTool(t, s, "deploy_design", map[string]any{
		"pattern_file": "version: 1.0\nservices: []",
		"dry_run":      false,
	})
	if res.IsError {
		t.Fatalf("unexpected error result")
	}
	if !strings.Contains(res.Content[0].(mcp.TextContent).Text, "deployed 1 resource") {
		t.Errorf("fallback text = %q", res.Content[0].(mcp.TextContent).Text)
	}
}

func TestDeployDesignToolMissingPattern(t *testing.T) {
	s := newTestMC(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be called")
	})

	res := callTool(t, s, "deploy_design", map[string]any{})
	if !res.IsError {
		t.Fatalf("expected error result for missing pattern_file")
	}
}

func TestServerInfoTool(t *testing.T) {
	s := newTestMC(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server_info must not call Meshery")
	})

	res := callTool(t, s, "server_info", map[string]any{})
	if res.IsError {
		t.Fatalf("unexpected error result")
	}
	if !strings.Contains(res.Content[0].(mcp.TextContent).Text, "0.1.0") {
		t.Errorf("server_info text = %q", res.Content[0].(mcp.TextContent).Text)
	}
}
