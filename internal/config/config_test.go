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

package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("MESHERY_SERVER_URL", "")
	t.Setenv("MESHERY_MCP_TRANSPORT", "")
	t.Setenv("MESHERY_MCP_HTTP_ADDR", "")
	t.Setenv("MESHERY_TOKEN_PATH", "")

	cfg := Load()

	if cfg.MeshServerURL != DefaultMeshServerURL {
		t.Errorf("MeshServerURL = %q, want %q", cfg.MeshServerURL, DefaultMeshServerURL)
	}
	if cfg.Transport != "stdio" {
		t.Errorf("Transport = %q, want stdio", cfg.Transport)
	}
	if cfg.HTTPAddr != DefaultHTTPAddr {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, DefaultHTTPAddr)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	t.Setenv("MESHERY_SERVER_URL", "https://meshery.example.com")
	t.Setenv("MESHERY_API_TOKEN", "s3cret")
	t.Setenv("MESHERY_MCP_TRANSPORT", "http")
	t.Setenv("MESHERY_MCP_HTTP_ADDR", "127.0.0.1:9999")

	cfg := Load()

	if cfg.MeshServerURL != "https://meshery.example.com" {
		t.Errorf("MeshServerURL = %q", cfg.MeshServerURL)
	}
	if cfg.MeshAPIToken != "s3cret" {
		t.Errorf("MeshAPIToken not read from env")
	}
	if cfg.Transport != "http" {
		t.Errorf("Transport = %q", cfg.Transport)
	}
	if cfg.HTTPAddr != "127.0.0.1:9999" {
		t.Errorf("HTTPAddr = %q", cfg.HTTPAddr)
	}
}

func TestRedactedURL(t *testing.T) {
	cfg := &Config{MeshServerURL: "https://user:pass@host:9081/api?x=1#frag"}
	got := cfg.RedactedURL()
	if got != "https://host:9081/api" {
		t.Errorf("RedactedURL() = %q, want https://host:9081/api", got)
	}

	empty := (&Config{}).RedactedURL()
	if empty != "<unset>" {
		t.Errorf("empty RedactedURL() = %q, want <unset>", empty)
	}
}
