$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$envPath = Join-Path $root '.env'

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
    $status = docker compose ps postgres 2>$null
    if ($LASTEXITCODE -eq 0 -and ($status -match 'healthy')) {
      return
    }

    Start-Sleep -Seconds 2
  }

  throw 'Postgres did not become healthy in time.'
}

Set-EnvFromFile -Path $envPath

if (-not $env:DATABASE_URL) {
  $env:DATABASE_URL = 'postgres://postgres:postgres@localhost:5433/recipes?sslmode=disable'
  Write-Host 'DATABASE_URL not set, defaulting to localhost:5433/recipes'
}

$env:FRONTEND_DEV_PROXY_URL = 'http://localhost:5173'

Push-Location $root
try {
  Write-Host 'Stopping app container if it is already running...'
  docker compose stop app | Out-Null

  Write-Host 'Starting Postgres container...'
  docker compose up -d postgres

  Write-Host 'Waiting for Postgres to become healthy...'
  Wait-ForPostgres
} finally {
  Pop-Location
}

$serverCommand = @"
`$env:FRONTEND_DEV_PROXY_URL = '$($env:FRONTEND_DEV_PROXY_URL)';
Set-Location '$root\server';
go run .\cmd\server
"@

$webCommand = @"
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
