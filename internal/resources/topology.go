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

// Package resources implements the MCP resource surface of the Meshery MCP
// demo server. Resources are read-only views over live Meshery state.
package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AkashNaickar/meshery-mcp-demo/internal/meshery"
)

// topologyURITemplate is the resource template exposing a cluster's live
// MeshSync-discovered topology as a graph.
const topologyURITemplate = "meshery://clusters/{cluster_id}/topology"

// clusterIDPattern extracts the cluster_id variable from a concrete resource URI.
var clusterIDPattern = regexp.MustCompile(`^meshery://clusters/([^/]+)/topology$`)

// subscriptionTracker records which resource URIs clients are subscribed to so
// the poller knows what to watch.
type subscriptionTracker struct {
	mu   sync.Mutex
	uris map[string]struct{}
}

func newSubscriptionTracker() *subscriptionTracker {
	return &subscriptionTracker{uris: make(map[string]struct{})}
}

func (t *subscriptionTracker) subscribe(uri string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.uris[uri] = struct{}{}
}

func (t *subscriptionTracker) unsubscribe(uri string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.uris, uri)
}

func (t *subscriptionTracker) snapshot() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]string, 0, len(t.uris))
	for uri := range t.uris {
		out = append(out, uri)
	}
	return out
}

// Register registers the topology resource template and starts the
// poll-and-notify subscription loop.
func Register(s *server.MCPServer, mc *meshery.Client) error {
	tracker := newSubscriptionTracker()

	template := mcp.NewResourceTemplate(topologyURITemplate, "Cluster Topology",
		mcp.WithTemplateTitle("Live cluster topology"),
		mcp.WithTemplateDescription("The MeshSync-discovered state of a cluster, rendered as a design graph."),
		mcp.WithTemplateMIMEType("application/json"),
	)
	s.AddResourceTemplate(template, topologyHandler(mc))

	// Track resources/subscribe and resources/unsubscribe so the poller only
	// watches URIs clients actually care about.
	s.GetHooks().AddBeforeAny(func(_ context.Context, _ any, method mcp.MCPMethod, message any) {
		switch method {
		case mcp.MethodResourcesSubscribe:
			if req, ok := message.(mcp.SubscribeRequest); ok && req.Params.URI != "" {
				tracker.subscribe(req.Params.URI)
			}
		case mcp.MethodResourcesUnsubscribe:
			if req, ok := message.(mcp.UnsubscribeRequest); ok && req.Params.URI != "" {
				tracker.unsubscribe(req.Params.URI)
			}
		}
	})

	go pollTopologyUpdates(s, mc, tracker)
	return nil
}

// topologyHandler reads a cluster's live topology graph.
func topologyHandler(mc *meshery.Client) server.ResourceTemplateHandlerFunc {
	return func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		clusterID, err := parseClusterID(req.Params.URI)
		if err != nil {
			return nil, err
		}

		topo, err := mc.Topology(ctx, clusterID)
		if err != nil {
			return nil, fmt.Errorf("read topology for cluster %s: %w", clusterID, err)
		}

		payload, err := json.MarshalIndent(topo, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("encode topology: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(payload),
			},
		}, nil
	}
}

// parseClusterID extracts the cluster_id variable from a concrete URI.
func parseClusterID(uri string) (string, error) {
	m := clusterIDPattern.FindStringSubmatch(uri)
	if m == nil {
		return "", fmt.Errorf("unsupported resource URI %q", uri)
	}
	return m[1], nil
}

// pollTopologyUpdates re-reads subscribed topologies and pushes a
// notifications/resources/updated message when their content changes. This is
// the poll-and-notify pattern: MeshSync exposes no push channel, so the server
// polls and notifies.
func pollTopologyUpdates(s *server.MCPServer, mc *meshery.Client, tracker *subscriptionTracker) {
	interval := 10 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fingerprints := make(map[string]string)

	for range ticker.C {
		for _, uri := range tracker.snapshot() {
			clusterID, err := parseClusterID(uri)
			if err != nil {
				continue
			}

			topo, err := mc.Topology(context.Background(), clusterID)
			if err != nil {
				// Transient errors (cluster temporarily unreachable) are
				// expected during demo; keep the fingerprint so a recovery
				// still triggers a notification.
				log.Printf("topology poll %s: %v", uri, err)
				continue
			}

			encoded, _ := json.Marshal(topo)
			fp := string(encoded)
			if prev, ok := fingerprints[uri]; !ok {
				fingerprints[uri] = fp
				continue
			} else if prev == fp {
				continue
			}
			fingerprints[uri] = fp

			log.Printf("topology changed for %s, notifying subscribers", uri)
			s.SendNotificationToAllClients(mcp.MethodNotificationResourceUpdated, map[string]any{
				"uri": uri,
			})
		}
	}
}
