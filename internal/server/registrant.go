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

import "github.com/mark3labs/mcp-go/server"

// Registrant registers one MCP surface, such as tools, resources, or prompts.
//
// This is the shared contract every surface area implements: a new tool,
// resource, or prompt is one file plus one line in the registry.
type Registrant interface {
	Register(*server.MCPServer) error
}

// RegistrantFunc adapts a registration function to the Registrant interface.
type RegistrantFunc func(*server.MCPServer) error

// Register invokes f with s.
func (f RegistrantFunc) Register(s *server.MCPServer) error {
	return f(s)
}

// Registry registers each configured MCP surface in order.
type Registry struct {
	registrants []Registrant
}

// NewRegistry creates a Registry from registrants.
func NewRegistry(registrants ...Registrant) *Registry {
	return &Registry{registrants: registrants}
}

// Add appends a registrant to the registry.
func (r *Registry) Add(registrant Registrant) {
	r.registrants = append(r.registrants, registrant)
}

// RegisterAll registers every configured surface in order and stops at the
// first registration error.
func (r *Registry) RegisterAll(s *server.MCPServer) error {
	for _, registrant := range r.registrants {
		if err := registrant.Register(s); err != nil {
			return err
		}
	}
	return nil
}
