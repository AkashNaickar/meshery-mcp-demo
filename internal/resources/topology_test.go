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

package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/AkashNaickar/meshery-mcp-demo/internal/meshery"
)

func TestParseClusterID(t *testing.T) {
	tests := []struct {
		uri     string
		want    string
		wantErr bool
	}{
		{uri: "meshery://clusters/ks-1/topology", want: "ks-1"},
		{uri: "meshery://clusters/a-b_c.d/topology", want: "a-b_c.d"},
		{uri: "meshery://clusters/other", wantErr: true},
		{uri: "meshery://designs/x", wantErr: true},
	}
	for _, tt := range tests {
		got, err := parseClusterID(tt.uri)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseClusterID(%q): expected error", tt.uri)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseClusterID(%q): %v", tt.uri, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseClusterID(%q) = %q, want %q", tt.uri, got, tt.want)
		}
	}
}

func TestTopologyHandler(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/system/meshsync/resources" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"evaluated": true,
			"components": [{"id": "c1", "kind": "Deployment", "name": "emoji"}],
			"relationships": []
		}`))
	}))
	defer srv.Close()

	mc, err := meshery.New(meshery.Options{BaseURL: srv.URL, Token: "test-token"})
	if err != nil {
		t.Fatalf("meshery.New: %v", err)
	}

	handler := topologyHandler(mc)
	contents, err := handler(context.Background(), mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{URI: "meshery://clusters/ks-1/topology"},
	})
	if err != nil {
		t.Fatalf("topologyHandler: %v", err)
	}
	if len(contents) != 1 {
		t.Fatalf("len(contents) = %d, want 1", len(contents))
	}

	text, ok := contents[0].(mcp.TextResourceContents)
	if !ok {
		t.Fatalf("content type = %T, want TextResourceContents", contents[0])
	}
	if text.URI != "meshery://clusters/ks-1/topology" {
		t.Errorf("URI = %q", text.URI)
	}

	var topo struct {
		Components []map[string]any `json:"components"`
		Evaluated  bool             `json:"evaluated"`
	}
	if err := json.Unmarshal([]byte(text.Text), &topo); err != nil {
		t.Fatalf("decode topology text: %v", err)
	}
	if len(topo.Components) != 1 {
		t.Errorf("len(components) = %d, want 1", len(topo.Components))
	}
	if !topo.Evaluated {
		t.Errorf("evaluated = false, want true")
	}
}

func TestTopologyHandlerBadURI(t *testing.T) {
	mc, err := meshery.New(meshery.Options{BaseURL: "http://127.0.0.1:1", Token: "test-token"})
	if err != nil {
		t.Fatalf("meshery.New: %v", err)
	}

	handler := topologyHandler(mc)
	if _, err := handler(context.Background(), mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{URI: "meshery://designs/x"},
	}); err == nil {
		t.Fatal("expected error for unsupported URI")
	}
}
