# Meshery MCP Demo

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![build-and-test](https://github.com/AkashNaickar/meshery-mcp-demo/actions/workflows/build-and-test.yml/badge.svg)](https://github.com/AkashNaickar/meshery-mcp-demo/actions/workflows/build-and-test.yml)

A production-grade proof-of-concept of a [Model Context Protocol](https://modelcontextprotocol.io) server for [Meshery](https://meshery.io), built in Go. It lets AI agents observe and operate Meshery across the full design lifecycle: validate, deploy, verify, and tear down designs against live Kubernetes clusters through natural tools, resources, and prompts.

> This POC demonstrates the architecture proposed for the [Meshery MCP Server](https://github.com/meshery-extensions/meshery-mcp-server) project: every MCP surface registers through one shared seam, backed by a single authenticated, sanitizing Meshery client.

## Demo

_Demo video: driving the server from MCP Inspector, validating a design, dry-run deploying, watching the topology resource update live, and verifying live workloads._

## Architecture

<img width="1323" height="1403" alt="mcp-server-design" src="https://github.com/user-attachments/assets/55bf08a4-7ce6-4c05-942a-41c897061a36" />


## What it demonstrates

### Tools

| Tool | Hints | What it does |
|---|---|---|
| `server_info` | readOnly | Metadata about the MCP server (version, runtime, Meshery endpoint) |
| `list_kubernetes_contexts` | readOnly | List Kubernetes contexts Meshery manages (`GET /api/system/kubernetes/contexts`) |
| `list_designs` | readOnly | Paginated designs with optional `search`, `page`, `page_size` (`GET /api/pattern`) |
| `validate_design` | readOnly | Lint a PatternFile YAML structurally + server-side without applying |
| `deploy_design` | mutating | Deploy a PatternFile to a context; `dry_run` returns a preview without mutation |
| `undeploy_design` | destructive | Tear down a design's resources; `force` skips confirmation guards |
| `get_cluster_resources` | readOnly | Inspect live pods/services/workloads for post-deploy verification |

### Resources

| Resource | What it does |
|---|---|
| `meshery://clusters/{cluster_id}/topology` | Live MeshSync topology graph; falls back to the direct K8s API when MeshSync has not synced |
| `meshery://designs/{design_id}` | Raw PatternFile exposed directly into the agent's context without a tool turn |

### Prompts

| Prompt | What it does |
|---|---|
| `deploy_application` | Guided workflow: list contexts, find design, validate, dry-run, confirm, deploy, verify health |
| `cluster_health_audit` | Reads topology + live resources, surfaces drift and optimization recommendations |

The registration seam (`internal/server/registrant.go`) is the architectural centerpiece: tools, resources, and prompts all implement one `Registrant` interface, so adding a surface is one file plus one line in the registry.

## Security

All tool outputs and resource payloads pass through a redaction engine (`internal/meshery/sanitize.go`) before reaching the agent. It strips:

- Kubernetes `Secret` `data` payloads
- Bearer/JWT tokens and AWS access keys
- Private keys and SSH material
- Fields named `token`, `password`, `api_key`, `authorization`, `credentials`, and similar

Redacted values are replaced with a fixed `[REDACTED]` marker so diffs and fingerprints stay stable.

## Error mapping

Meshery failures are mapped to JSON-RPC error codes (`internal/meshery/errors.go`):

| Code | Meaning |
|---|---|
| `-32001` | Meshery Daemon Unreachable (connection / 5xx) |
| `-32002` | Auth Failure (401 / 403) |
| `-32602` | Invalid Parameter Schemas (client 4xx) |
| `-32603` | Unknown error fallback |

## Quickstart

### Prerequisites

- Go 1.26+
- A running Meshery Server (see [meshery.io/docs](https://docs.meshery.io)) with a connected Kubernetes context
- `mesheryctl` authenticated: `mesheryctl system login` writes `~/.meshery/auth.json`
- A kubeconfig context (default `kind-meshery-demo`) for the topology fallback

### Run

```bash
go build -o bin/meshery-mcp-demo ./cmd/meshery-mcp-demo

# stdio transport (default, for Claude Desktop / Cursor / OpenCode)
./bin/meshery-mcp-demo

# streamable HTTP / SSE transport on a custom port
./bin/meshery-mcp-demo --transport sse --port 8080

# or via MCP Inspector
npx @modelcontextprotocol/inspector ./bin/meshery-mcp-demo
```

### Configuration

| Environment variable | Default | Description |
|---|---|---|
| `MESHERY_SERVER_URL` | `http://localhost:9081` | Meshery Server base URL |
| `MESHERY_TOKEN_PATH` | `~/.meshery/auth.json` | Path to the mesheryctl auth file |
| `MESHERY_TOKEN` | unset | Raw token that overrides the auth file |
| `MESHERY_API_TOKEN` | unset | Alias for `MESHERY_TOKEN` |
| `MESHERY_MCP_TRANSPORT` | `stdio` | `stdio`, `http`, or `sse` |
| `MESHERY_MCP_HTTP_ADDR` | `127.0.0.1:8080` | Listen address for http/sse |
| `MESHERY_KUBECONFIG` | `~/.kube/config` | Kubeconfig for the topology fallback |
| `MESHERY_KUBECONFIG_CONTEXT` | `kind-meshery-demo` | Context for the topology fallback |

## MCP client configuration

### Claude Desktop

`claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "meshery": {
      "command": "D:\\Projekt\\meshery-mcp-demo\\bin\\meshery-mcp-demo.exe"
    }
  }
}
```

### Cursor

`.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "meshery": {
      "command": "D:\\Projekt\\meshery-mcp-demo\\bin\\meshery-mcp-demo.exe"
    }
  }
}
```

### OpenCode

`opencode.json` (or `~/.config/opencode/opencode.jsonc`):

```jsonc
{
  "mcp": {
    "meshery": {
      "type": "local",
      "command": ["D:\\Projekt\\meshery-mcp-demo\\bin\\meshery-mcp-demo.exe"],
      "enabled": true
    }
  }
}
```

## End-to-end verification walkthrough

1. **Inspect clusters**
   `list_kubernetes_contexts` -> returns `kind-meshery-demo` and peers.

2. **Find a design**
   `list_designs` with `{ "search": "nginx", "page": 1, "page_size": 10 }`.

3. **Validate before deploying**
   `validate_design` with `{ "pattern_file": "<design YAML>" }` -> confirm `valid: true`.

4. **Dry-run preview**
   `deploy_design` with `{ "pattern_file": "<design YAML>", "context_id": "<ctx>", "dry_run": true }` -> returns the planned resources without mutating anything.

5. **Deploy**
   `deploy_design` with `{ "pattern_file": "<design YAML>", "context_id": "<ctx>", "dry_run": false }`.

6. **Verify live workloads**
   `get_cluster_resources` with `{ "context_id": "<ctx>", "namespace": "default" }`, and read `meshery://clusters/{cluster_id}/topology`.

7. **Tear down**
   `undeploy_design` with `{ "pattern_id": "<design_id>", "context_id": "<ctx>", "force": true }`.

## Development

```bash
make test    # go test ./... -race
make lint    # golangci-lint run
make build   # build to bin/
```

CI runs lint, vet, race-enabled tests, and cross-platform builds (linux/windows/darwin x amd64/arm64). The topology fallback and notification subscription are covered by unit tests; the live-push can be verified by subscribing to a topology resource and scaling a workload.

## License

Apache License 2.0. See [LICENSE](LICENSE).
