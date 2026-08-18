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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// newTestServer spins up an httptest server that records requests and serves
// canned responses from a handler map keyed by path.
func newTestServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(srv.Close)
	return srv
}

// newTestClient builds a client against srv with an explicit token so no auth
// file is needed.
func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	c, err := New(Options{BaseURL: srv.URL, Token: "test-token"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestListDesigns(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/pattern" {
			t.Errorf("path = %q, want /api/pattern", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if got := r.Header.Get("meshery-token"); got != "test-token" {
			t.Errorf("meshery-token header = %q", got)
		}
		if got := r.Header.Get("Cookie"); got == "" {
			t.Errorf("Cookie header not set")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"page": 1, "page_size": 25, "count": 2,
			"patterns": [
				{"id": "a1", "name": "emojivoto", "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-02T00:00:00Z"},
				{"id": "b2", "name": "bookinfo", "created_at": "2026-01-03T00:00:00Z", "updated_at": "2026-01-04T00:00:00Z"}
			]
		}`))
	})

	c := newTestClient(t, srv)
	list, err := c.ListDesigns(context.Background(), 1, 25)
	if err != nil {
		t.Fatalf("ListDesigns: %v", err)
	}
	if list.Count != 2 {
		t.Errorf("Count = %d, want 2", list.Count)
	}
	if len(list.Patterns) != 2 {
		t.Errorf("len(Patterns) = %d, want 2", len(list.Patterns))
	}
	if list.Patterns[0].Name != "emojivoto" {
		t.Errorf("first pattern name = %q", list.Patterns[0].Name)
	}
}

func TestListKubernetesContexts(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/system/kubernetes/contexts" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"total_count": 1,
			"contexts": [{
				"id": "ctx-1", "name": "docker-desktop", "server": "https://127.0.0.1:6443",
				"kubernetesServerId": "ks-1", "connectionId": "conn-1", "deployment_type": "docker"
			}]
		}`))
	})

	c := newTestClient(t, srv)
	list, err := c.ListKubernetesContexts(context.Background())
	if err != nil {
		t.Fatalf("ListKubernetesContexts: %v", err)
	}
	if list.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1", list.TotalCount)
	}
	if list.Contexts[0].Name != "docker-desktop" {
		t.Errorf("context name = %q", list.Contexts[0].Name)
	}
}

func TestDeployDesign(t *testing.T) {
	var gotBody map[string]any
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/pattern/deploy" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Query().Get("dryRun") != "false" {
			t.Errorf("dryRun = %q, want false", r.URL.Query().Get("dryRun"))
		}
		if got := r.URL.Query()["contexts"]; len(got) != 1 || got[0] != "ctx-1" {
			t.Errorf("contexts query = %v, want [ctx-1]", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "dep-1", "name": "emojivoto", "dryRun": false,
			"deployed": [{"kind": "Deployment", "name": "emoji", "status": "applied"}]
		}`))
	})

	c := newTestClient(t, srv)
	resp, err := c.DeployDesign(context.Background(), DeployRequest{
		PatternFile: "version: 1.0\nservices: []",
		Contexts:    []string{"ctx-1"},
	})
	if err != nil {
		t.Fatalf("DeployDesign: %v", err)
	}
	if resp.ID != "dep-1" {
		t.Errorf("ID = %q", resp.ID)
	}
	if len(resp.Deployed) != 1 {
		t.Errorf("len(Deployed) = %d, want 1", len(resp.Deployed))
	}

	if gotBody["pattern_file"] != "version: 1.0\nservices: []" {
		t.Errorf("pattern_file body = %v", gotBody["pattern_file"])
	}
	if _, ok := gotBody["contexts"]; ok {
		t.Errorf("contexts must be a query param, not body; got body contexts = %v", gotBody["contexts"])
	}
}

func TestDeployDesignDefaultsToAllContexts(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("contexts"); got != "all" {
			t.Errorf("contexts query = %q, want all", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"d1","name":"x","dryRun":false}`))
	})

	c := newTestClient(t, srv)
	if _, err := c.DeployDesign(context.Background(), DeployRequest{PatternFile: "version: 1.0"}); err != nil {
		t.Fatalf("DeployDesign: %v", err)
	}
}

func TestTopology(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/system/meshsync/resources" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("asDesign") != "true" {
			t.Errorf("asDesign = %q, want true", r.URL.Query().Get("asDesign"))
		}
		if r.URL.Query().Get("clusterId") != "ks-1" {
			t.Errorf("clusterId = %q, want ks-1", r.URL.Query().Get("clusterId"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"evaluated": true,
			"components": [
				{"id": "c1", "kind": "Deployment", "name": "emoji"},
				{"id": "c2", "kind": "Service", "name": "web"}
			],
			"relationships": [{"kind": "network"}]
		}`))
	})

	c := newTestClient(t, srv)
	topo, err := c.Topology(context.Background(), "ks-1")
	if err != nil {
		t.Fatalf("Topology: %v", err)
	}
	if len(topo.Components) != 2 {
		t.Errorf("len(Components) = %d, want 2", len(topo.Components))
	}
	if !topo.Evaluated {
		t.Errorf("Evaluated = false, want true")
	}
	if len(topo.Relationships) != 1 {
		t.Errorf("len(Relationships) = %d, want 1", len(topo.Relationships))
	}
}

func TestClientErrors(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"boom"}`, http.StatusBadRequest)
	})

	c := newTestClient(t, srv)
	_, err := c.ListDesigns(context.Background(), 1, 25)
	if err == nil {
		t.Fatal("ListDesigns: expected error for 400 response")
	}
}

func TestAuthFileLoading(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(path, []byte(`{"token": "file-token", "meshery-provider": "Meshery"}`), 0o600); err != nil {
		t.Fatalf("write auth file: %v", err)
	}

	c, err := New(Options{BaseURL: "http://localhost:9081", TokenPath: path})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.token != "file-token" {
		t.Errorf("token = %q, want file-token", c.token)
	}
	if c.provider != "Meshery" {
		t.Errorf("provider = %q, want Meshery", c.provider)
	}
}

func TestAuthFileMissing(t *testing.T) {
	_, err := New(Options{BaseURL: "http://localhost:9081", TokenPath: filepath.Join(t.TempDir(), "missing.json")})
	if err == nil {
		t.Fatal("New: expected error for missing auth file")
	}
}

func TestExpandPathTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}

	// Unix-style separator in the default path.
	got, err := expandPath("~/.meshery/auth.json")
	if err != nil {
		t.Fatalf("expandPath: %v", err)
	}
	want := filepath.Join(home, ".meshery", "auth.json")
	if got != want {
		t.Errorf("expandPath(~/.meshery/auth.json) = %q, want %q", got, want)
	}

	// Windows-style separator.
	got2, err := expandPath(`~\.meshery\auth.json`)
	if err != nil {
		t.Fatalf("expandPath: %v", err)
	}
	if got2 != want {
		t.Errorf("expandPath(~\\\\.meshery\\\\auth.json) = %q, want %q", got2, want)
	}

	// Absolute path passes through.
	abs := filepath.Join(home, "x.json")
	got3, err := expandPath(abs)
	if err != nil {
		t.Fatalf("expandPath: %v", err)
	}
	if got3 != abs {
		t.Errorf("expandPath(abs) = %q, want %q", got3, abs)
	}
}
