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

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AkashNaickar/meshery-mcp-demo/internal/meshery"
)

// designURITemplate exposes a stored design's raw PatternFile directly into the
// agent's context window without a tool execution turn.
const designURITemplate = "meshery://designs/{design_id}"

// designIDPattern extracts the design_id variable from a concrete resource URI.
var designIDPattern = regexp.MustCompile(`^meshery://designs/([^/]+)$`)

// RegisterDesigns registers the design resource template.
func RegisterDesigns(s *server.MCPServer, mc *meshery.Client) error {
	template := mcp.NewResourceTemplate(designURITemplate, "Design PatternFile",
		mcp.WithTemplateTitle("Meshery design"),
		mcp.WithTemplateDescription("The raw PatternFile (design) stored on the Meshery Server, exposed directly into the agent context."),
		mcp.WithTemplateMIMEType("application/yaml"),
	)
	s.AddResourceTemplate(template, designHandler(mc))
	return nil
}

// designHandler returns a stored design's raw PatternFile, sanitized.
func designHandler(mc *meshery.Client) server.ResourceTemplateHandlerFunc {
	return func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		designID, err := parseDesignID(req.Params.URI)
		if err != nil {
			return nil, err
		}

		detail, err := mc.GetDesign(ctx, designID)
		if err != nil {
			return nil, fmt.Errorf("read design %s: %w", designID, err)
		}

		// Render as a JSON document so Secret data / credentials within the
		// design payload can be redacted uniformly before exposing it.
		doc, err := json.MarshalIndent(detail, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("encode design: %w", err)
		}
		doc = mc.SanitizeJSON(doc)

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(doc),
			},
		}, nil
	}
}

// parseDesignID extracts the design_id variable from a concrete URI.
func parseDesignID(uri string) (string, error) {
	m := designIDPattern.FindStringSubmatch(uri)
	if m == nil {
		return "", fmt.Errorf("unsupported resource URI %q", uri)
	}
	return m[1], nil
}
