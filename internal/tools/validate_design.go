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

package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AkashNaickar/meshery-mcp-demo/internal/meshery"
)

// registerValidateDesign registers the validate_design tool, which lints a
// PatternFile without applying it to any cluster.
func registerValidateDesign(s *server.MCPServer, mc *meshery.Client) error {
	tool := mcp.NewTool("validate_design",
		mcp.WithDescription("Lint and validate a Meshery design (PatternFile YAML) against Meshery schemas without applying it to any cluster."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithString("pattern_file",
			mcp.Required(),
			mcp.Description("The design YAML (PatternFile) to validate."),
		),
	)
	s.AddTool(tool, validateDesignHandler(mc))
	return nil
}

// validateDesignHandler validates a design and returns the findings.
func validateDesignHandler(mc *meshery.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		patternFile := req.GetString("pattern_file", "")
		if patternFile == "" {
			return mcp.NewToolResultErrorf("validate_design requires a pattern_file argument"), nil
		}

		result, err := mc.ValidateDesign(ctx, patternFile)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("validate design", err), nil
		}

		fallback := "design is valid"
		if !result.Valid {
			fallback = "design has validation issues"
		}
		return mcp.NewToolResultStructured(result, fallback), nil
	}
}
