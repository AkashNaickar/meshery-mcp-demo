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
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AkashNaickar/meshery-mcp-demo/internal/meshery"
)

// registerUndeployDesign registers the undeploy_design tool, which tears down
// resources defined by a design. It is the destructive counterpart to
// deploy_design and carries the destructiveHint annotation.
func registerUndeployDesign(s *server.MCPServer, mc *meshery.Client) error {
	tool := mcp.NewTool("undeploy_design",
		mcp.WithDescription("Tear down the resources defined by a Meshery design from a Kubernetes context. Use force to skip confirmation guards."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithString("pattern_id",
			mcp.Description("ID of the stored design to undeploy. Takes precedence over pattern_file."),
		),
		mcp.WithString("pattern_file",
			mcp.Description("Inline design YAML to undeploy, used when pattern_id is absent."),
		),
		mcp.WithString("context_id",
			mcp.Description("The ID of the Kubernetes context to undeploy from. Empty uses the server's current context."),
		),
		mcp.WithBoolean("force",
			mcp.Description("Ignore confirmation guards and undeploy immediately."),
			mcp.DefaultBool(false),
		),
	)
	s.AddTool(tool, undeployDesignHandler(mc))
	return nil
}

// undeployDesignHandler tears down a deployed design.
func undeployDesignHandler(mc *meshery.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		patternID := req.GetString("pattern_id", "")
		patternFile := req.GetString("pattern_file", "")
		if patternID == "" && patternFile == "" {
			return mcp.NewToolResultErrorf("undeploy_design requires a pattern_id or pattern_file argument"), nil
		}

		var contexts []string
		if id := req.GetString("context_id", ""); id != "" {
			contexts = []string{id}
		}

		resp, err := mc.UndeployDesign(ctx, meshery.UndeployRequest{
			PatternID:   patternID,
			PatternFile: patternFile,
			Contexts:    contexts,
			Force:       req.GetBool("force", false),
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("undeploy design", err), nil
		}
		if resp.Error != "" {
			return mcp.NewToolResultErrorf("undeploy design: %s", resp.Error), nil
		}

		result := map[string]any{
			"status":      "undeployed",
			"id":          resp.ID,
			"name":        resp.Name,
			"messages":    resp.Messages,
			"removed":     resp.Removed,
			"not_removed": resp.NotRemoved,
		}

		fallback := fmt.Sprintf("undeployed %d resource(s): %s", len(resp.Removed), joinStatuses(resp.Removed))
		return mcp.NewToolResultStructured(result, fallback), nil
	}
}
