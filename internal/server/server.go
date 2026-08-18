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

package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"github.com/AkashNaickar/meshery-mcp-demo/internal/config"
	"github.com/AkashNaickar/meshery-mcp-demo/internal/meshery"
	"github.com/AkashNaickar/meshery-mcp-demo/internal/prompts"
	"github.com/AkashNaickar/meshery-mcp-demo/internal/resources"
	"github.com/AkashNaickar/meshery-mcp-demo/internal/tools"
	"github.com/AkashNaickar/meshery-mcp-demo/internal/version"
)

// New creates an MCP server with all registered MCP surfaces, backed by the
// shared Meshery client.
func New(mc *meshery.Client) (*server.MCPServer, error) {
	s := server.NewMCPServer(version.Name, version.Version, server.WithHooks(&server.Hooks{}))

	registry := NewRegistry(
		RegistrantFunc(func(s *server.MCPServer) error {
			return tools.Register(s, mc)
		}),
		RegistrantFunc(func(s *server.MCPServer) error {
			return resources.Register(s, mc)
		}),
		RegistrantFunc(func(s *server.MCPServer) error {
			return prompts.Register(s)
		}),
	)

	if err := registry.RegisterAll(s); err != nil {
		return nil, err
	}

	return s, nil
}

// Serve runs the MCP server over the transport selected in cfg until the
// process is interrupted or, for network transports, gracefully shut down.
func Serve(s *server.MCPServer, cfg *config.Config) error {
	switch cfg.Transport {
	case "stdio":
		return server.ServeStdio(s)
	case "http", "sse":
		return serveHTTP(s, cfg)
	default:
		return fmt.Errorf("unsupported transport %q (want stdio, http, or sse)", cfg.Transport)
	}
}

// serveHTTP exposes the MCP server over the streamable HTTP transport (or SSE,
// negotiated per session) and shuts down gracefully on SIGINT/SIGTERM.
func serveHTTP(s *server.MCPServer, cfg *config.Config) error {
	httpServer := server.NewStreamableHTTPServer(s)

	errCh := make(chan error, 1)
	go func() {
		log.Printf("serving MCP over streamable HTTP on %s", cfg.HTTPAddr)
		errCh <- httpServer.Start(cfg.HTTPAddr)
	}()

	ctx, stop := signalContext()
	defer stop()

	select {
	case err := <-errCh:
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	case <-ctx.Done():
		log.Printf("shutting down MCP HTTP server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	}
}
