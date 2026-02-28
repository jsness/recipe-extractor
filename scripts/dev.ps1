$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$envPath = Join-Path $root '.env'

if (Test-Path $envPath) {
  Get-Content $envPath | ForEach-Object {
    if ($_ -match '^\s*#' -or $_ -match '^\s*$') { return }
    $parts = $_ -split '=', 2
    if ($parts.Length -eq 2) {
      $name = $parts[0].Trim()
      $value = $parts[1].Trim()
      [System.Environment]::SetEnvironmentVariable($name, $value)
    }
  }
}

if (-not $env:DATABASE_URL) {
  $env:DATABASE_URL = 'postgres://postgres:postgres@localhost:5433/recipes?sslmode=disable'
  Write-Host 'DATABASE_URL not set, defaulting to localhost:5433/recipes'
}

Write-Host 'Starting Postgres...'
Push-Location $root
try {
  docker compose up -d
} finally {
  Pop-Location
}

Write-Host 'Running migrations...'
Push-Location (Join-Path $root 'server')
try {
  go run github.com/jsness/go-migrate-lite/cmd/migrate-lite
} finally {
  Pop-Location
}

Write-Host 'Starting server (new window)...'
Start-Process -FilePath powershell -ArgumentList @(
  '-NoProfile',
  '-Command',
  "Set-Location '$root\\server'; go run .\\cmd\\server"
)

Write-Host 'Starting web app (new window)...'
Start-Process -FilePath powershell -ArgumentList @(
  '-NoProfile',
  '-Command',
  "Set-Location '$root\\web'; npm run dev"
)