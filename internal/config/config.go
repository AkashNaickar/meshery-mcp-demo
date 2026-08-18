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
	"net/url"
	"os"
)

const (
	// DefaultMeshServerURL is the default base URL of the Meshery Server REST API.
	DefaultMeshServerURL = "http://localhost:9081"

	// DefaultTokenPath is the default location of the mesheryctl auth file.
	DefaultTokenPath = "~/.meshery/auth.json"

	// DefaultHTTPAddr is the default listen address for the streamable HTTP transport.
	DefaultHTTPAddr = "127.0.0.1:8080"
)

// Config holds runtime configuration for the Meshery MCP demo server.
type Config struct {
	// MeshServerURL is the base URL of the Meshery Server REST API.
	MeshServerURL string
	// MeshTokenPath is the path to the mesheryctl auth file (token + provider cookies).
	MeshTokenPath string
	// MeshAPIToken is an optional raw token that overrides the auth file.
	MeshAPIToken string
	// Transport selects the MCP transport: stdio, http, or sse.
	Transport string
	// HTTPAddr is the listen address for the http and sse transports.
	HTTPAddr string
}

// Load reads configuration from the environment, applying defaults where unset.
func Load() *Config {
	return &Config{
		MeshServerURL: envOr("MESHERY_SERVER_URL", DefaultMeshServerURL),
		MeshTokenPath: envOr("MESHERY_TOKEN_PATH", DefaultTokenPath),
		MeshAPIToken:  os.Getenv("MESHERY_API_TOKEN"),
		Transport:     envOr("MESHERY_MCP_TRANSPORT", "stdio"),
		HTTPAddr:      envOr("MESHERY_MCP_HTTP_ADDR", DefaultHTTPAddr),
	}
}

// RedactedURL returns the Meshery Server URL with any userinfo, query, and
// fragment components removed, suitable for logging.
func (c *Config) RedactedURL() string {
	if c.MeshServerURL == "" {
		return "<unset>"
	}
	u, err := url.Parse(c.MeshServerURL)
	if err != nil {
		return "<invalid>"
	}
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

// envOr returns the value of the environment variable key, or fallback when
// the variable is unset or empty.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
