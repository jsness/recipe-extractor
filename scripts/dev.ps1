$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$envPath = Join-Path $root '.env'
$devComposeFile = Join-Path $root 'compose.dev.yml'
$devComposeProject = if ($env:DEV_COMPOSE_PROJECT) { $env:DEV_COMPOSE_PROJECT } else { 'recipe-extractor-local' }

function Set-EnvFromFile {
  param(
    [string]$Path
  )

  if (-not (Test-Path $Path)) {
    return
  }

  Get-Content $Path | ForEach-Object {
    if ($_ -match '^\s*#' -or $_ -match '^\s*$') {
      return
    }

    $parts = $_ -split '=', 2
    if ($parts.Length -ne 2) {
      return
    }

    $name = $parts[0].Trim()
    $value = $parts[1].Trim()

    Set-Item -Path "Env:$name" -Value $value
  }
}

function Wait-ForPostgres {
  param(
    [int]$TimeoutSeconds = 60
  )

  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)

  while ((Get-Date) -lt $deadline) {
    $status = docker compose -p $devComposeProject -f $devComposeFile ps postgres 2>$null
    if ($LASTEXITCODE -eq 0 -and ($status -match 'healthy')) {
      return
    }

    Start-Sleep -Seconds 2
  }

  throw 'Postgres did not become healthy in time.'
}

Set-EnvFromFile -Path $envPath

$devHttpAddr = if ($env:DEV_HTTP_ADDR) { $env:DEV_HTTP_ADDR } else { ':8081' }
$devVitePort = if ($env:DEV_VITE_PORT) { $env:DEV_VITE_PORT } else { '5174' }
$devPostgresPort = if ($env:DEV_POSTGRES_PORT) { $env:DEV_POSTGRES_PORT } else { '5434' }

$env:DATABASE_URL = if ($env:DEV_DATABASE_URL) {
  $env:DEV_DATABASE_URL
} else {
  "postgres://postgres:postgres@localhost:$devPostgresPort/recipes?sslmode=disable"
}
Write-Host "Using dev database at $($env:DATABASE_URL)"

$env:HTTP_ADDR = $devHttpAddr
$env:FRONTEND_DEV_PROXY_URL = "http://localhost:$devVitePort"
$env:VITE_DEV_PORT = $devVitePort
$env:VITE_API_PROXY_TARGET = "http://localhost$devHttpAddr"

Push-Location $root
try {
  Write-Host "Starting isolated dev Postgres container on localhost:$devPostgresPort..."
  docker compose -p $devComposeProject -f $devComposeFile up -d postgres

  Write-Host 'Waiting for Postgres to become healthy...'
  Wait-ForPostgres
} finally {
  Pop-Location
}

$serverCommand = @"
`$env:DATABASE_URL = '$($env:DATABASE_URL)';
`$env:HTTP_ADDR = '$($env:HTTP_ADDR)';
`$env:FRONTEND_DEV_PROXY_URL = '$($env:FRONTEND_DEV_PROXY_URL)';
Set-Location '$root\server';
go run .\cmd\server
"@

$webCommand = @"
`$env:VITE_DEV_PORT = '$($env:VITE_DEV_PORT)';
`$env:VITE_API_PROXY_TARGET = '$($env:VITE_API_PROXY_TARGET)';
Set-Location '$root\web';
npm run dev
"@

Write-Host 'Starting Go server (new window)...'
Start-Process -FilePath powershell -ArgumentList @(
  '-NoProfile',
  '-Command',
  $serverCommand
)

Write-Host 'Starting Vite dev server (new window)...'
Start-Process -FilePath powershell -ArgumentList @(
  '-NoProfile',
  '-Command',
  $webCommand
)
