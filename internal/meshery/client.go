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

// Package meshery provides a small authenticated REST client for the
// Meshery Server API. It is the single integration boundary between the MCP
// surfaces and Meshery, per the architecture in the design document.
package meshery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Client talks to a Meshery Server REST API.
type Client struct {
	baseURL    string
	token      string
	provider   string
	httpClient *http.Client
}

// Options configures a Client.
type Options struct {
	// BaseURL is the Meshery Server base URL, e.g. http://localhost:9081.
	BaseURL string
	// TokenPath is the path to the mesheryctl auth file (~/.meshery/auth.json).
	TokenPath string
	// Token, when set, overrides the token read from TokenPath.
	Token string
}

// authFile mirrors the JSON written by `mesheryctl system login`.
type authFile struct {
	Token    string `json:"token"`
	Provider string `json:"meshery-provider"`
}

// New creates a Client, loading credentials from the auth file unless an
// explicit token is provided.
func New(opts Options) (*Client, error) {
	c := &Client{
		baseURL: strings.TrimSuffix(opts.BaseURL, "/"),
		token:   opts.Token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	if c.token == "" {
		if err := c.loadAuthFile(opts.TokenPath); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// loadAuthFile reads the token and provider cookie written by mesheryctl.
func (c *Client) loadAuthFile(path string) error {
	expanded, err := expandPath(path)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(expanded)
	if err != nil {
		return fmt.Errorf("read Meshery auth file %s: %w (run mesheryctl system login)", expanded, err)
	}

	var auth authFile
	if err := json.Unmarshal(data, &auth); err != nil {
		return fmt.Errorf("parse Meshery auth file %s: %w", expanded, err)
	}

	if auth.Token == "" {
		return fmt.Errorf("auth file %s contains no token; re-run mesheryctl system login", expanded)
	}

	c.token = auth.Token
	c.provider = auth.Provider
	if c.provider == "" {
		c.provider = "Meshery"
	}
	return nil
}

// expandPath resolves ~ to the user home directory, accepting both Unix and
// Windows separators in the remainder of the path.
func expandPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		rest := strings.TrimPrefix(path, "~")
		rest = strings.TrimPrefix(rest, "/")
		rest = strings.TrimPrefix(rest, `\`)
		path = filepath.Join(home, rest)
	}
	return filepath.Clean(path), nil
}

// request performs an authenticated HTTP request and decodes the response.
func (c *Client) request(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("parse URL: %w", err)
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), reader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	// Meshery accepts the provider token both as a header and a cookie; send
	// both to match how the Meshery UI authenticates.
	req.Header.Set("meshery-token", c.token)
	req.Header.Set("Cookie", "meshery-provider="+c.provider+"; token="+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request %s %s: %w", method, u.Path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("meshery API %s %s: %s: %s", method, u.Path, resp.Status, strings.TrimSpace(string(msg)))
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response from %s: %w", u.Path, err)
		}
	}
	return nil
}
