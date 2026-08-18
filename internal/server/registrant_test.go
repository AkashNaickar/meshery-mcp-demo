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
	"errors"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/server"
)

func TestRegisterAllRunsEverySurface(t *testing.T) {
	var calls []string
	registry := NewRegistry(
		Named("tools", RegistrantFunc(func(*server.MCPServer) error {
			calls = append(calls, "tools")
			return nil
		})),
		Named("resources", RegistrantFunc(func(*server.MCPServer) error {
			calls = append(calls, "resources")
			return nil
		})),
		Named("prompts", RegistrantFunc(func(*server.MCPServer) error {
			calls = append(calls, "prompts")
			return nil
		})),
	)

	s := server.NewMCPServer("test", "0.0.0")
	if err := registry.RegisterAll(s); err != nil {
		t.Fatalf("RegisterAll returned unexpected error: %v", err)
	}

	if len(calls) != 3 {
		t.Fatalf("expected all 3 registrants to run, got %v", calls)
	}
}

func TestRegisterAllStopsAtFirstErrorAndNamesSurface(t *testing.T) {
	original := errors.New("boom")
	var calls []string
	registry := NewRegistry(
		Named("tools", RegistrantFunc(func(*server.MCPServer) error {
			calls = append(calls, "tools")
			return nil
		})),
		Named("resources", RegistrantFunc(func(*server.MCPServer) error {
			calls = append(calls, "resources")
			return original
		})),
		Named("prompts", RegistrantFunc(func(*server.MCPServer) error {
			calls = append(calls, "prompts")
			return nil
		})),
	)

	s := server.NewMCPServer("test", "0.0.0")
	err := registry.RegisterAll(s)
	if err == nil {
		t.Fatal("expected RegisterAll to return an error")
	}

	if !errors.Is(err, original) {
		t.Fatalf("expected original error to be wrapped for inspection, got %v", err)
	}
	if got := err.Error(); !strings.Contains(got, "resources") {
		t.Fatalf("expected error to name the failing surface, got %q", got)
	}
	if len(calls) != 2 {
		t.Fatalf("expected fail-fast after 2 registrants, got %v", calls)
	}
}

func TestRegisterAllNamesUnlabeledRegistrantByType(t *testing.T) {
	registry := NewRegistry(RegistrantFunc(func(*server.MCPServer) error {
		return errors.New("boom")
	}))

	s := server.NewMCPServer("test", "0.0.0")
	err := registry.RegisterAll(s)
	if err == nil {
		t.Fatal("expected RegisterAll to return an error")
	}

	if got := err.Error(); !strings.Contains(got, "RegistrantFunc") {
		t.Fatalf("expected error to name the registrant type, got %q", got)
	}
}
