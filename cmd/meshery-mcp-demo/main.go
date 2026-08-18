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

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/AkashNaickar/meshery-mcp-demo/internal/config"
	"github.com/AkashNaickar/meshery-mcp-demo/internal/k8s"
	"github.com/AkashNaickar/meshery-mcp-demo/internal/meshery"
	"github.com/AkashNaickar/meshery-mcp-demo/internal/server"
	"github.com/AkashNaickar/meshery-mcp-demo/internal/version"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	log.SetOutput(os.Stderr)

	transport := flag.String("transport", "", "MCP transport: stdio (default) or sse")
	port := flag.Int("port", 0, "Port for the sse/http transport (default 8080)")
	flag.Parse()

	cfg := config.Load()
	if *transport != "" {
		cfg.Transport = *transport
	}
	if *port != 0 {
		cfg.HTTPAddr = fmt.Sprintf("127.0.0.1:%d", *port)
	}

	log.Printf("starting %s %s (commit %s, Meshery Server: %s, transport: %s)",
		version.Name, version.Version, version.CommitSHA, cfg.RedactedURL(), cfg.Transport)

	mc, err := meshery.New(meshery.Options{
		BaseURL:   cfg.MeshServerURL,
		TokenPath: cfg.MeshTokenPath,
		Token:     cfg.MeshAPIToken,
	})
	if err != nil {
		log.Fatalf("create Meshery client: %v", err)
	}

	// Wire the k8s topology fallback so the topology resource reflects live
	// cluster state even when MeshSync has not synced the server.
	kc, err := k8s.New(k8s.Options{
		KubeconfigPath: expandTilde(cfg.KubeconfigPath),
		Context:        cfg.KubeconfigContext,
	})
	if err != nil {
		log.Printf("topology fallback disabled: %v", err)
	} else {
		mc.SetTopologySource(kc)
		log.Printf("topology fallback enabled (kubeconfig context %s)", cfg.KubeconfigContext)
	}

	srv, err := server.New(mc)
	if err != nil {
		log.Fatalf("create MCP server: %v", err)
	}

	if err := server.Serve(srv, cfg); err != nil {
		log.Fatalf("serve MCP server: %v", err)
	}
}

// expandTilde resolves a leading ~ to the user home directory. It is a small
// local copy so cmd does not reach into the meshery package's unexported
// expandPath.
func expandTilde(path string) string {
	if path == "" || !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	rest := strings.TrimPrefix(path, "~")
	rest = strings.TrimPrefix(rest, "/")
	rest = strings.TrimPrefix(rest, `\`)
	rest = strings.ReplaceAll(rest, `\`, string(filepath.Separator))
	rest = strings.ReplaceAll(rest, "/", string(filepath.Separator))
	return filepath.Join(home, rest)
}
