# Seed a sample design into the connected Meshery Server so the demo shows data.
# Windows/PowerShell version of scripts/seed.sh.
#
# Usage: .\scripts\seed.ps1
# Env:   MESHERY_SERVER_URL (default http://localhost:9081)
#        MESHERY_TOKEN_PATH (default ~/.meshery/auth.json)

$ErrorActionPreference = "Stop"

$server = if ($env:MESHERY_SERVER_URL) { $env:MESHERY_SERVER_URL } else { "http://localhost:9081" }
$tokenPath = if ($env:MESHERY_TOKEN_PATH) { $env:MESHERY_TOKEN_PATH } else { Join-Path $HOME ".meshery\auth.json" }

if (-not (Test-Path $tokenPath)) {
  Write-Error "auth file not found at $tokenPath (run mesheryctl system login)"
}

$auth = Get-Content $tokenPath -Raw | ConvertFrom-Json
$token = $auth.token
$provider = if ($auth.'meshery-provider') { $auth.'meshery-provider' } else { "Meshery" }

$design = '{"name":"emojivoto-demo","design_file":"version: 1.0\nservices:\n  - name: emoji\n    type: Deployment\n    namespace: emojivoto\n"}'

Write-Host "seeding design into $server ..."
Invoke-RestMethod -Method Post -Uri "$server/api/pattern" `
  -ContentType "application/json" `
  -Headers @{ "meshery-token" = $token; "Cookie" = "meshery-provider=$provider; token=$token" } `
  -Body $design

Write-Host ""
Write-Host "done. Run list_designs in the MCP client to see it."