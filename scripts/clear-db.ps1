param(
  [switch]$Force
)

$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot

if (-not $Force) {
  $confirmation = Read-Host "This will delete all data in the local Postgres database. Type 'yes' to continue"
  if ($confirmation -ne 'yes') {
    Write-Host 'Aborted.'
    exit 0
  }
}

Push-Location $root
try {
  docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U postgres -d recipes -c "DROP SCHEMA IF EXISTS public CASCADE;"
  docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U postgres -d recipes -c "CREATE SCHEMA public;"
  docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U postgres -d recipes -c "GRANT ALL ON SCHEMA public TO postgres;"
  docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U postgres -d recipes -c "GRANT ALL ON SCHEMA public TO public;"

  Write-Host 'Database cleared.'
  Write-Host 'Re-run migrations with:'
  Write-Host '  cd server'
  Write-Host '  go run github.com/jsness/go-migrate-lite/cmd/migrate-lite'
} finally {
  Pop-Location
}
