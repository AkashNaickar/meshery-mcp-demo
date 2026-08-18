#!/usr/bin/env bash
# Seed a sample design into the connected Meshery Server so the demo shows data.
#
# Reads the token and provider from ~/.meshery/auth.json (written by
# `mesheryctl system login`) and POSTs a small design to /api/pattern.
#
# Usage: ./scripts/seed.sh
# Env:   MESHERY_SERVER_URL (default http://localhost:9081)
#        MESHERY_TOKEN_PATH (default ~/.meshery/auth.json)

set -euo pipefail

MESHERY_SERVER_URL="${MESHERY_SERVER_URL:-http://localhost:9081}"
MESHERY_TOKEN_PATH="${MESHERY_TOKEN_PATH:-$HOME/.meshery/auth.json}"

if [[ ! -f "$MESHERY_TOKEN_PATH" ]]; then
  echo "error: auth file not found at $MESHERY_TOKEN_PATH (run mesheryctl system login)" >&2
  exit 1
fi

TOKEN=$(jq -r '.token' "$MESHERY_TOKEN_PATH")
PROVIDER=$(jq -r '.["meshery-provider"] // "Meshery"' "$MESHERY_TOKEN_PATH")

DESIGN='{"name":"emojivoto-demo","design_file":"version: 1.0\nservices:\n  - name: emoji\n    type: Deployment\n    namespace: emojivoto\n"}'

echo "seeding design into $MESHERY_SERVER_URL ..."
curl -sS -X POST "$MESHERY_SERVER_URL/api/pattern" \
  -H "Content-Type: application/json" \
  -H "meshery-token: $TOKEN" \
  -H "Cookie: meshery-provider=$PROVIDER; token=$TOKEN" \
  -d "$DESIGN"

echo
echo "done. Run list_designs in the MCP client to see it."