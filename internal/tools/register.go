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

// Package tools implements the MCP tool surface of the Meshery MCP demo
// server. Every tool registers through the shared Registrant seam.
package tools

import (
	"github.com/mark3labs/mcp-go/server"

	"github.com/AkashNaickar/meshery-mcp-demo/internal/meshery"
)

// Register registers every tool exposed by the server behind the shared
// Meshery client, the single integration boundary.
func Register(s *server.MCPServer, mc *meshery.Client) error {
	if err := registerServerInfo(s); err != nil {
		return err
	}
	if err := registerListDesigns(s, mc); err != nil {
		return err
	}
	if err := registerListKubernetesContexts(s, mc); err != nil {
		return err
	}
	if err := registerDeployDesign(s, mc); err != nil {
		return err
	}
	return nil
}
