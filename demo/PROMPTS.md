# LFX Demo — Prompts & Guide

Everything you need to run the Meshery MCP Server demo.

## Prereqs (after a reboot)
```powershell
docker start meshery-meshery-1 meshery-watchtower-1
Test-NetConnection localhost -Port 9081   # expect True
kubectl get deploy,po,svc                 # nginx-demo 3/3
```

## Key ids
- Kind cluster context id: get it from `list_kubernetes_contexts` (look for the `kind-*` context). In the recorded demo this is `2e41583b9afa9112f4fa9a40d8e79f3a`, but it is environment-specific.
- Design id: get fresh from `list_designs` (or the Meshery UI Designs page)

## Screen layout (3 panes)
- **Main:** opencode agent chat
- **Top-right:** terminal `kubectl get deploy,po,svc -w`
- **Bottom-right:** Meshery UI at `:9081` (Designs page)

## opencode prompts (tools only)

1. **Reveal surface** — "List every MCP tool you have available, with a one-line description of each."
2. **server_info** — "What server am I connected to? Give me the name and version."
3. **contexts** — "List the Kubernetes contexts Meshery manages. Which one is the kind cluster?"
4. **list_designs** — "List Meshery designs, page 1 with 3 per page. Then search for any design with 'nginx'."
5. **validate (good)** — "Validate this design YAML and tell me if it's valid:" + paste `demo/good-design.yaml`
6. **validate (broken)** — "Now validate this one — it should have a problem:" + paste `demo/broken-design.yaml`
7. **deploy dry-run** — "Dry-run deploying this design to the kind-meshery-demo context and show what it would create without applying anything:" + paste good design + "(context id <kind-context-id>)"
8. **deploy apply** — "Now run the real deploy for the same design. Show what the server reports, including any note about whether it applied."
9. **get resources** — "Show me the live resources on the kind cluster — deployments, pods, services. (context id <kind-context-id>)"
10. **undeploy** — "Undeploy this design from the kind cluster: <pattern_id> (context id <kind-context-id>)"

## MCP Inspector beats (resources, prompts, live-push)

Resources, prompts, and subscription are NOT exposed as tools in opencode, so
run these in the MCP Inspector:
```
npx -y @modelcontextprotocol/inspector "D:\Projekt\meshery-mcp-demo\bin\meshery-mcp-demo.exe"
```

- **Topology resource:** Resources tab → read `meshery://clusters/<kind-context-id>/topology` (clear the field, paste the full URI once)
- **Design resource:** Resources tab → read `meshery://designs/<id>`
- **Prompts:** Prompts tab → `deploy_application`, `cluster_health_audit`
- **Live-push:** Resources tab → subscribe to topology → in terminal `kubectl scale deployment nginx-demo --replicas=4` → watch `notifications/resources/updated` show 4 pods → scale back `--replicas=3`

## Deploy honesty rule
- `dry_run: true` shows the plan (works).
- `dry_run: false` returns the plan + an honest no-op note on Meshery v1.0.66.
- Do NOT claim MCP created the deployment. The app is already running; prove the
  agent SEES it (get_cluster_resources + kubectl), not that it created it.

## Sample designs
- `demo/good-design.yaml` — valid PatternFile → `valid: true`
- `demo/broken-design.yaml` — missing a component type → `valid: false` with an error
