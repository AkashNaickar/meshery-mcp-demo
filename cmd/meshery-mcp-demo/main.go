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
	"log"
	"os"

	"github.com/AkashNaickar/meshery-mcp-demo/internal/config"
	"github.com/AkashNaickar/meshery-mcp-demo/internal/meshery"
	"github.com/AkashNaickar/meshery-mcp-demo/internal/server"
	"github.com/AkashNaickar/meshery-mcp-demo/internal/version"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	log.SetOutput(os.Stderr)

	cfg := config.Load()
	log.Printf("starting %s %s (commit %s, Meshery Server: %s, transport: %s)", version.Name, version.Version, version.CommitSHA, cfg.RedactedURL(), cfg.Transport)

	mc, err := meshery.New(meshery.Options{
		BaseURL:   cfg.MeshServerURL,
		TokenPath: cfg.MeshTokenPath,
		Token:     cfg.MeshAPIToken,
	})
	if err != nil {
		log.Fatalf("create Meshery client: %v", err)
	}

	srv, err := server.New(mc)
	if err != nil {
		log.Fatalf("create MCP server: %v", err)
	}

	if err := server.Serve(srv, cfg); err != nil {
		log.Fatalf("serve MCP server: %v", err)
	}
}
