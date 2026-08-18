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

func TestDeployDesignToolEmptyFallback(t *testing.T) {
	// The v1.0.66 no-op signature: Meshery returns a null dry-run response with
	// no deployed items. The handler should fall back to the design's declared
	// components so the client still gets a meaningful result.
	s := newTestMC(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/pattern/deploy" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"d1","name":"demo","dryRun":true,"dryRunResponse":null}`))
	})

	const design = "name: demo\nschemaVersion: designs.meshery.io/v1beta3\ncomponents:\n- name: web\n  type: Deployment\n- name: web-svc\n  type: Service\n"

	res := callTool(t, s, "deploy_design", map[string]any{
		"pattern_file": design,
		"dry_run":      true,
	})
	if res.IsError {
		t.Fatalf("unexpected error result")
	}

	// Structured content should list the locally-parsed components.
	encoded, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(encoded, &parsed); err != nil {
		t.Fatalf("decode structured: %v", err)
	}

	deployed, ok := parsed["deployed"].([]any)
	if !ok || len(deployed) != 2 {
		t.Fatalf("deployed = %v, want 2 fallback components", parsed["deployed"])
	}
	if _, ok := parsed["note"]; !ok {
		t.Errorf("expected fallback note")
	}

	text := res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "validated (dry-run) 2 resource") {
		t.Errorf("fallback text = %q", text)
	}
}

func TestDeployDesignToolNoFallbackWhenPopulated(t *testing.T) {
	// When Meshery returns real deployed items, the fallback must not fire.
	s := newTestMC(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"d1","name":"demo","deployed":[{"kind":"Deployment","name":"emoji","status":"applied"}]}`))
	})

	res := callTool(t, s, "deploy_design", map[string]any{
		"pattern_file": "name: demo\ncomponents:\n- name: other\n  type: Deployment\n",
		"dry_run":      false,
	})
	if res.IsError {
		t.Fatalf("unexpected error result")
	}

	text := res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "deployed 1 resource") {
		t.Errorf("fallback text = %q", text)
	}
	if strings.Contains(text, "parsed locally") {
		t.Errorf("fallback should not fire when server returns items: %q", text)
	}
}

func TestValidateDesignTool(t *testing.T) {
	s := newTestMC(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/pattern/validate" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true,"issues":[]}`))
	})

	res := callTool(t, s, "validate_design", map[string]any{
		"pattern_file": "name: demo\nschemaVersion: designs.meshery.io/v1beta3\ncomponents:\n- name: web\n  type: Deployment\n",
	})
	if res.IsError {
		t.Fatalf("unexpected error result")
	}
	if !strings.Contains(res.Content[0].(mcp.TextContent).Text, "valid") {
		t.Errorf("fallback text = %q", res.Content[0].(mcp.TextContent).Text)
	}
}

func TestValidateDesignToolMissingPattern(t *testing.T) {
	s := newTestMC(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be called")
	})

	res := callTool(t, s, "validate_design", map[string]any{})
	if !res.IsError {
		t.Fatalf("expected error for missing pattern_file")
	}
}

func TestValidateDesignToolStructuralError(t *testing.T) {
	// Server returns valid, but the design is structurally invalid (no type).
	s := newTestMC(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true,"issues":[]}`))
	})

	res := callTool(t, s, "validate_design", map[string]any{
		"pattern_file": "name: demo\ncomponents:\n- name: web\n",
	})
	if res.IsError {
		t.Fatalf("unexpected error result")
	}
	// Structured content should carry the error issue.
	encoded, _ := json.Marshal(res.StructuredContent)
	if !strings.Contains(string(encoded), "no type") {
		t.Errorf("structured content missing structural issue: %s", encoded)
	}
}

func TestUndeployDesignTool(t *testing.T) {
	s := newTestMC(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("delete") != "true" {
			t.Errorf("delete = %q", r.URL.Query().Get("delete"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"d1","name":"demo","removed":[{"kind":"Deployment","name":"web","status":"deleted"}]}`))
	})

	res := callTool(t, s, "undeploy_design", map[string]any{
		"pattern_id": "d1",
		"context_id": "ctx-1",
		"force":      true,
	})
	if res.IsError {
		t.Fatalf("unexpected error result")
	}
	if !strings.Contains(res.Content[0].(mcp.TextContent).Text, "undeployed 1 resource") {
		t.Errorf("fallback text = %q", res.Content[0].(mcp.TextContent).Text)
	}
}

func TestUndeployDesignToolMissingArgs(t *testing.T) {
	s := newTestMC(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be called")
	})

	res := callTool(t, s, "undeploy_design", map[string]any{})
	if !res.IsError {
		t.Fatalf("expected error for missing pattern_id/pattern_file")
	}
}

func TestGetClusterResourcesTool(t *testing.T) {
	s := newTestMC(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/system/meshsync/resources" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"resources":[{"kind":"Pod","name":"web-1","namespace":"default","status":"Running"}]}`))
	})

	res := callTool(t, s, "get_cluster_resources", map[string]any{
		"context_id": "ctx-1",
		"namespace":  "default",
	})
	if res.IsError {
		t.Fatalf("unexpected error result")
	}
	if !strings.Contains(res.Content[0].(mcp.TextContent).Text, "1 resource") {
		t.Errorf("fallback text = %q", res.Content[0].(mcp.TextContent).Text)
	}
}
