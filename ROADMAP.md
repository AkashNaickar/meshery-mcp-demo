# Roadmap

A 12-week plan for turning this POC into the full Meshery MCP Server, mapped to the funded project issues in [meshery-extensions/meshery-mcp-server](https://github.com/meshery-extensions/meshery-mcp-server).

## Week 1-2: Foundation (issues #4, #5, #6)

- Merge the repository scaffold (PR #28)
- Land the shared Meshery REST/GraphQL client as the single integration boundary
- Dual transport: stdio + streamable HTTP with graceful shutdown and session management
- CI: lint, tests, multi-platform builds

## Week 3-4: Design lifecycle tools (issue #8)

- `list_designs`, `get_design`, `create_design`, `deploy_design`, `undeploy_design`
- Structured results with typed JSON output
- Design import/export helpers

## Week 5-6: Cluster connections and registry (issues #9, #10)

- `list_clusters`, `get_cluster`, `connect_cluster`
- Component and model registry queries, schema resolution
- Context switching between multiple Meshery instances (issue #13)

## Week 7-8: Resources and prompts (issues #11, #12)

- MeshSync topology resources with subscription push (validated in this POC)
- Guided prompt templates: deployment, design review, performance testing

## Week 9-10: Environments, workspaces, performance (issues #7, #14)

- Environment and workspace management tools
- Nighthawk-backed performance testing via Meshery

## Week 11-12: Hardening and release (issues #15, #16, #17)

- Unit, integration, and end-to-end tests (issue #16)
- Release automation: multi-arch binaries, container images, changelog (issue #15)
- Documentation: user guide, configuration reference, AI client integration examples (issue #17)

## Principles (from the design document)

- Every surface registers through the shared `Registrant` seam
- The Meshery client is the only integration boundary; no tool talks to Meshery directly
- Resources are read-only; mutating actions happen through tools with explicit intent
- No credentials in logs or tool output