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
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AkashNaickar/meshery-mcp-demo/internal/meshery"
)

// registerListDesigns registers the list_designs tool.
func registerListDesigns(s *server.MCPServer, mc *meshery.Client) error {
	tool := mcp.NewTool("list_designs",
		mcp.WithDescription("List Meshery designs stored on the connected Meshery Server."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithString("search", mcp.Description("Optional free-text filter applied by the server.")),
		mcp.WithNumber("page", mcp.Description("Page number to fetch (1-based).")),
		mcp.WithNumber("page_size", mcp.Description("Number of designs per page.")),
	)
	s.AddTool(tool, listDesignsHandler(mc))
	return nil
}

// listDesignsHandler returns the designs stored on the Meshery Server.
func listDesignsHandler(mc *meshery.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		page := int(req.GetFloat("page", 1))
		pageSize := int(req.GetFloat("page_size", 25))

		list, err := mc.SearchDesigns(ctx, meshery.DesignSearchOptions{
			Search:   req.GetString("search", ""),
			Page:     page,
			PageSize: pageSize,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("list designs", err), nil
		}

		type designView struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			UpdatedAt string `json:"updated_at"`
		}

		designs := make([]designView, 0, len(list.Patterns))
		for _, d := range list.Patterns {
			designs = append(designs, designView{
				ID:        d.ID,
				Name:      d.Name,
				UpdatedAt: d.UpdatedAt.Format(time.RFC3339),
			})
		}

		fallback := fmt.Sprintf("%d designs found (page %d, page size %d)", len(designs), page, pageSize)
		return mcp.NewToolResultStructured(map[string]any{
			"designs":   designs,
			"count":     len(designs),
			"page":      page,
			"page_size": pageSize,
		}, fallback), nil
	}
}
